// Package services 提供业务服务层
//
// MQTTService 处理与 MQTT 相关的业务逻辑，包括：
// - 设备状态管理
// - 命令下发
// - 事件处理
// - 遥测数据收集
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/mqtt"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// MQTTService MQTT 服务层
//
// 负责管理 MQTT 连接、消息处理和设备状态同步
type MQTTService struct {
	client     *mqtt.Client
	tabletRepo repositories.TabletRepository
	reportRepo repositories.ReportRepository

	// 命令响应管理
	pendingCommands map[string]chan *models.CommandResult
	mu              sync.RWMutex

	// 配置
	tenantID string
}

// MQTTServiceConfig MQTT 服务配置
type MQTTServiceConfig struct {
	BrokerURL  string
	Port       int
	ClientID   string
	Username   string
	Password   string
	UseTLS     bool
	KeepAlive  time.Duration
	CleanStart bool
	TenantID   string
}

// NewMQTTService 创建 MQTT 服务
func NewMQTTService(
	tabletRepo repositories.TabletRepository,
	reportRepo repositories.ReportRepository,
	cfg *MQTTServiceConfig,
) *MQTTService {
	// 创建 MQTT 客户端配置
	mqttConfig := &mqtt.Config{
		BrokerURL:     cfg.BrokerURL,
		Port:          cfg.Port,
		ClientID:      cfg.ClientID,
		Username:      cfg.Username,
		Password:      cfg.Password,
		UseTLS:        cfg.UseTLS,
		KeepAlive:     cfg.KeepAlive,
		CleanStart:    cfg.CleanStart,
		AutoReconnect: true,
	}

	return &MQTTService{
		client:          mqtt.NewClient(mqttConfig),
		tabletRepo:      tabletRepo,
		reportRepo:      reportRepo,
		pendingCommands: make(map[string]chan *models.CommandResult),
		tenantID:        cfg.TenantID,
	}
}

// Connect 连接到 MQTT Broker
func (s *MQTTService) Connect(ctx context.Context) error {
	if err := s.client.Connect(ctx); err != nil {
		return fmt.Errorf("MQTT 连接失败: %w", err)
	}

	// 订阅设备状态主题
	statusTopic := mqtt.StatusWildcard(s.tenantID)
	if err := s.client.Subscribe(ctx, statusTopic, s.handleStatusMessage); err != nil {
		slog.Error("订阅状态主题失败", "topic", statusTopic, "error", err)
	}

	// 订阅设备事件主题
	eventTopic := mqtt.EventWildcard(s.tenantID)
	if err := s.client.Subscribe(ctx, eventTopic, s.handleEventMessage); err != nil {
		slog.Error("订阅事件主题失败", "topic", eventTopic, "error", err)
	}

	// 订阅设备遥测主题
	telemetryTopic := mqtt.TelemetryWildcard(s.tenantID)
	if err := s.client.Subscribe(ctx, telemetryTopic, s.handleTelemetryMessage); err != nil {
		slog.Error("订阅遥测主题失败", "topic", telemetryTopic, "error", err)
	}

	// 订阅命令响应主题（使用通配符）
	responseTopic := fmt.Sprintf("kiosk/%s/+response/+", s.tenantID)
	if err := s.client.Subscribe(ctx, responseTopic, s.handleCommandResponse); err != nil {
		slog.Error("订阅响应主题失败", "topic", responseTopic, "error", err)
	}

	slog.Info("✅ MQTT 服务已连接并订阅所有主题")
	return nil
}

// Disconnect 断开 MQTT 连接
func (s *MQTTService) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// IsConnected 检查是否已连接
func (s *MQTTService) IsConnected() bool {
	return s.client.IsConnected()
}

// Publish publishes a message to a topic without waiting for response
func (s *MQTTService) Publish(topic string, payload []byte) error {
	ctx := context.Background()
	return s.client.Publish(ctx, topic, payload)
}

