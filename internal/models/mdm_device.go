package models

// MDMTablet 企业级MDM平板设备模型
type MDMTablet struct {
	ID               string   `json:"id" db:"id"`
	Number           string   `json:"number" db:"number"` // 设备唯一标识
	Name             string   `json:"name" db:"name"`
	Description      string   `json:"description" db:"description"`
	IMEI             string   `json:"imei" db:"imei"`
	Phone            string   `json:"phone" db:"phone"`
	Model            string   `json:"model" db:"model"`
	Manufacturer     string   `json:"manufacturer" db:"manufacturer"`
	OSVersion        string   `json:"os_version" db:"os_version"`
	SDKVersion       int      `json:"sdk_version" db:"sdk_version"`
	AppVersion       string   `json:"app_version" db:"app_version"`
	AppVersionCode   int      `json:"app_version_code" db:"app_version_code"`
	Carrier          string   `json:"carrier" db:"carrier"`
	LastLat          *float64 `json:"last_lat" db:"last_lat"`
	LastLng          *float64 `json:"last_lng" db:"last_lng"`
	LastLocationTime *int64   `json:"last_location_time" db:"last_location_time"`
	LastSeen         *int64   `json:"last_seen" db:"last_seen"`
	Status           string   `json:"status" db:"status"` // active, inactive, lost, retired
	ConfigurationID  *string  `json:"configuration_id" db:"configuration_id"`
	GroupID          *string  `json:"group_id" db:"group_id"`
	TenantID         string   `json:"tenant_id" db:"tenant_id"`
	Metadata         string   `json:"metadata" db:"metadata"` // JSON
	CreatedAt        int64    `json:"created_at" db:"created_at"`
	UpdatedAt        int64    `json:"updated_at" db:"updated_at"`
}

// MDMTabletStatus 设备状态枚举
type MDMTabletStatus string

const (
	MDMTabletStatusActive   MDMTabletStatus = "active"
	MDMTabletStatusInactive MDMTabletStatus = "inactive"
	MDMTabletStatusLost    MDMTabletStatus = "lost"
	MDMTabletStatusRetired MDMTabletStatus = "retired"
)

// MDMTabletGroup 设备分组
type MDMTabletGroup struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	ParentID    *string `json:"parent_id" db:"parent_id"` // 支持层级
	Description string  `json:"description" db:"description"`
	TenantID    string  `json:"tenant_id" db:"tenant_id"`
	CreatedAt   int64   `json:"created_at" db:"created_at"`
	UpdatedAt   int64   `json:"updated_at" db:"updated_at"`
}

// MDMTabletTag 设备标签
type MDMTabletTag struct {
	ID        string `json:"id" db:"id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	Tag       string `json:"tag" db:"tag"`
	Value     string `json:"value" db:"value"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}

// MDMTabletEvent 设备事件
type MDMTabletEvent struct {
	ID        string `json:"id" db:"id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	EventType string `json:"event_type" db:"event_type"`
	EventData string `json:"event_data" db:"event_data"` // JSON
	CreatedAt int64  `json:"created_at" db:"created_at"`
}

// DeviceSearchFilter 设备搜索过滤 (MDM版本)
type DeviceSearchFilter struct {
	TenantID        string   `json:"tenant_id"`
	Status          string   `json:"status,omitempty"`
	GroupID         string   `json:"group_id,omitempty"`
	ConfigurationID string   `json:"configuration_id,omitempty"`
	Search          string   `json:"search,omitempty"` // 搜索 name, number, imei
	Tags            []string `json:"tags,omitempty"`
	HasLocation     *bool    `json:"has_location,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
}

// GPSData GPS位置数据
type GPSData struct {
	Lat       float64 `json:"lat" db:"lat"`
	Lng       float64 `json:"lng" db:"lng"`
	Timestamp int64   `json:"timestamp" db:"timestamp"`
}
