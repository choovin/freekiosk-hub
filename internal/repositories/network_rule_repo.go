// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// NetworkRuleRepository 网络规则仓库接口
type NetworkRuleRepository interface {
	InitSchema(ctx context.Context) error

	// 网络规则CRUD
	CreateRule(rule *models.NetworkRule) error
	GetRuleByID(id string) (*models.NetworkRule, error)
	UpdateRule(rule *models.NetworkRule) error
	DeleteRule(id string) error
	ListRules(tenantID string, limit, offset int) ([]*models.NetworkRule, int64, error)
	ListEnabledRules(tenantID string) ([]*models.NetworkRule, error)
	ListDeviceRules(deviceID string) ([]*models.NetworkRule, error)

	// 流量日志
	CreateTrafficLog(log *models.TrafficLog) error
	ListTrafficLogs(tenantID string, deviceID string, startDate, endDate string, limit, offset int) ([]*models.TrafficLog, int64, error)

	// 流量统计
	CreateOrUpdateStats(stats *models.TrafficStats) error
	GetStats(tenantID, deviceID, date, domain string) (*models.TrafficStats, error)
	GetDeviceDailyStats(tenantID, deviceID, date string) ([]*models.TrafficStats, error)
	GetTopDomains(tenantID, deviceID, date string, limit int) ([]*models.TrafficStats, error)

	// 白名单
	CreateWhitelist(entry *models.NetworkWhitelist) error
	DeleteWhitelist(id string) error
	ListWhitelist(tenantID string) ([]*models.NetworkWhitelist, error)
}

// SQLiteNetworkRuleRepository SQLite实现
type SQLiteNetworkRuleRepository struct {
	db *sqlx.DB
}