// SendCommand 发送命令到设备
//
// 发送命令并等待设备响应
func (s *MQTTService) SendCommand(ctx context.Context, deviceID string, cmdType models.CommandType, params map[string]interface{}, timeout time.Duration) (*models.CommandResult, error) {
	// 生成命令 ID
	commandID := uuid.New().String()

	// 创建命令消息
	cmd := &models.Command{
		ID:        commandID,
		Type:      cmdType,
		Timestamp: time.Now(),
		Params:    params,
		Timeout:   int(timeout.Seconds()),
	}

	// 序列化命令
	payload, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("序列化命令失败: %w", err)
	}

	// 创建响应通道
	respChan := make(chan *models.CommandResult, 1)
	s.mu.Lock()
	s.pendingCommands[commandID] = respChan
	s.mu.Unlock()

	// 清理
	defer func() {
		s.mu.Lock()
		delete(s.pendingCommands, commandID)
		s.mu.Unlock()
		close(respChan)
	}()

	// 发布命令
	topicBuilder := mqtt.NewTopicBuilder(s.tenantID, deviceID)
	if err := s.client.Publish(ctx, topicBuilder.CommandTopic(), payload); err != nil {
		return nil, fmt.Errorf("发布命令失败: %w", err)
	}

	slog.Info("📤 命令已发送",
		"commandId", commandID,
		"deviceId", deviceID,
		"type", cmdType,
	)

	// 等待响应
	select {
	case result := <-respChan:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("命令超时")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendCommandAsync 异步发送命令（不等待响应）
func (s *MQTTService) SendCommandAsync(ctx context.Context, deviceID string, cmdType models.CommandType, params map[string]interface{}) (string, error) {
	// 生成命令 ID
	commandID := uuid.New().String()

	// 创建命令消息
	cmd := &models.Command{
		ID:        commandID,
		Type:      cmdType,
		Timestamp: time.Now(),
		Params:    params,
		Timeout:   30, // 默认 30 秒超时
	}

	// 序列化命令
	payload, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("序列化命令失败: %w", err)
	}

	// 发布命令
	topicBuilder := mqtt.NewTopicBuilder(s.tenantID, deviceID)
	if err := s.client.Publish(ctx, topicBuilder.CommandTopic(), payload); err != nil {
		return "", fmt.Errorf("发布命令失败: %w", err)
	}

	slog.Info("📤 异步命令已发送",
		"commandId", commandID,
		"deviceId", deviceID,
		"type", cmdType,
	)

	return commandID, nil
}

// handleStatusMessage 处理设备状态消息
func (s *MQTTService) handleStatusMessage(topic string, payload []byte) error {
	var status models.DeviceStatusInfo
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("解析状态消息失败: %w", err)
	}

	slog.Debug("📥 收到设备状态",
		"deviceId", status.DeviceID,
		"battery", status.BatteryLevel,
		"screenOn", status.ScreenOn,
	)

	// TODO: 更新数据库中的设备状态
	// 可以通过 tabletRepo 更新设备信息

	return nil
}

// handleEventMessage 处理设备事件消息
func (s *MQTTService) handleEventMessage(topic string, payload []byte) error {
	var event models.DeviceEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("解析事件消息失败: %w", err)
	}

	slog.Info("📥 收到设备事件",
		"deviceId", event.DeviceID,
		"type", event.Type,
	)

	// 根据事件类型处理
	switch event.Type {
	case "user_interaction":
		// 用户交互事件
	case "error":
		// 错误事件
		slog.Error("设备错误事件",
			"deviceId", event.DeviceID,
			"data", event.Data,
		)
	case "security":
		// 安全事件
		slog.Warn("设备安全事件",
			"deviceId", event.DeviceID,
			"data", event.Data,
		)
	}

	return nil
}

// handleTelemetryMessage 处理设备遥测消息
func (s *MQTTService) handleTelemetryMessage(topic string, payload []byte) error {
	var telemetry models.DeviceTelemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return fmt.Errorf("解析遥测消息失败: %w", err)
	}

	slog.Debug("📥 收到设备遥测",
		"deviceId", telemetry.DeviceID,
		"cpu", telemetry.CPUUsage,
		"memory", telemetry.MemoryUsage,
		"temperature", telemetry.Temperature,
	)

	// TODO: 存储遥测数据到时序数据库
	// 可以使用 TimescaleDB 存储历史数据

	return nil
}

// handleCommandResponse 处理命令响应
func (s *MQTTService) handleCommandResponse(topic string, payload []byte) error {
	var result models.CommandResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("解析命令响应失败: %w", err)
	}

	slog.Info("📥 收到命令响应",
		"commandId", result.CommandID,
		"success", result.Success,
	)

	// 查找并通知等待的命令
	s.mu.RLock()
	ch, ok := s.pendingCommands[result.CommandID]
	s.mu.RUnlock()

	if ok {
		select {
		case ch <- &result:
		default:
			slog.Warn("命令响应通道已满", "commandId", result.CommandID)
		}
	}

	return nil
}