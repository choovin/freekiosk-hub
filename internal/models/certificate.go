package models

import (
	"time"
)

// DeviceCertificate represents a device's X.509 certificate
type DeviceCertificate struct {
	ID             string     `json:"id" db:"id"`
	DeviceID       string     `json:"device_id" db:"device_id"`
	CertificatePem string     `json:"certificate_pem" db:"certificate_pem"`
	SerialNumber   string     `json:"serial_number" db:"serial_number"`
	IssuedAt       time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at" db:"revoked_at"`
	RevokedReason  string     `json:"revoked_reason" db:"revoked_reason"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID                string     `json:"id" db:"id"`
	DeviceID          string     `json:"device_id" db:"device_id"`
	TokenHash         string     `json:"token_hash" db:"token_hash"`
	IssuedAt          time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt         time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at" db:"revoked_at"`
	PreviousTokenHash string     `json:"previous_token_hash" db:"previous_token_hash"`
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// DeviceACL represents an EMQX access control rule
type DeviceACL struct {
	ID         string    `json:"id" db:"id"`
	DeviceID   string    `json:"device_id" db:"device_id"`
	Permission string    `json:"permission" db:"permission"`
	Action     string    `json:"action" db:"action"`
	Topic      string    `json:"topic" db:"topic"`
	Priority   int       `json:"priority" db:"priority"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// IntegrityCheck represents a Play Integrity verification record
type IntegrityCheck struct {
	ID                      string                 `json:"id" db:"id"`
	DeviceID                string                 `json:"device_id" db:"device_id"`
	RequestHash             string                 `json:"request_hash" db:"request_hash"`
	DeviceRecognitionVerdict string                `json:"device_recognition_verdict" db:"device_recognition_verdict"`
	AppRecognitionVerdict   string                 `json:"app_recognition_verdict" db:"app_recognition_verdict"`
	Details                 map[string]interface{} `json:"details" db:"details"`
	CheckedAt               time.Time              `json:"checked_at" db:"checked_at"`
}

// IntegrityVerdict represents Play Integrity verdict values
type IntegrityVerdict string

const (
	IntegrityMeetsDeviceIntegrity IntegrityVerdict = "MEETS_DEVICE_INTEGRITY"
	IntegrityMeetsBasicIntegrity  IntegrityVerdict = "MEETS_BASIC_INTEGRITY"
	IntegrityMeetsStrongIntegrity IntegrityVerdict = "MEETS_STRONG_INTEGRITY"
	IntegrityNoIntegrity          IntegrityVerdict = "NO_INTEGRITY"
)
