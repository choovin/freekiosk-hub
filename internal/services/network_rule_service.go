// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// NetworkRuleService 网络规则服务接口
type NetworkRuleService interface {
	// 网络规则CRUD
	CreateRule(tenantID, name, description string, ruleType models.NetworkRuleType, pattern string, action models.NetworkRuleAction, priority int, enabled bool, deviceID string) (*models.NetworkRule, error)
	GetRule(id string) (*models.NetworkRule, error)
	UpdateRule(rule *models.NetworkRule) error
	DeleteRule(id string) error
	ListRules(tenantID string, limit, offset int) ([]*models.NetworkRule, int64, error)

	// 规则匹配检查
	MatchRule(deviceID string, domain, ip, url string) (*models.NetworkRule, error)
	IsWhitelisted(deviceID string, domain, ip, url string) (bool, error)

	// 流量日志
	RecordTrafficLog(tenantID, deviceID, sourceIP, destIP string, destPort int, protocol, domain, url string, bytesIn, bytesOut int64, ruleID, action string) error
	ListTrafficLogs(tenantID, deviceID, startDate, endDate string, limit, offset int) ([]*models.TrafficLog, int64, error)

	// 流量统计
	GetTrafficStats(tenantID, deviceID, date string) ([]*models.TrafficStats, error)
	GetTopDomains(tenantID, deviceID, date string, limit int) ([]*models.TrafficStats, error)

	// 白名单
	AddToWhitelist(tenantID, name, entryType, pattern, description, deviceID string) error
	RemoveFromWhitelist(id string) error
	ListWhitelist(tenantID string) ([]*models.NetworkWhitelist, error)
}

// DefaultNetworkRuleService 默认实现
type DefaultNetworkRuleService struct {
	repo NetworkRuleRepository
}

// NetworkRuleRepository 仓库接口
type NetworkRuleRepository interface {
	InitSchema(ctx context.Context) error
	CreateRule(rule *models.NetworkRule) error
	GetRuleByID(id string) (*models.NetworkRule, error)
	UpdateRule(rule *models.NetworkRule) error
	DeleteRule(id string) error
	ListRules(tenantID string, limit, offset int) ([]*models.NetworkRule, int64, error)
	ListEnabledRules(tenantID string) ([]*models.NetworkRule, error)
	ListDeviceRules(deviceID string) ([]*models.NetworkRule, error)
	CreateTrafficLog(log *models.TrafficLog) error
	ListTrafficLogs(tenantID string, deviceID string, startDate, endDate string, limit, offset int) ([]*models.TrafficLog, int64, error)
	CreateOrUpdateStats(stats *models.TrafficStats) error
	GetStats(tenantID, deviceID, date, domain string) (*models.TrafficStats, error)
	GetDeviceDailyStats(tenantID, deviceID, date string) ([]*models.TrafficStats, error)
	GetTopDomains(tenantID, deviceID, date string, limit int) ([]*models.TrafficStats, error)
	CreateWhitelist(entry *models.NetworkWhitelist) error
	DeleteWhitelist(id string) error
	ListWhitelist(tenantID string) ([]*models.NetworkWhitelist, error)
}

// NewNetworkRuleService 创建网络规则服务
func NewNetworkRuleService(repo NetworkRuleRepository) *DefaultNetworkRuleService {
	return &DefaultNetworkRuleService{repo: repo}
}

