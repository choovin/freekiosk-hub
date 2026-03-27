// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package models

import "time"

// DevicePhoto 设备拍照记录
type DevicePhoto struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	URL         string    `json:"url" db:"url"`               // 照片URL
	ThumbnailURL string   `json:"thumbnail_url" db:"thumbnail_url"` // 缩略图URL
	FileSize    int64     `json:"file_size" db:"file_size"`   // 文件大小(bytes)
	Width       int       `json:"width" db:"width"`          // 宽度
	Height      int       `json:"height" db:"height"`        // 高度
	CapturedAt  time.Time `json:"captured_at" db:"captured_at"` // 拍摄时间
	Description string    `json:"description" db:"description"` // 描述
	CreatedAt   int64     `json:"created_at" db:"created_at"`
}

// Contact 设备联系人
type Contact struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	DeviceID    string `json:"device_id" db:"device_id"`
	Name        string `json:"name" db:"name"`
	Phone       string `json:"phone" db:"phone"`
	Email       string `json:"email" db:"email"`
	Company     string `json:"company" db:"company"`
	JobTitle    string `json:"job_title" db:"job_title"`
	Address     string `json:"address" db:"address"`
	Note        string `json:"note" db:"note"`
	Starred     bool   `json:"starred" db:"starred"`     // 是否星标
	Frequency   int    `json:"frequency" db:"frequency"` // 联系频率
	LastContact *int64 `json:"last_contact" db:"last_contact"` // 上次联系时间
	CreatedAt  int64  `json:"created_at" db:"created_at"`
	UpdatedAt  int64  `json:"updated_at" db:"updated_at"`
}

// LDAPConfig LDAP配置
type LDAPConfig struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Name        string `json:"name" db:"name"`             // 配置名称
	Server      string `json:"server" db:"server"`         // LDAP服务器地址
	Port        int    `json:"port" db:"port"`            // 端口，默认389
	UseSSL      bool   `json:"use_ssl" db:"use_ssl"`      // 是否使用SSL
	BaseDN      string `json:"base_dn" db:"base_dn"`      // 基础DN
	BindDN      string `json:"bind_dn" db:"bind_dn"`      // 绑定DN
	BindPassword string `json:"-" db:"bind_password"`     // 绑定密码(不返回)
	UserFilter  string `json:"user_filter" db:"user_filter"` // 用户过滤规则
	GroupFilter string `json:"group_filter" db:"group_filter"` // 组过滤规则
	SyncInterval int   `json:"sync_interval" db:"sync_interval"` // 同步间隔(分钟)
	Enabled     bool   `json:"enabled" db:"enabled"`     // 是否启用
	LastSyncAt  *int64 `json:"last_sync_at" db:"last_sync_at"` // 上次同步时间
	CreatedAt   int64  `json:"created_at" db:"created_at"`
	UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

// LDAPUser LDAP同步用户
type LDAPUser struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	LDAPConfigID string `json:"ldap_config_id" db:"ldap_config_id"`
	Username     string `json:"username" db:"username"`
	DisplayName  string `json:"display_name" db:"display_name"`
	Email        string `json:"email" db:"email"`
	Phone        string `json:"phone" db:"phone"`
	Department   string `json:"department" db:"department"`
	JobTitle     string `json:"job_title" db:"job_title"`
	Groups       string `json:"groups" db:"groups"`       // JSON数组
	DN           string `json:"dn" db:"dn"`               // LDAP DN
	SyncedAt     int64  `json:"synced_at" db:"synced_at"`
	CreatedAt    int64  `json:"created_at" db:"created_at"`
	UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

// WhiteLabelConfig 白标配置
type WhiteLabelConfig struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Name        string `json:"name" db:"name"`               // 配置名称
	LogoURL     string `json:"logo_url" db:"logo_url"`      // Logo URL
	FaviconURL  string `json:"favicon_url" db:"favicon_url"` // Favicon URL
	PrimaryColor string `json:"primary_color" db:"primary_color"` // 主色调
	SecondaryColor string `json:"secondary_color" db:"secondary_color"` // 次色调
	AccentColor string `json:"accent_color" db:"accent_color"`     // 强调色
	BackgroundColor string `json:"background_color" db:"background_color"` // 背景色
	TextColor   string `json:"text_color" db:"text_color"`   // 文本色
	CustomCSS   string `json:"custom_css" db:"custom_css"` // 自定义CSS
	CustomJS    string `json:"custom_js" db:"custom_js"`  // 自定义JS
	FooterText  string `json:"footer_text" db:"footer_text"` // 页脚文本
	LoginBgURL  string `json:"login_bg_url" db:"login_bg_url"` // 登录背景图
	Enabled     bool   `json:"enabled" db:"enabled"`     // 是否启用
	CreatedAt   int64  `json:"created_at" db:"created_at"`
	UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}
