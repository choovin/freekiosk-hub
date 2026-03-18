package mqtt

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// DeviceStatusHandler 设备状态消息处理器
type DeviceStatusHandler struct {
	// 状态消息通道
	statusChan chan<- *models.DeviceStatus
}

// NewDeviceStatusHandler 创建状态处理器
func NewDeviceStatusHandler(statusChan chan<- *models.DeviceStatus) *DeviceStatusHandler {
	return &DeviceStatusHandler{statusChan: statusChan}
}

// Handle 处理状态消息
func (h *DeviceStatusHandler) Handle(topic string, payload []byte) error {
	var status models.DeviceStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("解析状态消息失败: %w", err)
	}

	// 从 Topic 提取设备信息
	// Topic 格式: kiosk/{tenant_id}/{device_id}/status
	// TODO: 解析并设置 TenantID 和 DeviceID

	select {
	case h.statusChan <- &status:
		log.Printf("[MQTT] 收到设备状态更新")
	default:
		log.Printf("[MQTT] 状态通道已满，丢弃消息")
	}

	return nil
}

// DeviceEventHandler 设备事件消息处理器
type DeviceEventHandler struct {
	// 事件消息通道
	eventChan chan<- *models.DeviceEvent
}

// NewDeviceEventHandler 创建事件处理器
func NewDeviceEventHandler(eventChan chan<- *models.DeviceEvent) *DeviceEventHandler {
	return &DeviceEventHandler{eventChan: eventChan}
}

// Handle 处理事件消息
func (h *DeviceEventHandler) Handle(topic string, payload []byte) error {
	var event models.DeviceEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("解析事件消息失败: %w", err)
	}

	select {
	case h.eventChan <- &event:
		log.Printf("[MQTT] 收到设备事件: %s", event.Type)
	default:
		log.Printf("[MQTT] 事件通道已满，丢弃消息")
	}

	return nil
}

// DeviceTelemetryHandler 设备遥测数据处理器
type DeviceTelemetryHandler struct {
	// 遥测数据通道
	telemetryChan chan<- *models.DeviceTelemetry
}

// NewDeviceTelemetryHandler 创建遥测处理器
func NewDeviceTelemetryHandler(telemetryChan chan<- *models.DeviceTelemetry) *DeviceTelemetryHandler {
	return &DeviceTelemetryHandler{telemetryChan: telemetryChan}
}

// Handle 处理遥测消息
func (h *DeviceTelemetryHandler) Handle(topic string, payload []byte) error {
	var telemetry models.DeviceTelemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return fmt.Errorf("解析遥测消息失败: %w", err)
	}

	select {
	case h.telemetryChan <- &telemetry:
		log.Printf("[MQTT] 收到遥测数据")
	default:
		// 遥测数据频率高，丢弃是正常的
	}

	return nil
}

// CommandResponseHandler 命令响应处理器
type CommandResponseHandler struct {
	// 响应通道映射（命令ID -> 响应通道）
	responseChans map[string]chan *models.CommandResult
}

// NewCommandResponseHandler 创建命令响应处理器
func NewCommandResponseHandler() *CommandResponseHandler {
	return &CommandResponseHandler{
		responseChans: make(map[string]chan *models.CommandResult),
	}
}

// Handle 处理命令响应
func (h *CommandResponseHandler) Handle(topic string, payload []byte) error {
	var result models.CommandResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("解析命令响应失败: %w", err)
	}

	// 查找对应的响应通道
	if ch, ok := h.responseChans[result.CommandID]; ok {
		select {
		case ch <- &result:
		default:
			log.Printf("[MQTT] 响应通道已满: %s", result.CommandID)
		}
	}

	return nil
}

// Register 注册命令响应通道
func (h *CommandResponseHandler) Register(commandID string, ch chan *models.CommandResult) {
	h.responseChans[commandID] = ch
}

// Unregister 注销命令响应通道
func (h *CommandResponseHandler) Unregister(commandID string) {
	delete(h.responseChans, commandID)
}