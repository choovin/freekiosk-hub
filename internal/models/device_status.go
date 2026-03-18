package models

import "time"

// DeviceStatus 设备实时状态
//
// 通过 MQTT Topic: kiosk/{tenant_id}/{device_id}/status 发布
// 使用保留消息确保新订阅者能立即获取最新状态
type DeviceStatus struct {
	// 设备标识
	DeviceID string    `json:"deviceId"`
	TenantID string    `json:"tenantId"`
	UpdatedAt time.Time `json:"timestamp"`

	// 电池状态
	BatteryLevel    int  `json:"batteryLevel"`    // 电池电量 (0-100)
	BatteryCharging bool `json:"batteryCharging"` // 是否充电中

	// 屏幕状态
	ScreenOn         bool `json:"screenOn"`         // 屏幕是否开启
	ScreenBrightness int  `json:"screenBrightness"` // 屏幕亮度 (0-255)

	// 音频状态
	Volume int `json:"volume"` // 音量 (0-100)

	// 网络状态
	WifiSSID     string `json:"wifiSsid"`     // WiFi SSID
	WifiStrength int    `json:"wifiStrength"` // WiFi 信号强度 (0-100)
	IPAddress    string `json:"ipAddress"`    // IP 地址

	// WebView 状态
	CurrentURL string `json:"currentUrl"` // 当前 URL
	Loading    bool   `json:"loading"`    // 是否正在加载

	// 存储状态
	StorageUsedMB  int64 `json:"storageUsedMb"`  // 已用存储 (MB)
	StorageTotalMB int64 `json:"storageTotalMb"` // 总存储 (MB)

	// 应用状态
	AppVersion string `json:"appVersion"` // 应用版本
	Uptime     int64  `json:"uptime"`     // 运行时间 (秒)
}

// DeviceEvent 设备事件
//
// 通过 MQTT Topic: kiosk/{tenant_id}/{device_id}/event 发布
// 用于记录用户交互、错误等离散事件
type DeviceEvent struct {
	// 事件标识
	DeviceID  string    `json:"deviceId"`
	TenantID  string    `json:"tenantId"`
	Type      string    `json:"type"`      // 事件类型
	Timestamp time.Time `json:"timestamp"` // 事件时间

	// 事件数据
	Data map[string]interface{} `json:"data"`
}

// DeviceTelemetry 设备遥测数据
//
// 通过 MQTT Topic: kiosk/{tenant_id}/{device_id}/telemetry 发布
// 高频遥测数据，使用 QoS 0 发送
type DeviceTelemetry struct {
	DeviceID  string    `json:"deviceId"`
	TenantID  string    `json:"tenantId"`
	Timestamp time.Time `json:"timestamp"`

	// 性能指标
	CPUUsage     float64 `json:"cpuUsage"`     // CPU 使用率 (0-100)
	MemoryUsage  float64 `json:"memoryUsage"`  // 内存使用率 (0-100)
	NetworkTX    int64   `json:"networkTx"`    // 网络发送量 (bytes)
	NetworkRX    int64   `json:"networkRx"`    // 网络接收量 (bytes)
	Temperature  float64 `json:"temperature"`  // 设备温度 (℃)
}

// CommandType 命令类型
type CommandType string

const (
	// 屏幕控制命令
	CommandSetBrightness CommandType = "setBrightness" // 设置亮度
	CommandSetScreen     CommandType = "setScreen"     // 开关屏幕

	// 音频控制命令
	CommandSetVolume CommandType = "setVolume" // 设置音量

	// WebView 控制命令
	CommandNavigate CommandType = "navigate" // 导航到 URL
	CommandReload   CommandType = "reload"   // 刷新页面

	// 系统控制命令
	CommandReboot    CommandType = "reboot"    // 重启设备
	CommandClearCache CommandType = "clearCache" // 清除缓存

	// 应用控制命令
	CommandUpdateApp CommandType = "updateApp" // 更新应用
)

// Command 下发给设备的命令
type Command struct {
	// 命令标识
	ID        string      `json:"id"`
	Type      CommandType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`

	// 命令参数
	Params map[string]interface{} `json:"params"`

	// 超时设置
	Timeout int `json:"timeout"` // 超时时间 (秒)
}

// CommandResult 命令执行结果
type CommandResult struct {
	CommandID string      `json:"commandId"` // 命令 ID
	Success   bool        `json:"success"`   // 是否成功
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}