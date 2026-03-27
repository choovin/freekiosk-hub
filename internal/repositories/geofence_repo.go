package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// GeofenceRepository 地理围栏仓库接口
type GeofenceRepository interface {
	InitSchema(ctx interface{}) error
	Create(gf *models.Geofence) error
	GetByID(id string) (*models.Geofence, error)
	Update(gf *models.Geofence) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.Geofence, int64, error)
	ListActive(tenantID string) ([]*models.Geofence, error)
	AssignDevice(geofenceID, deviceID, tenantID, assignedBy string) error
	UnassignDevice(geofenceID, deviceID string) error
	GetDeviceGeofences(deviceID string) ([]*models.Geofence, error)
	GetAssignedDevices(geofenceID string) ([]string, error)
	RecordEvent(event *models.GeofenceEvent) error
	GetDeviceEvents(deviceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error)
	GetGeofenceEvents(geofenceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error)
}

// SQLiteGeofenceRepository SQLite实现
type SQLiteGeofenceRepository struct {
	db *sqlx.DB
}

// NewSQLiteGeofenceRepository 创建地理围栏仓库
func NewSQLiteGeofenceRepository(db interface{}) *SQLiteGeofenceRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteGeofenceRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteGeofenceRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS geofences (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			tenant_id TEXT NOT NULL,
			fence_type TEXT NOT NULL DEFAULT 'circle',
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			radius REAL NOT NULL DEFAULT 100,
			coordinates TEXT,
			is_active INTEGER DEFAULT 1,
			alert_on_enter INTEGER DEFAULT 1,
			alert_on_exit INTEGER DEFAULT 1,
			time_restriction TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_geofences_tenant ON geofences(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_geofences_active ON geofences(is_active);

		CREATE TABLE IF NOT EXISTS geofence_assignments (
			id TEXT PRIMARY KEY,
			geofence_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			assigned_by TEXT,
			is_active INTEGER DEFAULT 1,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (geofence_id) REFERENCES geofences(id) ON DELETE CASCADE,
			UNIQUE(geofence_id, device_id)
		);

		CREATE INDEX IF NOT EXISTS idx_geofence_assignments_device ON geofence_assignments(device_id);
		CREATE INDEX IF NOT EXISTS idx_geofence_assignments_geofence ON geofence_assignments(geofence_id);

		CREATE TABLE IF NOT EXISTS geofence_events (
			id TEXT PRIMARY KEY,
			geofence_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			geofence_name TEXT,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (geofence_id) REFERENCES geofences(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_geofence_events_device ON geofence_events(device_id);
		CREATE INDEX IF NOT EXISTS idx_geofence_events_geofence ON geofence_events(geofence_id);
		CREATE INDEX IF NOT EXISTS idx_geofence_events_timestamp ON geofence_events(timestamp);
	`
	_, err := r.db.Exec(schema)
	return err
}

// Create 创建地理围栏
func (r *SQLiteGeofenceRepository) Create(gf *models.Geofence) error {
	query := `
		INSERT INTO geofences (
			id, name, description, tenant_id, fence_type,
			latitude, longitude, radius, coordinates,
			is_active, alert_on_enter, alert_on_exit, time_restriction,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		gf.ID, gf.Name, gf.Description, gf.TenantID, gf.FenceType,
		gf.Latitude, gf.Longitude, gf.Radius, gf.Coordinates,
		boolToInt(gf.IsActive), boolToInt(gf.AlertOnEnter), boolToInt(gf.AlertOnExit), gf.TimeRestriction,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create geofence: %w", err)
	}
	gf.CreatedAt = now
	gf.UpdatedAt = now
	return nil
}

// GetByID 根据ID获取
func (r *SQLiteGeofenceRepository) GetByID(id string) (*models.Geofence, error) {
	var gf models.Geofence
	query := `SELECT * FROM geofences WHERE id = ?`
	err := r.db.Get(&gf, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get geofence: %w", err)
	}
	gf.IsActive = gf.IsActive // restore bool from int
	return &gf, nil
}

// Update 更新地理围栏
func (r *SQLiteGeofenceRepository) Update(gf *models.Geofence) error {
	query := `
		UPDATE geofences SET
			name = ?, description = ?, fence_type = ?,
			latitude = ?, longitude = ?, radius = ?, coordinates = ?,
			is_active = ?, alert_on_enter = ?, alert_on_exit = ?, time_restriction = ?,
			updated_at = ?
		WHERE id = ?
	`
	gf.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		gf.Name, gf.Description, gf.FenceType,
		gf.Latitude, gf.Longitude, gf.Radius, gf.Coordinates,
		boolToInt(gf.IsActive), boolToInt(gf.AlertOnEnter), boolToInt(gf.AlertOnExit), gf.TimeRestriction,
		gf.UpdatedAt, gf.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update geofence: %w", err)
	}
	return nil
}

// Delete 删除地理围栏
func (r *SQLiteGeofenceRepository) Delete(id string) error {
	query := `DELETE FROM geofences WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// List 获取地理围栏列表
func (r *SQLiteGeofenceRepository) List(tenantID string, limit, offset int) ([]*models.Geofence, int64, error) {
	var geofences []*models.Geofence
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT * FROM geofences WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&geofences, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list geofences: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM geofences WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return geofences, total, nil
}

// ListActive 获取所有激活的地理围栏
func (r *SQLiteGeofenceRepository) ListActive(tenantID string) ([]*models.Geofence, error) {
	var geofences []*models.Geofence
	query := `SELECT * FROM geofences WHERE tenant_id = ? AND is_active = 1`
	err := r.db.Select(&geofences, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active geofences: %w", err)
	}
	return geofences, nil
}

// AssignDevice 分配围栏到设备
func (r *SQLiteGeofenceRepository) AssignDevice(geofenceID, deviceID, tenantID, assignedBy string) error {
	id := fmt.Sprintf("gfa-%d", time.Now().UnixNano())
	query := `
		INSERT OR REPLACE INTO geofence_assignments (id, geofence_id, device_id, tenant_id, assigned_by, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query, id, geofenceID, deviceID, tenantID, assignedBy, now)
	return err
}

// UnassignDevice 取消分配
func (r *SQLiteGeofenceRepository) UnassignDevice(geofenceID, deviceID string) error {
	query := `DELETE FROM geofence_assignments WHERE geofence_id = ? AND device_id = ?`
	_, err := r.db.Exec(query, geofenceID, deviceID)
	return err
}

// GetDeviceGeofences 获取设备关联的所有围栏
func (r *SQLiteGeofenceRepository) GetDeviceGeofences(deviceID string) ([]*models.Geofence, error) {
	var geofences []*models.Geofence
	query := `
		SELECT g.* FROM geofences g
		JOIN geofence_assignments ga ON g.id = ga.geofence_id
		WHERE ga.device_id = ? AND ga.is_active = 1 AND g.is_active = 1
	`
	err := r.db.Select(&geofences, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device geofences: %w", err)
	}
	return geofences, nil
}

// GetAssignedDevices 获取围栏关联的所有设备ID
func (r *SQLiteGeofenceRepository) GetAssignedDevices(geofenceID string) ([]string, error) {
	var deviceIDs []string
	query := `SELECT device_id FROM geofence_assignments WHERE geofence_id = ? AND is_active = 1`
	err := r.db.Select(&deviceIDs, query, geofenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned devices: %w", err)
	}
	return deviceIDs, nil
}

// RecordEvent 记录围栏事件
func (r *SQLiteGeofenceRepository) RecordEvent(event *models.GeofenceEvent) error {
	query := `
		INSERT INTO geofence_events (id, geofence_id, device_id, tenant_id, event_type, latitude, longitude, geofence_name, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		event.ID, event.GeofenceID, event.DeviceID, event.TenantID, event.EventType,
		event.Latitude, event.Longitude, event.GeofenceName, event.Timestamp, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record geofence event: %w", err)
	}
	event.CreatedAt = now
	return nil
}

// GetDeviceEvents 获取设备的所有围栏事件
func (r *SQLiteGeofenceRepository) GetDeviceEvents(deviceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var events []*models.GeofenceEvent
	query := `SELECT * FROM geofence_events WHERE device_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&events, query, deviceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get device geofence events: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM geofence_events WHERE device_id = ?`
	r.db.Get(&total, countQuery, deviceID)

	return events, total, nil
}

// GetGeofenceEvents 获取围栏的所有事件
func (r *SQLiteGeofenceRepository) GetGeofenceEvents(geofenceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var events []*models.GeofenceEvent
	query := `SELECT * FROM geofence_events WHERE geofence_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&events, query, geofenceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get geofence events: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM geofence_events WHERE geofence_id = ?`
	r.db.Get(&total, countQuery, geofenceID)

	return events, total, nil
}
