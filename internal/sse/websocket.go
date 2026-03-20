package sse

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// WebSocketMessage WebSocket 消息格式
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// DeviceStatusUpdate 设备状态更新消息
type DeviceStatusUpdate struct {
	DeviceID string                 `json:"device_id"`
	Online   bool                   `json:"online"`
	Status   map[string]interface{} `json:"status,omitempty"`
}

// CommandResultUpdate 命令结果更新消息
type CommandResultUpdate struct {
	CommandID string               `json:"command_id"`
	DeviceID  string               `json:"device_id"`
	Success   bool                 `json:"success"`
	Result    *models.CommandResult `json:"result,omitempty"`
}

// AlertUpdate 告警更新消息
type AlertUpdate struct {
	AlertID   string                 `json:"alert_id"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// WebSocketHub WebSocket 连接管理中心
type WebSocketHub struct {
	// 连接管理
	connections map[*Connection]bool
	register    chan *Connection
	unregister  chan *Connection

	// 消息通道
	broadcast       chan *WebSocketMessage
	deviceChannels  map[string]map[*Connection]bool // device_id -> connections
	tenantChannels  map[string]map[*Connection]bool // tenant_id -> connections

	mu sync.RWMutex
}

// Connection WebSocket 连接
type Connection struct {
	ID        string
	TenantID  string
	UserID    string
	Send      chan *WebSocketMessage
	Channels  map[string]bool // 订阅的频道
}

// NewWebSocketHub 创建 WebSocket Hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		connections:    make(map[*Connection]bool),
		register:       make(chan *Connection, 256),
		unregister:     make(chan *Connection, 256),
		broadcast:      make(chan *WebSocketMessage, 1024),
		deviceChannels: make(map[string]map[*Connection]bool),
		tenantChannels: make(map[string]map[*Connection]bool),
	}
}

// Run 启动 WebSocketHub
func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn] = true
			h.mu.Unlock()
			slog.Debug("WebSocket connected", "connId", conn.ID, "total", len(h.connections))

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.Send)

				// 从所有频道移除
				for channel := range conn.Channels {
					if strings.HasPrefix(channel, "device:") {
						deviceID := strings.TrimPrefix(channel, "device:")
						if conns, ok := h.deviceChannels[deviceID]; ok {
							delete(conns, conn)
						}
					} else if strings.HasPrefix(channel, "tenant:") {
						tenantID := strings.TrimPrefix(channel, "tenant:")
						if conns, ok := h.tenantChannels[tenantID]; ok {
							delete(conns, conn)
						}
					}
				}
			}
			h.mu.Unlock()
			slog.Debug("WebSocket disconnected", "connId", conn.ID, "total", len(h.connections))

		case msg := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.connections {
				select {
				case conn.Send <- msg:
				default:
					// 通道已满，跳过
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册连接
func (h *WebSocketHub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister 注销连接
func (h *WebSocketHub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// Subscribe 订阅频道
func (h *WebSocketHub) Subscribe(conn *Connection, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn.Channels[channel] = true

	if strings.HasPrefix(channel, "device:") {
		deviceID := strings.TrimPrefix(channel, "device:")
		if h.deviceChannels[deviceID] == nil {
			h.deviceChannels[deviceID] = make(map[*Connection]bool)
		}
		h.deviceChannels[deviceID][conn] = true
	} else if strings.HasPrefix(channel, "tenant:") {
		tenantID := strings.TrimPrefix(channel, "tenant:")
		if h.tenantChannels[tenantID] == nil {
			h.tenantChannels[tenantID] = make(map[*Connection]bool)
		}
		h.tenantChannels[tenantID][conn] = true
	}
}

// Unsubscribe 取消订阅
func (h *WebSocketHub) Unsubscribe(conn *Connection, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(conn.Channels, channel)

	if strings.HasPrefix(channel, "device:") {
		deviceID := strings.TrimPrefix(channel, "device:")
		if conns, ok := h.deviceChannels[deviceID]; ok {
			delete(conns, conn)
		}
	} else if strings.HasPrefix(channel, "tenant:") {
		tenantID := strings.TrimPrefix(channel, "tenant:")
		if conns, ok := h.tenantChannels[tenantID]; ok {
			delete(conns, conn)
		}
	}
}

// Broadcast 广播消息
func (h *WebSocketHub) Broadcast(msg *WebSocketMessage) {
	h.broadcast <- msg
}

// BroadcastToDevice 向设备订阅者广播
func (h *WebSocketHub) BroadcastToDevice(deviceID string, msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if conns, ok := h.deviceChannels[deviceID]; ok {
		for conn := range conns {
			select {
			case conn.Send <- msg:
			default:
			}
		}
	}
}

// BroadcastToTenant 向租户订阅者广播
func (h *WebSocketHub) BroadcastToTenant(tenantID string, msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if conns, ok := h.tenantChannels[tenantID]; ok {
		for conn := range conns {
			select {
			case conn.Send <- msg:
			default:
			}
		}
	}
}

// NotifyDeviceStatus 通知设备状态变更
func (h *WebSocketHub) NotifyDeviceStatus(tenantID, deviceID string, update *DeviceStatusUpdate) {
	msg := &WebSocketMessage{
		Type:    "device_status",
		Channel: "device:" + deviceID,
		Data:    update,
		Timestamp: time.Now(),
	}

	// 发送给设备订阅者
	h.BroadcastToDevice(deviceID, msg)

	// 发送给租户订阅者
	msg.Channel = "tenant:" + tenantID
	h.BroadcastToTenant(tenantID, msg)
}

// NotifyCommandResult 通知命令结果
func (h *WebSocketHub) NotifyCommandResult(tenantID, deviceID string, update *CommandResultUpdate) {
	msg := &WebSocketMessage{
		Type:    "command_result",
		Channel: "device:" + deviceID,
		Data:    update,
		Timestamp: time.Now(),
	}

	h.BroadcastToDevice(deviceID, msg)
	msg.Channel = "tenant:" + tenantID
	h.BroadcastToTenant(tenantID, msg)
}

// NotifyAlert 通知告警
func (h *WebSocketHub) NotifyAlert(tenantID string, alert *AlertUpdate) {
	msg := &WebSocketMessage{
		Type:    "alert",
		Channel: "tenant:" + tenantID,
		Data:    alert,
		Timestamp: time.Now(),
	}

	h.BroadcastToTenant(tenantID, msg)
}

// GetConnectionCount 获取连接数
func (h *WebSocketHub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
