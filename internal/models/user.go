package models

// User 用户
type User struct {
	ID           string   `json:"id" db:"id"`
	TenantID     string   `json:"tenant_id" db:"tenant_id"`
	Username     string   `json:"username" db:"username"`
	PasswordHash string   `json:"-" db:"password_hash"`
	Email        string   `json:"email" db:"email"`
	Phone        string   `json:"phone,omitempty" db:"phone"`
	RealName     string   `json:"real_name,omitempty" db:"real_name"`
	Role         string   `json:"role" db:"role"` // admin, manager, operator, viewer
	Status       string   `json:"status" db:"status"` // active, inactive, locked
	LastLoginAt  int64    `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP  string   `json:"last_login_ip,omitempty" db:"last_login_ip"`
	CreatedAt    int64    `json:"created_at" db:"created_at"`
	UpdatedAt    int64    `json:"updated_at" db:"updated_at"`
}

// Role 角色
type Role struct {
	ID          string   `json:"id" db:"id"`
	TenantID    string   `json:"tenant_id" db:"tenant_id"`
	Name        string   `json:"name" db:"name"`
	Code        string   `json:"code" db:"code"` // admin, manager, operator, viewer
	Description string   `json:"description,omitempty" db:"description"`
	Permissions string   `json:"permissions" db:"permissions"` // JSON array of permission codes
	IsSystem    bool     `json:"is_system" db:"is_system"` // system roles cannot be deleted
	CreatedAt   int64    `json:"created_at" db:"created_at"`
	UpdatedAt   int64    `json:"updated_at" db:"updated_at"`
}

// Permission 权限
type Permission struct {
	ID          string `json:"id" db:"id"`
	Code        string `json:"code" db:"code"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	Category    string `json:"category" db:"category"` // device, config, user, system
}

// UserSession 用户会话
type UserSession struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Token     string `json:"token" db:"token"`
	ExpiresAt int64  `json:"expires_at" db:"expires_at"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}
