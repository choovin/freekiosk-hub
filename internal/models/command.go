package models

import (
	"time"
)

// CommandType 命令类型
type CommandType string

const (
	CommandSetBrightness  CommandType = "setBrightness"
	CommandSetScreen      CommandType = "setScreen"
	CommandSetVolume      CommandType = "setVolume"
	CommandNavigate       CommandType = "navigate"
	CommandReload         CommandType = "reload"
	CommandReboot         CommandType = "reboot"
	CommandClearCache     CommandType = "clearCache"
	CommandUpdateApp      CommandType = "updateApp"
	CommandScreenshot     CommandType = "screenshot"
	CommandGetLogs        CommandType = "getLogs"
	CommandGetWifiInfo    CommandType = "getWifiInfo"
	CommandInstallApp     CommandType = "installApp"
	CommandUninstallApp   CommandType = "uninstallApp"
	CommandSetRotation    CommandType = "setRotation"
	CommandSetKioskMode   CommandType = "setKioskMode"
	CommandPlaySound      CommandType = "playSound"
	CommandStopSound      CommandType = "stopSound"
	CommandSpeak          CommandType = "speak"
	CommandWakeUp         CommandType = "wakeUp"
	CommandSleep          CommandType = "sleep"
)

// Command 表示发送给设备的命令
type Command struct {
	ID        string                 `json:"id"`
	Type      CommandType            `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"` // 秒
	Priority  int                    `json:"priority,omitempty"`
}

// CommandResult 表示命令执行结果
type CommandResult struct {
	CommandID   string                 `json:"commandId"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    int64                  `json:"duration,omitempty"` // 毫秒
}

// CommandTarget 表示命令目标
type CommandTarget struct {
	DeviceIDs []string `json:"device_ids,omitempty"`
	GroupIDs  []string `json:"group_ids,omitempty"`
	All       bool     `json:"all,omitempty"`
}

// BatchCommandResult 批量命令结果
type BatchCommandResult struct {
	BatchID    string                   `json:"batchId"`
	TotalCount int                      `json:"totalCount"`
	SuccessCount int                    `json:"successCount"`
	FailedCount  int                    `json:"failedCount"`
	Results    []DeviceCommandResult    `json:"results"`
	CreatedAt  time.Time                `json:"createdAt"`
}

// DeviceCommandResult 单个设备的命令结果
type DeviceCommandResult struct {
	DeviceID string         `json:"deviceId"`
	Success  bool           `json:"success"`
	Result   *CommandResult `json:"result,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// CommandRecord 命令历史记录
type CommandRecord struct {
	ID           string                 `json:"id" db:"id"`
	TenantID     string                 `json:"tenant_id" db:"tenant_id"`
	DeviceID     string                 `json:"device_id" db:"device_id"`
	CommandType  CommandType            `json:"command_type" db:"command_type"`
	CommandID    string                 `json:"command_id" db:"command_id"`
	Payload      map[string]interface{} `json:"payload" db:"payload"`
	Result       map[string]interface{} `json:"result,omitempty" db:"result"`
	Status       string                 `json:"status" db:"status"` // pending, success, failed, timeout
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Duration     int64                  `json:"duration,omitempty" db:"duration"`
	ErrorMessage string                 `json:"error_message,omitempty" db:"error_message"`
}

// CommandStatus 命令状态
type CommandStatus string

const (
	CommandStatusPending  CommandStatus = "pending"
	CommandStatusSuccess  CommandStatus = "success"
	CommandStatusFailed   CommandStatus = "failed"
	CommandStatusTimeout  CommandStatus = "timeout"
	CommandStatusCanceled CommandStatus = "canceled"
)
