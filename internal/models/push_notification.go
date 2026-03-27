package models

// PushNotification 推送通知
type PushNotification struct {
	ID          string   `json:"id" db:"id"`
	TenantID    string   `json:"tenant_id" db:"tenant_id"`
	DeviceID    string   `json:"device_id,omitempty" db:"device_id"` // 可选，空表示设备组
	GroupID     string   `json:"group_id,omitempty" db:"group_id"`   // 可选，空表示所有设备
	Title       string   `json:"title" db:"title"`
	Content     string   `json:"content" db:"content"`
	Priority    string   `json:"priority" db:"priority"` // low, normal, high, urgent
	Type        string   `json:"type" db:"type"`         // info, warning, alert, system
	Actions     string   `json:"actions,omitempty" db:"actions"` // JSON array of actions
	ScheduledAt int64    `json:"scheduled_at,omitempty" db:"scheduled_at"` // 计划发送时间
	ExpiredAt   int64    `json:"expired_at,omitempty" db:"expired_at"`     // 过期时间
	Status      string   `json:"status" db:"status"`   // pending, sent, delivered, failed, expired
	SentAt      int64    `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt   int64    `json:"created_at" db:"created_at"`
	UpdatedAt   int64    `json:"updated_at" db:"updated_at"`
}

// PushNotificationReceipt 推送回执
type PushNotificationReceipt struct {
	ID             string `json:"id" db:"id"`
	NotificationID string `json:"notification_id" db:"notification_id"`
	DeviceID      string `json:"device_id" db:"device_id"`
	Status        string `json:"status" db:"status"`   // sent, delivered, read, failed
	DeliveredAt   int64  `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt        int64  `json:"read_at,omitempty" db:"read_at"`
	ErrorMessage  string `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     int64  `json:"created_at" db:"created_at"`
}

// PushAction 推送动作
type PushAction struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Action  string `json:"action"` // URL or intent action
	Icon    string `json:"icon,omitempty"`
	Options string `json:"options,omitempty"` // JSON for additional options
}
