package models

// FieldTripDevice represents a field-trip managed device
type FieldTripDevice struct {
	ID           string  `json:"id" db:"id"`
	Name         string  `json:"name" db:"name"`
	GroupID      string  `json:"group_id" db:"group_id"`
	ApiKeyHash   string  `json:"-" db:"api_key_hash"`
	HubURL       string  `json:"hub_url" db:"hub_url"`
	LastSeen     *int64  `json:"last_seen" db:"last_seen"`
	LastLat      *float64 `json:"last_lat" db:"last_lat"`
	LastLng      *float64 `json:"last_lng" db:"last_lng"`
	Status       string  `json:"status" db:"status"`
	SigningPubKey string `json:"signing_pubkey" db:"signing_pubkey"`
	CreatedAt    int64   `json:"created_at" db:"created_at"`
	UpdatedAt    int64   `json:"updated_at" db:"updated_at"`
}

// FieldTripGroup represents a device group
type FieldTripGroup struct {
	ID             string `json:"id" db:"id"`
	Name           string `json:"name" db:"name"`
	GroupKey       string `json:"-" db:"group_key"`
	BroadcastSound string `json:"broadcast_sound" db:"broadcast_sound"`
	UpdatePolicy   string `json:"update_policy" db:"update_policy"`
	CreatedAt      int64  `json:"created_at" db:"created_at"`
	UpdatedAt      int64  `json:"updated_at" db:"updated_at"`
}

// BindRequest is the payload from tablet QR scan
type BindRequest struct {
	DeviceID string `json:"device_id"`
	GroupKey string `json:"group_key"`
	ApiKey   string `json:"api_key"`
	HubURL   string `json:"hub_url"`
}

// BindResponse returned to tablet after successful binding
type BindResponse struct {
	DeviceID       string `json:"device_id"`
	GroupID        string `json:"group_id"`
	GroupName      string `json:"group_name"`
	DeviceName     string `json:"device_name"`
	SigningPubKey  string `json:"signing_pubkey"`
	BroadcastSound string `json:"broadcast_sound"`
	UpdatePolicy   string `json:"update_policy"`
	// MQTT Broker info for tablet connection
	MqttBrokerURL string `json:"mqtt_broker_url"`
	MqttPort      int    `json:"mqtt_port"`
}

// GPSReport sent by tablet
type GPSReport struct {
	DeviceID  string  `json:"device_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
}

// CommandPoll returns pending commands for a device
type CommandPoll struct {
	Whitelist []string `json:"whitelist,omitempty"`
	OTAURL    string   `json:"ota_url,omitempty"`
	Broadcast string   `json:"broadcast,omitempty"`
}

// Broadcast represents a broadcast message record
type Broadcast struct {
	ID              string `json:"id" db:"id"`
	GroupID         string `json:"group_id" db:"group_id"`
	Message         string `json:"message" db:"message"`
	Sound           string `json:"sound" db:"sound"`
	CreatedBy       string `json:"created_by" db:"created_by"`
	CreatedAt       int64  `json:"created_at" db:"created_at"`
	DeliveredCount  int    `json:"delivered_count" db:"delivered_count"`
	FailedCount     int    `json:"failed_count" db:"failed_count"`
}

// PendingCommand represents a pending command for a device
type PendingCommand struct {
	ID           string  `json:"id" db:"id"`
	DeviceID    string  `json:"device_id" db:"device_id"`
	CommandType string  `json:"command_type" db:"command_type"`
	Payload     string  `json:"payload" db:"payload"`
	Status      string  `json:"status" db:"status"`
	CreatedAt   int64   `json:"created_at" db:"created_at"`
	DeliveredAt *int64  `json:"delivered_at" db:"delivered_at"`
}

// FieldTripDeviceInfo represents system information reported by a tablet
type FieldTripDeviceInfo struct {
	BatteryLevel    int    `json:"battery_level"`
	BatteryCharging bool   `json:"battery_charging"`
	IMEI            string `json:"imei"`
	PhoneNumber     string `json:"phone_number"`
	SerialNumber    string `json:"serial_number"`
	AndroidVersion string `json:"android_version"`
	AppVersion     string `json:"app_version"`
	FreeStorage     int64  `json:"free_storage"`
	TotalStorage    int64  `json:"total_storage"`
	FreeMemory      int64  `json:"free_memory"`
	TotalMemory     int64  `json:"total_memory"`
}

// DeviceConfig represents configuration for a device (kiosk mode settings)
type DeviceConfig struct {
	KioskMode      bool     `json:"kiosk_mode"`
	AllowedApps    []string `json:"allowed_apps"`
	ScreenLocked   bool     `json:"screen_locked"`
	Permissions    []string `json:"permissions"`
}
