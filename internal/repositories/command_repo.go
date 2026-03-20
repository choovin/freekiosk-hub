package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// CommandRecordRepository 命令记录仓储接口
type CommandRecordRepository interface {
	Create(ctx context.Context, record *models.CommandRecord) error
	Update(ctx context.Context, record *models.CommandRecord) error
	GetByID(ctx context.Context, id string) (*models.CommandRecord, error)
	GetByCommandID(ctx context.Context, commandID string) (*models.CommandRecord, error)
	ListByDevice(ctx context.Context, tenantID, deviceID string, limit, offset int) ([]*models.CommandRecord, int64, error)
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*models.CommandRecord, int64, error)
	ListPending(ctx context.Context, limit int) ([]*models.CommandRecord, error)
	DeleteOldRecords(ctx context.Context, before time.Time) (int64, error)
}

type commandRecordRepository struct {
	db *sqlx.DB
}

// NewCommandRecordRepository 创建命令记录仓储
func NewCommandRecordRepository(db *sqlx.DB) CommandRecordRepository {
	return &commandRecordRepository{db: db}
}

func (r *commandRecordRepository) Create(ctx context.Context, record *models.CommandRecord) error {
	query := `
		INSERT INTO command_history (id, tenant_id, device_id, command_type, command_id, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		record.ID,
		record.TenantID,
		record.DeviceID,
		record.CommandType,
		record.CommandID,
		record.Payload,
		record.Status,
		record.CreatedAt,
	)
	return err
}

func (r *commandRecordRepository) Update(ctx context.Context, record *models.CommandRecord) error {
	query := `
		UPDATE command_history
		SET result = $1, status = $2, completed_at = $3, duration = $4, error_message = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		record.Result,
		record.Status,
		record.CompletedAt,
		record.Duration,
		record.ErrorMessage,
		record.ID,
	)
	return err
}

func (r *commandRecordRepository) GetByID(ctx context.Context, id string) (*models.CommandRecord, error) {
	var record models.CommandRecord
	query := `SELECT * FROM command_history WHERE id = $1`
	err := r.db.GetContext(ctx, &record, query, id)
	if err != nil {
		return nil, fmt.Errorf("command record not found: %w", err)
	}
	return &record, nil
}

func (r *commandRecordRepository) GetByCommandID(ctx context.Context, commandID string) (*models.CommandRecord, error) {
	var record models.CommandRecord
	query := `SELECT * FROM command_history WHERE command_id = $1`
	err := r.db.GetContext(ctx, &record, query, commandID)
	if err != nil {
		return nil, fmt.Errorf("command record not found: %w", err)
	}
	return &record, nil
}

func (r *commandRecordRepository) ListByDevice(ctx context.Context, tenantID, deviceID string, limit, offset int) ([]*models.CommandRecord, int64, error) {
	var records []*models.CommandRecord
	query := `
		SELECT * FROM command_history
		WHERE tenant_id = $1 AND device_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	err := r.db.SelectContext(ctx, &records, query, tenantID, deviceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM command_history WHERE tenant_id = $1 AND device_id = $2`
	err = r.db.GetContext(ctx, &total, countQuery, tenantID, deviceID)

	return records, total, err
}

func (r *commandRecordRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*models.CommandRecord, int64, error) {
	var records []*models.CommandRecord
	query := `
		SELECT * FROM command_history
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &records, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM command_history WHERE tenant_id = $1`
	err = r.db.GetContext(ctx, &total, countQuery, tenantID)

	return records, total, err
}

func (r *commandRecordRepository) ListPending(ctx context.Context, limit int) ([]*models.CommandRecord, error) {
	var records []*models.CommandRecord
	query := `
		SELECT * FROM command_history
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &records, query, limit)
	return records, err
}

func (r *commandRecordRepository) DeleteOldRecords(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM command_history WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
