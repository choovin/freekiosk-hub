package models

// ConfigurationProfile 配置档案
type ConfigurationProfile struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	TenantID    string   `json:"tenant_id" db:"tenant_id"`
	// 密码策略
	PasswordPolicy *PasswordPolicy `json:"password_policy,omitempty" db:"-"`
	PasswordMinLength int `json:"password_min_length" db:"password_min_length"`
	PasswordRequireNumber bool `json:"password_require_number" db:"password_require_number"`
	PasswordRequireSpecial bool `json:"password_require_special" db:"password_require_special"`
	PasswordExpireDays int `json:"password_expire_days" db:"password_expire_days"`
	// 应用限制
	AppWhitelist []string `json:"app_whitelist,omitempty" db:"-"`
	AppBlacklist []string `json:"app_blacklist,omitempty" db:"-"`
	AllowInstallUnknownApps bool `json:"allow_install_unknown_apps" db:"allow_install_unknown_apps"`
	// DB存储字段
	AppWhitelistJSON string `json:"-" db:"app_whitelist"`
	AppBlacklistJSON string `json:"-" db:"app_blacklist"`
	// 时间限制
	TimeRestrictions *TimeRestriction `json:"time_restrictions,omitempty" db:"-"`
	AllowedHoursStart string `json:"allowed_hours_start" db:"allowed_hours_start"` // HH:MM格式
	AllowedHoursEnd string `json:"allowed_hours_end" db:"allowed_hours_end"`
	AllowedDays []int `json:"allowed_days,omitempty" db:"-"` // 0=周日,1=周一...
	// DB存储字段
	AllowedDaysJSON string `json:"-" db:"allowed_days"`
	// 其他设置
	DeviceTimeout int `json:"device_timeout" db:"device_timeout"` // 分钟
	EnableGPS bool `json:"enable_gps" db:"enable_gps"`
	EnableCamera bool `json:"enable_camera" db:"enable_camera"`
	EnableUSB bool `json:"enable_usb" db:"enable_usb"`
	// JSON存储
	SettingsJSON string `json:"settings_json" db:"settings_json"`
	CreatedAt   int64  `json:"created_at" db:"created_at"`
	UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

// PasswordPolicy 密码策略
type PasswordPolicy struct {
	MinLength         int  `json:"min_length"`
	RequireNumber     bool `json:"require_number"`
	RequireSpecial    bool `json:"require_special"`
	ExpireDays        int  `json:"expire_days"`
	PreventReuse      int  `json:"prevent_reuse"` // 历史密码不能重复使用的次数
}

// TimeRestriction 时间限制
type TimeRestriction struct {
	StartHour int `json:"start_hour"` // 0-23
	EndHour   int `json:"end_hour"`   // 0-23
	StartMin  int `json:"start_min"`  // 0-59
	EndMin    int `json:"end_min"`    // 0-59
	AllowedDays []int `json:"allowed_days"` // 0=周日, 1=周一, ...
}

// DeviceConfiguration 设备配置绑定记录
type DeviceConfiguration struct {
	ID              string `json:"id" db:"id"`
	DeviceID        string `json:"device_id" db:"device_id"`
	ConfigurationID string `json:"configuration_id" db:"configuration_id"`
	TenantID        string `json:"tenant_id" db:"tenant_id"`
	AssignedBy      string `json:"assigned_by" db:"assigned_by"`
	AssignedAt      int64  `json:"assigned_at" db:"assigned_at"`
	CreatedAt       int64  `json:"created_at" db:"created_at"`
}

// AppRule 应用规则
type AppRule struct {
	PackageName string `json:"package_name" db:"package_name"`
	RuleType    string `json:"rule_type" db:"rule_type"` // whitelist/blacklist
	AppName     string `json:"app_name" db:"app_name"`
}
