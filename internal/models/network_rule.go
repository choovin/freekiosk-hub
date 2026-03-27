// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package models

import "time"

// NetworkRuleType 网络规则类型
type NetworkRuleType string

const (
	NetworkRuleTypeDomain NetworkRuleType = "domain" // 域名过滤
	NetworkRuleTypeIP     NetworkRuleType = "ip"     // IP地址过滤
	NetworkRuleTypeURL    NetworkRuleType = "url"    // URL过滤
)

// NetworkRuleAction 规则动作
type NetworkRuleAction string

const (
	NetworkRuleActionBlock NetworkRuleAction = "block" // 阻止
	NetworkRuleActionAllow NetworkRuleAction = "allow" // 允许
	NetworkRuleActionLog   NetworkRuleAction = "log"   // 仅记录
)

// NetworkRule 网络规则
type NetworkRule struct {
	ID          string           `json:"id" db:"id"`
	TenantID    string           `json:"tenant_id" db:"tenant_id"`
	Name        string           `json:"name" db:"name"`                 // 规则名称
	Description string           `json:"description,omitempty" db:"description"` // 规则描述
	Type       NetworkRuleType  `json:"type" db:"type"`                 // domain/ip/url
	Pattern     string           `json:"pattern" db:"pattern"`           // 匹配模式
	Action     NetworkRuleAction `json:"action" db:"action"`           // block/allow/log
	Priority   int              `json:"priority" db:"priority"`       // 优先级，数字越小优先级越高
	Enabled    bool             `json:"enabled" db:"enabled"`         // 是否启用
	DeviceID   string           `json:"device_id,omitempty" db:"device_id"` // 关联的设备ID，为空则应用于所有设备
	CreatedAt  int64            `json:"created_at" db:"created_at"`
	UpdatedAt  int64            `json:"updated_at" db:"updated_at"`
}

// TrafficLog 流量日志
type TrafficLog struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	DeviceID     string    `json:"device_id" db:"device_id"`
	SourceIP     string    `json:"source_ip" db:"source_ip"`
	DestinationIP string   `json:"destination_ip" db:"destination_ip"`
	DestinationPort int    `json:"destination_port" db:"destination_port"`
	Domain       string    `json:"domain,omitempty" db:"domain"`
	URL          string    `json:"url,omitempty" db:"url"`
	Protocol     string    `json:"protocol" db:"protocol"` // TCP/UDP/HTTP/HTTPS/DNS
	BytesIn      int64     `json:"bytes_in" db:"bytes_in"`
	BytesOut     int64     `json:"bytes_out" db:"bytes_out"`
	RuleID       string    `json:"rule_id,omitempty" db:"rule_id"` // 匹配的规则ID
	Action       string    `json:"action" db:"action"` // allowed/blocked/logged
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// TrafficStats 流量统计
type TrafficStats struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	DeviceID     string `json:"device_id" db:"device_id"`
	Date         string `json:"date" db:"date"` // YYYY-MM-DD格式
	Domain       string `json:"domain,omitempty" db:"domain"`
	TotalBytes   int64  `json:"total_bytes" db:"total_bytes"`
	RequestCount int64  `json:"request_count" db:"request_count"`
	BlockedCount int64  `json:"blocked_count" db:"blocked_count"`
}

// NetworkWhitelist 网络白名单
type NetworkWhitelist struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Name        string `json:"name" db:"name"`
	Type        string `json:"type" db:"type"` // domain/ip/url
	Pattern     string `json:"pattern" db:"pattern"`
	Description string `json:"description,omitempty" db:"description"`
	DeviceID   string `json:"device_id,omitempty" db:"device_id"`
	CreatedAt  int64  `json:"created_at" db:"created_at"`
}
