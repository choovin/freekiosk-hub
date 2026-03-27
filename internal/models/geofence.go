package models

// Geofence 地理围栏模型
type Geofence struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	// 围栏类型: circle (圆形) 或 polygon (多边形)
	FenceType   string    `json:"fence_type" db:"fence_type"`
	// 圆形围栏: 中心点和半径(米)
	Latitude    float64   `json:"latitude" db:"latitude"`
	Longitude   float64   `json:"longitude" db:"longitude"`
	Radius      float64   `json:"radius" db:"radius"` // 米
	// 多边形围栏: 顶点坐标JSON
	Coordinates string    `json:"coordinates" db:"coordinates"` // JSON数组
	// 状态
	IsActive   bool      `json:"is_active" db:"is_active"`
	// 触发动作
	AlertOnEnter  bool   `json:"alert_on_enter" db:"alert_on_enter"`
	AlertOnExit   bool   `json:"alert_on_exit" db:"alert_on_exit"`
	// 时间限制 (可选)
	TimeRestriction string `json:"time_restriction" db:"time_restriction"` // JSON
	// 元数据
	CreatedAt  int64     `json:"created_at" db:"created_at"`
	UpdatedAt  int64     `json:"updated_at" db:"updated_at"`
}

// GeofenceEvent 地理围栏事件记录
type GeofenceEvent struct {
	ID          string `json:"id" db:"id"`
	GeofenceID  string `json:"geofence_id" db:"geofence_id"`
	DeviceID    string `json:"device_id" db:"device_id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	// 事件类型: enter (进入) 或 exit (离开)
	EventType   string `json:"event_type" db:"event_type"`
	// 触发时的设备位置
	Latitude    float64 `json:"latitude" db:"latitude"`
	Longitude   float64 `json:"longitude" db:"longitude"`
	// 围栏名称 (冗余存储便于查询)
	GeofenceName string `json:"geofence_name" db:"geofence_name"`
	// 时间戳
	Timestamp   int64  `json:"timestamp" db:"timestamp"`
	CreatedAt  int64  `json:"created_at" db:"created_at"`
}

// GeofenceAssignment 设备与围栏的关联
type GeofenceAssignment struct {
	ID          string `json:"id" db:"id"`
	GeofenceID  string `json:"geofence_id" db:"geofence_id"`
	DeviceID    string `json:"device_id" db:"device_id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	AssignedBy  string `json:"assigned_by" db:"assigned_by"`
	IsActive   bool   `json:"is_active" db:"is_active"`
	CreatedAt  int64  `json:"created_at" db:"created_at"`
}
