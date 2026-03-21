package models

import (
	"encoding/json"
	"time"
)

// SecurityPolicy 安全策略
type SecurityPolicy struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Settings    SecurityPolicySettings `json:"settings"`
	AppWhitelist []AppWhitelistEntry  `json:"app_whitelist"`
	CreatedAt   time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at" db:"updated_at"`
}

// SecurityPolicySettings 安全策略设置
type SecurityPolicySettings struct {
	// 密码策略
	PasswordPolicy PasswordPolicyConfig `json:"password_policy"`

	// 超时设置
	TimeoutSettings TimeoutConfig `json:"timeout_settings"`

	// 系统加固
	SystemHardening HardeningConfig `json:"system_hardening"`

	// 应用加固
	AppHardening AppHardeningConfig `json:"app_hardening"`

	// 网络限制
	NetworkRestrictions NetworkConfig `json:"network_restrictions"`
}

// PasswordPolicyConfig 密码策略配置
type PasswordPolicyConfig struct {
	Enabled          bool   `json:"enabled"`
	MinLength        int    `json:"min_length"`
	RequireUppercase bool   `json:"require_uppercase"`
	RequireLowercase bool   `json:"require_lowercase"`
	RequireNumbers   bool   `json:"require_numbers"`
	RequireSymbols   bool   `json:"require_symbols"`
	MaxAttempts     int    `json:"max_attempts"`
	LockoutDuration  int    `json:"lockout_duration_minutes"` // 分钟
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	ScreenOffTimeout       int  `json:"screen_off_timeout"`        // 秒，屏幕自动关闭时间
	LockTimeout           int  `json:"lock_timeout"`              // 秒，自动锁定时间
	SessionTimeout        int  `json:"session_timeout"`          // 秒，会话超时时间
	InactivityLockTimeout int  `json:"inactivity_lock_timeout"`   // 秒，无操作自动锁定
}

// HardeningConfig 系统加固配置
type HardeningConfig struct {
	DisableUsbDebug       bool `json:"disable_usb_debug"`        // 禁用 USB 调试
	DisableAdbInstall     bool `json:"disable_adb_install"`       // 禁用 ADB 安装
	DisableSettingsAccess bool `json:"disable_settings_access"`   // 禁用设置访问
	DisableScreenshot     bool `json:"disable_screenshot"`        // 禁用截图
	DisableScreenCapture bool `json:"disable_screen_capture"`     // 禁用屏幕录制
	DisableStatusBar     bool `json:"disable_status_bar"`       // 禁用状态栏
	DisableNavigationBar bool `json:"disable_navigation_bar"`     // 禁用导航栏
	SafeBoot             bool `json:"safe_boot"`               // 安全启动
	DisablePowerMenu     bool `json:"disable_power_menu"`       // 禁用电源菜单
}

// AppHardeningConfig 应用加固配置
type AppHardeningConfig struct {
	AllowBackgroundSwitch bool `json:"allow_background_switch"` // 允许后台应用切换
	AllowReturnKey        bool `json:"allow_return_key"`        // 允许返回键
	AllowRecentApps       bool `json:"allow_recent_apps"`       // 允许最近应用
	ForceFullScreen       bool `json:"force_full_screen"`      // 强制全屏
	HideHomeIndicator     bool `json:"hide_home_indicator"`     // 隐藏 Home 指示器
}

// NetworkConfig 网络限制配置
type NetworkConfig struct {
	AllowWiFi       bool     `json:"allow_wifi"`        // 允许 WiFi
	AllowBluetooth   bool     `json:"allow_bluetooth"`    // 允许蓝牙
	AllowMobileData bool     `json:"allow_mobile_data"`  // 允许移动数据
	AllowedWiFiSSIDs []string `json:"allowed_wifi_ssids"` // 允许的 WiFi SSID 列表
	BlockedPorts     []int    `json:"blocked_ports"`     // 阻止的端口
	ProxyAddress    string    `json:"proxy_address"`     // 代理服务器地址
}

// AppWhitelistEntry 应用白名单条目
type AppWhitelistEntry struct {
	PackageName       string `json:"package_name"`        // 包名
	AppName           string `json:"app_name"`           // 应用名称
	AutoLaunch        bool   `json:"auto_launch"`        // 自启动
	AllowNotifications bool   `json:"allow_notifications"` // 允许通知
	DefaultShortcut   bool   `json:"default_shortcut"`    // 显示默认快捷方式
}

// ToMap converts settings to JSONB map
func (sp *SecurityPolicy) ToMap() (map[string]interface{}, error) {
	data, err := json.Marshal(sp.Settings)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

// ToWhitelistMap converts app whitelist to JSONB array
func (sp *SecurityPolicy) ToWhitelistMap() ([]byte, error) {
	return json.Marshal(sp.AppWhitelist)
}

// GetDefaultSecurityPolicy 返回默认安全策略
func GetDefaultSecurityPolicy(tenantID string) *SecurityPolicy {
	return &SecurityPolicy{
		TenantID:    tenantID,
		Name:        "Default Policy",
		Description: "Default security policy",
		Settings: SecurityPolicySettings{
			PasswordPolicy: PasswordPolicyConfig{
				Enabled:          true,
				MinLength:        4,
				RequireUppercase: false,
				RequireLowercase: false,
				RequireNumbers:   true,
				RequireSymbols:   false,
				MaxAttempts:      3,
				LockoutDuration:  15,
			},
			TimeoutSettings: TimeoutConfig{
				ScreenOffTimeout:       300,
				LockTimeout:           60,
				SessionTimeout:        3600,
				InactivityLockTimeout: 300,
			},
			SystemHardening: HardeningConfig{
				DisableUsbDebug:       true,
				DisableAdbInstall:     true,
				DisableSettingsAccess: true,
				DisableScreenshot:     true,
				DisableScreenCapture:  true,
				DisableStatusBar:     true,
				DisableNavigationBar:  true,
				SafeBoot:             false,
				DisablePowerMenu:     true,
			},
			AppHardening: AppHardeningConfig{
				AllowBackgroundSwitch: false,
				AllowReturnKey:        false,
				AllowRecentApps:       false,
				ForceFullScreen:       true,
				HideHomeIndicator:     true,
			},
			NetworkRestrictions: NetworkConfig{
				AllowWiFi:       true,
				AllowBluetooth:  false,
				AllowMobileData: true,
			},
		},
		AppWhitelist: []AppWhitelistEntry{},
	}
}