// NewSQLiteNetworkRuleRepository 创建网络规则仓库
func NewSQLiteNetworkRuleRepository(db interface{}) *SQLiteNetworkRuleRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteNetworkRuleRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteNetworkRuleRepository) InitSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS network_rules (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL DEFAULT 'domain',
			pattern TEXT NOT NULL,
			action TEXT NOT NULL DEFAULT 'block',
			priority INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			device_id TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_network_rules_tenant ON network_rules(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_network_rules_device ON network_rules(device_id);
		CREATE INDEX IF NOT EXISTS idx_network_rules_enabled ON network_rules(enabled);

		CREATE TABLE IF NOT EXISTS traffic_logs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			source_ip TEXT,
			destination_ip TEXT,
			destination_port INTEGER,
			domain TEXT,
			url TEXT,
			protocol TEXT,
			bytes_in INTEGER DEFAULT 0,
			bytes_out INTEGER DEFAULT 0,
			rule_id TEXT,
			action TEXT,
			timestamp TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_traffic_logs_tenant ON traffic_logs(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_traffic_logs_device ON traffic_logs(device_id);
		CREATE INDEX IF NOT EXISTS idx_traffic_logs_timestamp ON traffic_logs(timestamp);

		CREATE TABLE IF NOT EXISTS traffic_stats (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			date TEXT NOT NULL,
			domain TEXT,
			total_bytes INTEGER DEFAULT 0,
			request_count INTEGER DEFAULT 0,
			blocked_count INTEGER DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_traffic_stats_tenant_device_date ON traffic_stats(tenant_id, device_id, date);
		CREATE INDEX IF NOT EXISTS idx_traffic_stats_domain ON traffic_stats(domain);

		CREATE TABLE IF NOT EXISTS network_whitelist (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'domain',
			pattern TEXT NOT NULL,
			description TEXT,
			device_id TEXT,
			created_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_network_whitelist_tenant ON network_whitelist(tenant_id);
	`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

// CreateRule 创建网络规则
func (r *SQLiteNetworkRuleRepository) CreateRule(rule *models.NetworkRule) error {
	query := `
		INSERT INTO network_rules (id, tenant_id, name, description, type, pattern, action, priority, enabled, device_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		rule.ID, rule.TenantID, rule.Name, rule.Description, rule.Type, rule.Pattern,
		rule.Action, rule.Priority, rule.Enabled, rule.DeviceID, now, now)
	if err != nil {
		return fmt.Errorf("failed to create network rule: %w", err)
	}
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return nil
}

// GetRuleByID 获取规则
func (r *SQLiteNetworkRuleRepository) GetRuleByID(id string) (*models.NetworkRule, error) {
	var rule models.NetworkRule
	query := `SELECT * FROM network_rules WHERE id = ?`
	err := r.db.Get(&rule, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get network rule: %w", err)
	}
	return &rule, nil
}

// UpdateRule 更新规则
func (r *SQLiteNetworkRuleRepository) UpdateRule(rule *models.NetworkRule) error {
	query := `
		UPDATE network_rules SET
			name = ?, description = ?, type = ?, pattern = ?, action = ?,
			priority = ?, enabled = ?, device_id = ?, updated_at = ?
		WHERE id = ?`
	rule.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		rule.Name, rule.Description, rule.Type, rule.Pattern, rule.Action,
		rule.Priority, rule.Enabled, rule.DeviceID, rule.UpdatedAt, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to update network rule: %w", err)
	}
	return nil
}

// DeleteRule 删除规则
func (r *SQLiteNetworkRuleRepository) DeleteRule(id string) error {
	query := `DELETE FROM network_rules WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// ListRules 获取规则列表
func (r *SQLiteNetworkRuleRepository) ListRules(tenantID string, limit, offset int) ([]*models.NetworkRule, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var rules []*models.NetworkRule
	query := `SELECT * FROM network_rules WHERE tenant_id = ? ORDER BY priority ASC, created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&rules, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list network rules: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM network_rules WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return rules, total, nil
}

// ListEnabledRules 获取所有启用的规则
func (r *SQLiteNetworkRuleRepository) ListEnabledRules(tenantID string) ([]*models.NetworkRule, error) {
	var rules []*models.NetworkRule
	query := `SELECT * FROM network_rules WHERE tenant_id = ? AND enabled = 1 ORDER BY priority ASC`
	err := r.db.Select(&rules, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled network rules: %w", err)
	}
	return rules, nil
}

// ListDeviceRules 获取设备关联的规则
func (r *SQLiteNetworkRuleRepository) ListDeviceRules(deviceID string) ([]*models.NetworkRule, error) {
	var rules []*models.NetworkRule
	query := `SELECT * FROM network_rules WHERE device_id = ? OR device_id = '' ORDER BY priority ASC`
	err := r.db.Select(&rules, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list device network rules: %w", err)
	}
	return rules, nil
}

// CreateTrafficLog 创建流量日志
func (r *SQLiteNetworkRuleRepository) CreateTrafficLog(log *models.TrafficLog) error {
	query := `
		INSERT INTO traffic_logs (id, tenant_id, device_id, source_ip, destination_ip, destination_port, domain, url, protocol, bytes_in, bytes_out, rule_id, action, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query,
		log.ID, log.TenantID, log.DeviceID, log.SourceIP, log.DestinationIP, log.DestinationPort,
		log.Domain, log.URL, log.Protocol, log.BytesIn, log.BytesOut, log.RuleID, log.Action, log.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create traffic log: %w", err)
	}
	return nil
}

// ListTrafficLogs 获取流量日志
func (r *SQLiteNetworkRuleRepository) ListTrafficLogs(tenantID string, deviceID string, startDate, endDate string, limit, offset int) ([]*models.TrafficLog, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []*models.TrafficLog
	query := `SELECT * FROM traffic_logs WHERE tenant_id = ?`
	countQuery := `SELECT COUNT(*) FROM traffic_logs WHERE tenant_id = ?`
	args := []interface{}{tenantID}

	if deviceID != "" {
		query += ` AND device_id = ?`
		countQuery += ` AND device_id = ?`
		args = append(args, deviceID)
	}
	if startDate != "" {
		query += ` AND timestamp >= ?`
		countQuery += ` AND timestamp >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND timestamp <= ?`
		countQuery += ` AND timestamp <= ?`
		args = append(args, endDate)
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	err := r.db.Select(&logs, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list traffic logs: %w", err)
	}

	var total int64
	r.db.Get(&total, countQuery, args[:len(args)-2]...)

	return logs, total, nil
}

// CreateOrUpdateStats 创建或更新流量统计
func (r *SQLiteNetworkRuleRepository) CreateOrUpdateStats(stats *models.TrafficStats) error {
	query := `
		INSERT INTO traffic_stats (id, tenant_id, device_id, date, domain, total_bytes, request_count, blocked_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			total_bytes = total_bytes + excluded.total_bytes,
			request_count = request_count + excluded.request_count,
			blocked_count = blocked_count + excluded.blocked_count`
	_, err := r.db.Exec(query, stats.ID, stats.TenantID, stats.DeviceID, stats.Date, stats.Domain, stats.TotalBytes, stats.RequestCount, stats.BlockedCount)
	if err != nil {
		return fmt.Errorf("failed to create/update traffic stats: %w", err)
	}
	return nil
}

// GetStats 获取统计
func (r *SQLiteNetworkRuleRepository) GetStats(tenantID, deviceID, date, domain string) (*models.TrafficStats, error) {
	var stats models.TrafficStats
	query := `SELECT * FROM traffic_stats WHERE tenant_id = ? AND device_id = ? AND date = ? AND domain = ?`
	err := r.db.Get(&stats, query, tenantID, deviceID, date, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic stats: %w", err)
	}
	return &stats, nil
}

// GetDeviceDailyStats 获取设备每日统计
func (r *SQLiteNetworkRuleRepository) GetDeviceDailyStats(tenantID, deviceID, date string) ([]*models.TrafficStats, error) {
	var stats []*models.TrafficStats
	query := `SELECT * FROM traffic_stats WHERE tenant_id = ? AND device_id = ? AND date = ? ORDER BY total_bytes DESC`
	err := r.db.Select(&stats, query, tenantID, deviceID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get device daily stats: %w", err)
	}
	return stats, nil
}

// GetTopDomains 获取流量最大的域名
func (r *SQLiteNetworkRuleRepository) GetTopDomains(tenantID, deviceID, date string, limit int) ([]*models.TrafficStats, error) {
	var stats []*models.TrafficStats
	query := `SELECT * FROM traffic_stats WHERE tenant_id = ? AND device_id = ? AND date = ? ORDER BY total_bytes DESC LIMIT ?`
	err := r.db.Select(&stats, query, tenantID, deviceID, date, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top domains: %w", err)
	}
	return stats, nil
}

// CreateWhitelist 创建白名单条目
func (r *SQLiteNetworkRuleRepository) CreateWhitelist(entry *models.NetworkWhitelist) error {
	query := `
		INSERT INTO network_whitelist (id, tenant_id, name, type, pattern, description, device_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, entry.ID, entry.TenantID, entry.Name, entry.Type, entry.Pattern, entry.Description, entry.DeviceID, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to create whitelist entry: %w", err)
	}
	return nil
}

// DeleteWhitelist 删除白名单条目
func (r *SQLiteNetworkRuleRepository) DeleteWhitelist(id string) error {
	query := `DELETE FROM network_whitelist WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// ListWhitelist 获取白名单列表
func (r *SQLiteNetworkRuleRepository) ListWhitelist(tenantID string) ([]*models.NetworkWhitelist, error) {
	var entries []*models.NetworkWhitelist
	query := `SELECT * FROM network_whitelist WHERE tenant_id = ? ORDER BY created_at DESC`
	err := r.db.Select(&entries, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list whitelist: %w", err)
	}
	return entries, nil
}
