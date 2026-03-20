package models

import (
	"time"
)

// Device represents a registered kiosk device
type Device struct {
	ID               string                 `json:"id" db:"id"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	DeviceKey        string                 `json:"device_key" db:"device_key"`
	Name             string                 `json:"name" db:"name"`
	Status           string                 `json:"status" db:"status"`
	DeviceInfo       map[string]interface{} `json:"device_info" db:"device_info"`
	SecurityPolicyID *string                `json:"security_policy_id" db:"security_policy_id"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LastSeenAt       *time.Time             `json:"last_seen_at" db:"last_seen_at"`
}

// DeviceStatus represents device status
type DeviceStatus string

const (
	DeviceStatusPending  DeviceStatus = "pending"
	DeviceStatusActive   DeviceStatus = "active"
	DeviceStatusDisabled DeviceStatus = "disabled"
	DeviceStatusDeleted  DeviceStatus = "deleted"
)

// DeviceInfo contains detailed device information
type DeviceInfo struct {
	Brand            string `json:"brand,omitempty"`
	Model            string `json:"model,omitempty"`
	AndroidVersion   string `json:"android_version,omitempty"`
	SDKVersion       int    `json:"sdk_version,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
	IMEI             string `json:"imei,omitempty"`
	WifiMAC          string `json:"wifi_mac,omitempty"`
	BatteryLevel     int    `json:"battery_level,omitempty"`
	BatteryCharging  bool   `json:"battery_charging,omitempty"`
	ScreenOn         bool   `json:"screen_on,omitempty"`
	ScreenBrightness int    `json:"screen_brightness,omitempty"`
	Volume           int    `json:"volume,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	FreeStorage      int64  `json:"free_storage,omitempty"`
	TotalStorage     int64  `json:"total_storage,omitempty"`
	FreeMemory       int64  `json:"free_memory,omitempty"`
	TotalMemory      int64  `json:"total_memory,omitempty"`
}

// ToMap converts DeviceInfo to a map for JSONB storage
func (d *DeviceInfo) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	if d.Brand != "" {
		result["brand"] = d.Brand
	}
	if d.Model != "" {
		result["model"] = d.Model
	}
	if d.AndroidVersion != "" {
		result["android_version"] = d.AndroidVersion
	}
	if d.SDKVersion > 0 {
		result["sdk_version"] = d.SDKVersion
	}
	if d.SerialNumber != "" {
		result["serial_number"] = d.SerialNumber
	}
	if d.IMEI != "" {
		result["imei"] = d.IMEI
	}
	if d.WifiMAC != "" {
		result["wifi_mac"] = d.WifiMAC
	}
	if d.BatteryLevel > 0 {
		result["battery_level"] = d.BatteryLevel
	}
	result["battery_charging"] = d.BatteryCharging
	result["screen_on"] = d.ScreenOn
	if d.ScreenBrightness > 0 {
		result["screen_brightness"] = d.ScreenBrightness
	}
	if d.Volume > 0 {
		result["volume"] = d.Volume
	}
	if d.AppVersion != "" {
		result["app_version"] = d.AppVersion
	}
	if d.FreeStorage > 0 {
		result["free_storage"] = d.FreeStorage
	}
	if d.TotalStorage > 0 {
		result["total_storage"] = d.TotalStorage
	}
	if d.FreeMemory > 0 {
		result["free_memory"] = d.FreeMemory
	}
	if d.TotalMemory > 0 {
		result["total_memory"] = d.TotalMemory
	}
	return result
}

// DeviceRegistration represents a device registration request
type DeviceRegistration struct {
	TenantID       string      `json:"tenant_id"`
	DeviceKey      string      `json:"device_key"`
	CSRPem         string      `json:"csr_pem"`
	IntegrityToken string      `json:"integrity_token,omitempty"`
	DeviceInfo     DeviceInfo  `json:"device_info"`
	Nonce          string      `json:"nonce,omitempty"`
}

// DeviceCredentials represents credentials returned after registration
type DeviceCredentials struct {
	DeviceID       string    `json:"device_id"`
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	ExpiresAt      time.Time `json:"expires_at"`
	CertificatePem string    `json:"certificate_pem"`
	SerialNumber   string    `json:"serial_number"`
	CertExpiresAt  time.Time `json:"cert_expires_at"`
}

// DeviceGroup represents a device group
type DeviceGroup struct {
	ID        string                 `json:"id" db:"id"`
	TenantID  string                 `json:"tenant_id" db:"tenant_id"`
	Name      string                 `json:"name" db:"name"`
	ParentID  *string                `json:"parent_id" db:"parent_id"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}
