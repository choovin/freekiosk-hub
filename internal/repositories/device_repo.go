package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// DeviceRepository handles device database operations
type DeviceRepository interface {
	// Device CRUD
	Create(ctx context.Context, device *models.Device) error
	GetByID(ctx context.Context, id string) (*models.Device, error)
	GetByDeviceKey(ctx context.Context, deviceKey string) (*models.Device, error)
	GetByTenantAndKey(ctx context.Context, tenantID, deviceKey string) (*models.Device, error)
	Update(ctx context.Context, device *models.Device) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, tenantID string, status string, limit, offset int) ([]*models.Device, int64, error)

	// Device status
	UpdateLastSeen(ctx context.Context, deviceID string, lastSeen time.Time) error
	UpdateStatus(ctx context.Context, deviceID string, status string) error

	// Group membership
	AddToGroup(ctx context.Context, deviceID, groupID string) error
	RemoveFromGroup(ctx context.Context, deviceID, groupID string) error
	GetGroups(ctx context.Context, deviceID string) ([]*models.DeviceGroup, error)
}

type deviceRepository struct {
	db *sqlx.DB
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *sqlx.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) Create(ctx context.Context, device *models.Device) error {
	query := `
		INSERT INTO devices (id, tenant_id, device_key, name, status, device_info, security_policy_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		device.ID, device.TenantID, device.DeviceKey, device.Name, device.Status, device.DeviceInfo, device.SecurityPolicyID,
	).Scan(&device.CreatedAt, &device.UpdatedAt)
}

func (r *deviceRepository) GetByID(ctx context.Context, id string) (*models.Device, error) {
	var device models.Device
	query := `SELECT * FROM devices WHERE id = $1`
	err := r.db.GetContext(ctx, &device, query, id)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return &device, nil
}

func (r *deviceRepository) GetByDeviceKey(ctx context.Context, deviceKey string) (*models.Device, error) {
	var device models.Device
	query := `SELECT * FROM devices WHERE device_key = $1`
	err := r.db.GetContext(ctx, &device, query, deviceKey)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return &device, nil
}

func (r *deviceRepository) GetByTenantAndKey(ctx context.Context, tenantID, deviceKey string) (*models.Device, error) {
	var device models.Device
	query := `SELECT * FROM devices WHERE tenant_id = $1 AND device_key = $2`
	err := r.db.GetContext(ctx, &device, query, tenantID, deviceKey)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return &device, nil
}

func (r *deviceRepository) Update(ctx context.Context, device *models.Device) error {
	query := `
		UPDATE devices
		SET name = $1, status = $2, device_info = $3, security_policy_id = $4, last_seen_at = $5
		WHERE id = $6
		RETURNING updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		device.Name, device.Status, device.DeviceInfo, device.SecurityPolicyID, device.LastSeenAt, device.ID,
	).Scan(&device.UpdatedAt)
}

func (r *deviceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *deviceRepository) List(ctx context.Context, tenantID string, status string, limit, offset int) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var args []interface{}
	argIndex := 1

	query := `SELECT * FROM devices WHERE tenant_id = $` + fmt.Sprint(argIndex)
	args = append(args, tenantID)
	argIndex++

	if status != "" && status != "all" {
		query += ` AND status = $` + fmt.Sprint(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprint(argIndex) + ` OFFSET $` + fmt.Sprint(argIndex+1)
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &devices, query, args...)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM devices WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	if status != "" && status != "all" {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, status)
	}
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)

	return devices, total, err
}

func (r *deviceRepository) UpdateLastSeen(ctx context.Context, deviceID string, lastSeen time.Time) error {
	query := `UPDATE devices SET last_seen_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, lastSeen, deviceID)
	return err
}

func (r *deviceRepository) UpdateStatus(ctx context.Context, deviceID string, status string) error {
	query := `UPDATE devices SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, deviceID)
	return err
}

func (r *deviceRepository) AddToGroup(ctx context.Context, deviceID, groupID string) error {
	query := `INSERT INTO device_group_members (device_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, deviceID, groupID)
	return err
}

func (r *deviceRepository) RemoveFromGroup(ctx context.Context, deviceID, groupID string) error {
	query := `DELETE FROM device_group_members WHERE device_id = $1 AND group_id = $2`
	_, err := r.db.ExecContext(ctx, query, deviceID, groupID)
	return err
}

func (r *deviceRepository) GetGroups(ctx context.Context, deviceID string) ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	query := `
		SELECT g.* FROM device_groups g
		INNER JOIN device_group_members m ON g.id = m.group_id
		WHERE m.device_id = $1
	`
	err := r.db.SelectContext(ctx, &groups, query, deviceID)
	return groups, err
}
