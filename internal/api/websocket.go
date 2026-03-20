package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/sse"
)

// WebSocketHandler WebSocket 处理器
type WebSocketHandler struct {
	hub *sse.WebSocketHub
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(hub *sse.WebSocketHub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 生产环境需要验证 Origin
		return true
	},
}

// HandleWebSocket 处理 WebSocket 连接
// GET /api/v2/ws
func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return err
	}
	defer conn.Close()

	// 创建连接
	connection := &sse.Connection{
		ID:       generateConnID(),
		TenantID: c.QueryParam("tenant_id"),
		UserID:   c.QueryParam("user_id"),
		Send:     make(chan *sse.WebSocketMessage, 256),
		Channels: make(map[string]bool),
	}

	// 注册连接
	h.hub.Register(connection)
	defer h.hub.Unregister(connection)

	// 发送欢迎消息
	welcome := &sse.WebSocketMessage{
		Type:      "connected",
		Data:      map[string]string{"connection_id": connection.ID},
		Timestamp: time.Now(),
	}
	conn.WriteJSON(welcome)

	// 启动写入协程
	go func() {
		for msg := range connection.Send {
			if err := conn.WriteJSON(msg); err != nil {
				slog.Debug("WebSocket write error", "error", err)
				return
			}
		}
	}()

	// 读取消息循环
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Debug("WebSocket read error", "error", err)
			}
			break
		}

		// 解析消息
		var msg struct {
			Type    string   `json:"type"`
			Channels []string `json:"channels,omitempty"`
			Data    interface{} `json:"data,omitempty"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Debug("Failed to parse WebSocket message", "error", err)
			continue
		}

		// 处理消息
		switch msg.Type {
		case "subscribe":
			// 订阅频道
			for _, channel := range msg.Channels {
				h.hub.Subscribe(connection, channel)
			}
			conn.WriteJSON(&sse.WebSocketMessage{
				Type:      "subscribed",
				Data:      map[string][]string{"channels": msg.Channels},
				Timestamp: time.Now(),
			})

		case "unsubscribe":
			// 取消订阅
			for _, channel := range msg.Channels {
				h.hub.Unsubscribe(connection, channel)
			}
			conn.WriteJSON(&sse.WebSocketMessage{
				Type:      "unsubscribed",
				Data:      map[string][]string{"channels": msg.Channels},
				Timestamp: time.Now(),
			})

		case "ping":
			// 心跳
			conn.WriteJSON(&sse.WebSocketMessage{
				Type:      "pong",
				Timestamp: time.Now(),
			})

		default:
			slog.Debug("Unknown WebSocket message type", "type", msg.Type)
		}
	}

	return nil
}

// HandleGetConnectionCount 获取连接数
// GET /api/v2/ws/connections
func (h *WebSocketHandler) HandleGetConnectionCount(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"connections": h.hub.GetConnectionCount(),
		"timestamp":   time.Now(),
	})
}

func generateConnID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
