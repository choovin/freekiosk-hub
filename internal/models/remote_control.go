package models

// RemoteSession 远程控制会话
type RemoteSession struct {
	ID          string    `json:"id" db:"id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	InitiatorID string    `json:"initiator_id" db:"initiator_id"` // 发起人ID
	// 会话状态: pending, active, ended, expired
	Status      string    `json:"status" db:"status"`
	// 会话类型: view (仅查看), control (控制)
	SessionType string    `json:"session_type" db:"session_type"`
	// WebRTC相关信息
	ICEServers string `json:"ice_servers" db:"ice_servers"` // JSON
	// 时间
	StartedAt   int64     `json:"started_at" db:"started_at"`
	EndedAt     *int64    `json:"ended_at,omitempty" db:"ended_at"`
	ExpiresAt   int64     `json:"expires_at" db:"expires_at"`
	CreatedAt  int64     `json:"created_at" db:"created_at"`
	UpdatedAt  int64     `json:"updated_at" db:"updated_at"`
}

// RemoteSessionEvent 远程控制事件记录
type RemoteSessionEvent struct {
	ID         string `json:"id" db:"id"`
	SessionID  string `json:"session_id" db:"session_id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
	// 事件类型: start, stop, error, screen_capture
	EventType  string `json:"event_type" db:"event_type"`
	Message    string `json:"message" db:"message"`
	Timestamp  int64  `json:"timestamp" db:"timestamp"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}

// ScreenCapture 屏幕截图记录
type ScreenCapture struct {
	ID        string `json:"id" db:"id"`
	SessionID string `json:"session_id" db:"session_id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
	// 文件信息
	FilePath   string `json:"file_path" db:"file_path"`
	FileSize   int64  `json:"file_size" db:"file_size"`
	MimeType   string `json:"mime_type" db:"mime_type"`
	// 时间戳
	CapturedAt int64  `json:"captured_at" db:"captured_at"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}

// RemoteCommand 远程控制命令
type RemoteCommand struct {
	ID        string `json:"id" db:"id"`
	SessionID string `json:"session_id" db:"session_id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
	// 命令类型: input, tap, swipe, text, screenshot, record_start, record_stop
	CommandType string `json:"command_type" db:"command_type"`
	// 命令参数 (JSON)
	Params    string `json:"params" db:"params"`
	// 状态: pending, sent, delivered, executed, failed
	Status    string `json:"status" db:"status"`
	// 响应
	Response  string `json:"response,omitempty" db:"response"`
	Timestamp int64  `json:"timestamp" db:"timestamp"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}
