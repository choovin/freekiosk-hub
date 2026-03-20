package models

import (
	"time"
)

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        string                 `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Slug      string                 `json:"slug" db:"slug"`
	Plan      string                 `json:"plan" db:"plan"`
	Status    string                 `json:"status" db:"status"`
	Settings  map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// TenantPlan represents available tenant plans
type TenantPlan string

const (
	PlanStarter      TenantPlan = "starter"
	PlanProfessional TenantPlan = "professional"
	PlanEnterprise   TenantPlan = "enterprise"
)

// TenantStatus represents tenant status
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// TenantQuota represents tenant resource limits
type TenantQuota struct {
	MaxDevices    int `json:"max_devices"`
	MaxUsers      int `json:"max_users"`
	MaxGroups     int `json:"max_groups"`
	RetentionDays int `json:"retention_days"`
	APIRateLimit  int `json:"api_rate_limit"`
	StorageGB     int `json:"storage_gb"`
}

// GetDefaultQuota returns the default quota for a plan
func GetDefaultQuota(plan TenantPlan) TenantQuota {
	switch plan {
	case PlanEnterprise:
		return TenantQuota{
			MaxDevices:    10000,
			MaxUsers:      100,
			MaxGroups:     500,
			RetentionDays: 90,
			APIRateLimit:  10000,
			StorageGB:     500,
		}
	case PlanProfessional:
		return TenantQuota{
			MaxDevices:    1000,
			MaxUsers:      20,
			MaxGroups:     100,
			RetentionDays: 31,
			APIRateLimit:  5000,
			StorageGB:     100,
		}
	default: // starter
		return TenantQuota{
			MaxDevices:    10,
			MaxUsers:      3,
			MaxGroups:     10,
			RetentionDays: 7,
			APIRateLimit:  1000,
			StorageGB:     10,
		}
	}
}
