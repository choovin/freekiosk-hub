package mqtt

import "fmt"

// Topic 模式定义
//
// MQTT Topic 命名规范:
//   - kiosk/{tenant_id}/{device_id}/status    - 设备状态（保留消息）
//   - kiosk/{tenant_id}/{device_id}/command   - 服务器下发命令
//   - kiosk/{tenant_id}/{device_id}/config    - 配置更新
//   - kiosk/{tenant_id}/{device_id}/event     - 设备事件
//   - kiosk/{tenant_id}/{device_id}/telemetry - 遥测数据
//   - kiosk/{tenant_id}/{device_id}/response/{command_id} - 命令响应
const (
	// 设备 -> 服务器 Topic
	TopicStatus    = "kiosk/%s/%s/status"    // 租户ID, 设备ID
	TopicEvent     = "kiosk/%s/%s/event"      // 租户ID, 设备ID
	TopicTelemetry = "kiosk/%s/%s/telemetry"  // 租户ID, 设备ID
	TopicResponse  = "kiosk/%s/%s/response/%s" // 租户ID, 设备ID, 命令ID

	// 服务器 -> 设备 Topic
	TopicCommand  = "kiosk/%s/%s/command"  // 租户ID, 设备ID
	TopicConfig   = "kiosk/%s/%s/config"   // 租户ID, 设备ID
	TopicFirmware = "kiosk/%s/%s/firmware" // 租户ID, 设备ID

	// 广播 Topic（租户级别）
	TopicBroadcastCommand = "kiosk/%s/broadcast/command" // 租户ID
	TopicBroadcastConfig  = "kiosk/%s/broadcast/config"  // 租户ID

	// 共享订阅模式（用于负载均衡）
	SharedSubscriptionPattern = "$share/%s/kiosk/+/+/command"
)

// TopicBuilder Topic 构建器
//
// 用于方便地构建特定设备的 Topic
type TopicBuilder struct {
	tenantID string
	deviceID string
}

// NewTopicBuilder 创建 Topic 构建器
func NewTopicBuilder(tenantID, deviceID string) *TopicBuilder {
	return &TopicBuilder{
		tenantID: tenantID,
		deviceID: deviceID,
	}
}

// StatusTopic 返回状态 Topic
func (tb *TopicBuilder) StatusTopic() string {
	return fmt.Sprintf(TopicStatus, tb.tenantID, tb.deviceID)
}

// CommandTopic 返回命令 Topic
func (tb *TopicBuilder) CommandTopic() string {
	return fmt.Sprintf(TopicCommand, tb.tenantID, tb.deviceID)
}

// ConfigTopic 返回配置 Topic
func (tb *TopicBuilder) ConfigTopic() string {
	return fmt.Sprintf(TopicConfig, tb.tenantID, tb.deviceID)
}

// EventTopic 返回事件 Topic
func (tb *TopicBuilder) EventTopic() string {
	return fmt.Sprintf(TopicEvent, tb.tenantID, tb.deviceID)
}

// TelemetryTopic 返回遥测 Topic
func (tb *TopicBuilder) TelemetryTopic() string {
	return fmt.Sprintf(TopicTelemetry, tb.tenantID, tb.deviceID)
}

// ResponseTopic 返回响应 Topic
func (tb *TopicBuilder) ResponseTopic(commandID string) string {
	return fmt.Sprintf(TopicResponse, tb.tenantID, tb.deviceID, commandID)
}

// FirmwareTopic 返回固件 Topic
func (tb *TopicBuilder) FirmwareTopic() string {
	return fmt.Sprintf(TopicFirmware, tb.tenantID, tb.deviceID)
}

// BroadcastCommandTopic 返回广播命令 Topic
func BroadcastCommandTopic(tenantID string) string {
	return fmt.Sprintf(TopicBroadcastCommand, tenantID)
}

// BroadcastConfigTopic 返回广播配置 Topic
func BroadcastConfigTopic(tenantID string) string {
	return fmt.Sprintf(TopicBroadcastConfig, tenantID)
}

// SharedCommandSubscription 返回共享订阅 Topic
func SharedCommandSubscription(group string) string {
	return fmt.Sprintf(SharedSubscriptionPattern, group)
}

// StatusWildcard 返回状态通配符 Topic（订阅所有设备状态）
func StatusWildcard(tenantID string) string {
	return fmt.Sprintf("kiosk/%s/+/status", tenantID)
}

// EventWildcard 返回事件通配符 Topic（订阅所有设备事件）
func EventWildcard(tenantID string) string {
	return fmt.Sprintf("kiosk/%s/+/event", tenantID)
}

// TelemetryWildcard 返回遥测通配符 Topic（订阅所有设备遥测）
func TelemetryWildcard(tenantID string) string {
	return fmt.Sprintf("kiosk/%s/+/telemetry", tenantID)
}