// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// AuditLogType represents the type of audit log entry
type AuditLogType string

const (
	AuditLogTypeUser     AuditLogType = "user"
	AuditLogTypeSystem   AuditLogType = "system"
	AuditLogTypeDevice   AuditLogType = "device"
)

// AuditAction represents the action performed
type AuditAction string

// User actions
const (
	AuditActionUserLogin       AuditAction = "user.login"
	AuditActionUserLogout      AuditAction = "user.logout"
	AuditActionUserCreate      AuditAction = "user.create"
	AuditActionUserUpdate      AuditAction = "user.update"
	AuditActionUserDelete      AuditAction = "user.delete"
)

// Tenant actions
const (
	AuditActionTenantCreate    AuditAction = "tenant.create"
	AuditActionTenantUpdate    AuditAction = "tenant.update"
	AuditActionTenantDelete    AuditAction = "tenant.delete"
	AuditActionTenantSuspend   AuditAction = "tenant.suspend"
)

// Device actions
const (
	AuditActionDeviceRegister  AuditAction = "device.register"
	AuditActionDeviceOnline    AuditAction = "device.online"
	AuditActionDeviceOffline   AuditAction = "device.offline"
	AuditActionDeviceUpdate    AuditAction = "device.update"
	AuditActionDeviceDelete    AuditAction = "device.delete"
)

// Policy actions
const (
	AuditActionPolicyCreate    AuditAction = "policy.create"
	AuditActionPolicyUpdate    AuditAction = "policy.update"
	AuditActionPolicyDelete    AuditAction = "policy.delete"
	AuditActionPolicyAssign    AuditAction = "policy.assign"
)

// Command actions
const (
	AuditActionCommandSend     AuditAction = "command.send"
	AuditActionCommandSuccess  AuditAction = "command.success"
	AuditActionCommandFailed   AuditAction = "command.failed"
	AuditActionCommandCancel   AuditAction = "command.cancel"
)

// Alert actions
const (
	AuditActionAlertTrigger    AuditAction = "alert.trigger"
	AuditActionAlertResolve    AuditAction = "alert.resolve"
	AuditActionAlertNotify    AuditAction = "alert.notify"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	ActorType    AuditLogType           `json:"actor_type"`
	ActorID      string                 `json:"actor_id"`
	Action       AuditAction            `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditService handles audit logging
type AuditService struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewAuditService creates a new audit service
func NewAuditService(db *sqlx.DB) *AuditService {
	return &AuditService{
		db:     db,
		logger: slog.Default().With("component", "audit"),
	}
}

// InitTable creates the audit_logs table
func (s *AuditService) InitTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		actor_type TEXT NOT NULL,
		actor_id TEXT,
		action TEXT NOT NULL,
		resource_type TEXT,
		resource_id TEXT,
		details TEXT NOT NULL DEFAULT '{}',
		ip_address TEXT,
		user_agent TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Log records an audit log entry
func (s *AuditService) Log(ctx context.Context, entry *AuditLog) error {
	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		return err
	}

	query := `INSERT INTO audit_logs (id, tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = s.db.ExecContext(ctx, query,
		entry.ID,
		entry.TenantID,
		entry.ActorType,
		entry.ActorID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		string(detailsJSON),
		entry.IPAddress,
		entry.UserAgent,
	)
	if err != nil {
		s.logger.Error("Failed to write audit log", "error", err, "action", entry.Action)
		return err
	}

	s.logger.Debug("Audit log recorded",
		"action", entry.Action,
		"actor_type", entry.ActorType,
		"actor_id", entry.ActorID,
		"resource_type", entry.ResourceType,
		"resource_id", entry.ResourceID,
	)

	return nil
}

// Query retrieves audit logs with filters
func (s *AuditService) Query(ctx context.Context, filter *AuditLogFilter) ([]*AuditLog, int64, error) {
	var logs []*AuditLog
	var total int64

	// Build query
	query := `SELECT id, tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	args := []interface{}{}

	if filter.TenantID != "" {
		query += ` AND tenant_id = $` + string(rune('1'+len(args)))
		countQuery += ` AND tenant_id = $` + string(rune('1'+len(args)))
		args = append(args, filter.TenantID)
	}
	if filter.ActorID != "" {
		query += ` AND actor_id = $` + string(rune('1'+len(args)))
		countQuery += ` AND actor_id = $` + string(rune('1'+len(args)))
		args = append(args, filter.ActorID)
	}
	if filter.Action != "" {
		query += ` AND action = $` + string(rune('1'+len(args)))
		countQuery += ` AND action = $` + string(rune('1'+len(args)))
		args = append(args, filter.Action)
	}
	if filter.ResourceType != "" {
		query += ` AND resource_type = $` + string(rune('1'+len(args)))
		countQuery += ` AND resource_type = $` + string(rune('1'+len(args)))
		args = append(args, filter.ResourceType)
	}

	// Count total
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	query += ` ORDER BY created_at DESC LIMIT $` + string(rune('1'+len(args)))
	args = append(args, filter.Limit)
	query += ` OFFSET $` + string(rune('1'+len(args)))
	args = append(args, filter.Offset)

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var log AuditLog
		var detailsJSON string
		err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.ActorType,
			&log.ActorID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal([]byte(detailsJSON), &log.Details); err != nil {
			log.Details = make(map[string]interface{})
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// AuditLogFilter contains filter options for querying audit logs
type AuditLogFilter struct {
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	Limit        int
	Offset       int
}