// CreateRule 创建网络规则
func (s *DefaultNetworkRuleService) CreateRule(tenantID, name, description string, ruleType models.NetworkRuleType, pattern string, action models.NetworkRuleAction, priority int, enabled bool, deviceID string) (*models.NetworkRule, error) {
	now := time.Now().Unix()
	rule := &models.NetworkRule{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Type:        ruleType,
		Pattern:     pattern,
		Action:      action,
		Priority:    priority,
		Enabled:     enabled,
		DeviceID:    deviceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRule(rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetRule 获取规则
func (s *DefaultNetworkRuleService) GetRule(id string) (*models.NetworkRule, error) {
	return s.repo.GetRuleByID(id)
}

// UpdateRule 更新规则
func (s *DefaultNetworkRuleService) UpdateRule(rule *models.NetworkRule) error {
	return s.repo.UpdateRule(rule)
}

// DeleteRule 删除规则
func (s *DefaultNetworkRuleService) DeleteRule(id string) error {
	return s.repo.DeleteRule(id)
}

// ListRules 获取规则列表
func (s *DefaultNetworkRuleService) ListRules(tenantID string, limit, offset int) ([]*models.NetworkRule, int64, error) {
	return s.repo.ListRules(tenantID, limit, offset)
}

// MatchRule 匹配规则
func (s *DefaultNetworkRuleService) MatchRule(deviceID string, domain, ip, url string) (*models.NetworkRule, error) {
	// Get device-specific rules first
	deviceRules, err := s.repo.ListDeviceRules(deviceID)
	if err != nil {
		return nil, err
	}

	// Get global rules
	globalRules, err := s.repo.ListDeviceRules("")
	if err != nil {
		return nil, err
	}

	// Combine rules (device-specific first, then global)
	allRules := append(deviceRules, globalRules...)

	for _, rule := range allRules {
		if !rule.Enabled {
			continue
		}

		var matched bool
		switch rule.Type {
		case models.NetworkRuleTypeDomain:
			matched = matchDomain(rule.Pattern, domain)
		case models.NetworkRuleTypeIP:
			matched = matchIP(rule.Pattern, ip)
		case models.NetworkRuleTypeURL:
			matched = matchURL(rule.Pattern, url)
		}

		if matched {
			return rule, nil
		}
	}

	return nil, nil
}

// matchDomain 匹配域名
func matchDomain(pattern, domain string) bool {
	if pattern == "" || domain == "" {
		return false
	}
	// Support wildcard patterns like *.example.com
	if len(pattern) >= 2 && pattern[:2] == "*." {
		suffix := pattern[2:]
		return len(domain) > len(suffix) && domain[len(domain)-len(suffix)-1] == '.' && domain[len(domain)-len(suffix):] == suffix
	}
	return domain == pattern
}

// matchIP 匹配IP地址
func matchIP(pattern, ip string) bool {
	if pattern == "" || ip == "" {
		return false
	}
	// Support CIDR notation in the future
	return ip == pattern
}

// matchURL 匹配URL
func matchURL(pattern, url string) bool {
	if pattern == "" || url == "" {
		return false
	}
	// Simple contains match for now
	return len(url) >= len(pattern) && contains(url, pattern)
}

// contains 判断s是否包含substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsWhitelisted 检查是否在白名单中
func (s *DefaultNetworkRuleService) IsWhitelisted(deviceID string, domain, ip, url string) (bool, error) {
	entries, err := s.repo.ListWhitelist("")
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.DeviceID != "" && entry.DeviceID != deviceID {
			continue
		}

		var matched bool
		switch entry.Type {
		case "domain":
			matched = matchDomain(entry.Pattern, domain)
		case "ip":
			matched = matchIP(entry.Pattern, ip)
		case "url":
			matched = matchURL(entry.Pattern, url)
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}

// RecordTrafficLog 记录流量日志
func (s *DefaultNetworkRuleService) RecordTrafficLog(tenantID, deviceID, sourceIP, destIP string, destPort int, protocol, domain, url string, bytesIn, bytesOut int64, ruleID, action string) error {
	log := &models.TrafficLog{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		DeviceID:       deviceID,
		SourceIP:       sourceIP,
		DestinationIP:  destIP,
		DestinationPort: destPort,
		Domain:         domain,
		URL:            url,
		Protocol:       protocol,
		BytesIn:        bytesIn,
		BytesOut:       bytesOut,
		RuleID:         ruleID,
		Action:         action,
		Timestamp:      time.Now(),
	}

	if err := s.repo.CreateTrafficLog(log); err != nil {
		return err
	}

	// Update stats
	date := time.Now().Format("2006-01-02")
	statsID := fmt.Sprintf("%s-%s-%s-%s", tenantID, deviceID, date, domain)
	blockedCount := int64(0)
	if action == "blocked" {
		blockedCount = 1
	}
	stats := &models.TrafficStats{
		ID:           statsID,
		TenantID:     tenantID,
		DeviceID:    deviceID,
		Date:         date,
		Domain:       domain,
		TotalBytes:   bytesIn + bytesOut,
		RequestCount: 1,
		BlockedCount: blockedCount,
	}

	return s.repo.CreateOrUpdateStats(stats)
}

// ListTrafficLogs 获取流量日志
func (s *DefaultNetworkRuleService) ListTrafficLogs(tenantID, deviceID, startDate, endDate string, limit, offset int) ([]*models.TrafficLog, int64, error) {
	return s.repo.ListTrafficLogs(tenantID, deviceID, startDate, endDate, limit, offset)
}

// GetTrafficStats 获取流量统计
func (s *DefaultNetworkRuleService) GetTrafficStats(tenantID, deviceID, date string) ([]*models.TrafficStats, error) {
	return s.repo.GetDeviceDailyStats(tenantID, deviceID, date)
}

// GetTopDomains 获取流量最大的域名
func (s *DefaultNetworkRuleService) GetTopDomains(tenantID, deviceID, date string, limit int) ([]*models.TrafficStats, error) {
	return s.repo.GetTopDomains(tenantID, deviceID, date, limit)
}

// AddToWhitelist 添加到白名单
func (s *DefaultNetworkRuleService) AddToWhitelist(tenantID, name, entryType, pattern, description, deviceID string) error {
	entry := &models.NetworkWhitelist{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        name,
		Type:        entryType,
		Pattern:     pattern,
		Description: description,
		DeviceID:   deviceID,
		CreatedAt:  time.Now().Unix(),
	}
	return s.repo.CreateWhitelist(entry)
}

// RemoveFromWhitelist 从白名单移除
func (s *DefaultNetworkRuleService) RemoveFromWhitelist(id string) error {
	return s.repo.DeleteWhitelist(id)
}

// ListWhitelist 获取白名单
func (s *DefaultNetworkRuleService) ListWhitelist(tenantID string) ([]*models.NetworkWhitelist, error) {
	return s.repo.ListWhitelist(tenantID)
}
