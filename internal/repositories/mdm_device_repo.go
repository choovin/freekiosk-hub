package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// MDMTabletRepository MDM设备仓库接口
type MDMTabletRepository interface {
	// 设备CRUD
	CreateDevice(device *models.MDMTablet) error
	GetDeviceByID(id string) (*models.MDMTablet, error)
	GetDeviceByNumber(number string) (*models.MDMTablet, error)
	UpdateDevice(device *models.MDMTablet) error
	DeleteDevice(id string) error
	ListDevices(tenantID string, limit, offset int) ([]*models.MDMTablet, int64, error)
	SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMTablet, int64, error)

	// 设备分组
	CreateGroup(group *models.MDMTabletGroup) error
	UpdateGroup(group *models.MDMTabletGroup) error
	DeleteGroup(id string) error
	ListGroups(tenantID string) ([]*models.MDMTabletGroup, error)

	// 设备标签
	AddTag(tag *models.MDMTabletTag) error
	RemoveTag(deviceID, tag string) error
	GetDeviceTags(deviceID string) ([]*models.MDMTabletTag, error)

	// 设备位置
	UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error
	GetDeviceLocation(deviceID string) (*models.GPSData, error)

	// 设备事件
	RecordEvent(event *models.MDMTabletEvent) error
	GetDeviceEvents(deviceID string, limit int) ([]*models.MDMTabletEvent, error)

	// 批量操作
	BulkUpdateStatus(deviceIDs []string, status string) error
	BulkAssignGroup(deviceIDs []string, groupID string) error
}

// SQLiteMDMTabletRepository SQLite实现的MDM设备仓库
type SQLiteMDMTabletRepository struct {
	db *sqlx.DB
}

// NewSQLiteMDMTabletRepository 创建MDM设备仓库实例
func NewSQLiteMDMTabletRepository(db interface{}) *SQLiteMDMTabletRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteMDMTabletRepository{db: sqlxDB}
}

// CreateDevice 创建设备
func (r *SQLiteMDMTabletRepository) CreateDevice(device *models.MDMTablet) error {
	query := `
		INSERT INTO mdm_devices (
			id, number, name, description, imei, phone, model, manufacturer,
			os_version, sdk_version, app_version, app_version_code, carrier,
			last_lat, last_lng, last_location_time, last_seen, status,
			configuration_id, group_id, tenant_id, metadata, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	now := nowUnix()
	_, err := r.db.Exec(query,
		device.ID, device.Number, device.Name, device.Description,
		device.IMEI, device.Phone, device.Model, device.Manufacturer,
		device.OSVersion, device.SDKVersion, device.AppVersion, device.AppVersionCode,
		device.Carrier, device.LastLat, device.LastLng, device.LastLocationTime,
		device.LastSeen, device.Status, device.ConfigurationID, device.GroupID,
		device.TenantID, device.Metadata, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}
	device.CreatedAt = now
	device.UpdatedAt = now
	return nil
}

// GetDeviceByID 根据ID获取设备
func (r *SQLiteMDMTabletRepository) GetDeviceByID(id string) (*models.MDMTablet, error) {
	var device models.MDMTablet
	query := `SELECT * FROM mdm_devices WHERE id = ?`
	err := r.db.Get(&device, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	return &device, nil
}

// GetDeviceByNumber 根据编号获取设备
func (r *SQLiteMDMTabletRepository) GetDeviceByNumber(number string) (*models.MDMTablet, error) {
	var device models.MDMTablet
	query := `SELECT * FROM mdm_devices WHERE number = ?`
	err := r.db.Get(&device, query, number)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	return &device, nil
}

// UpdateDevice 更新设备
func (r *SQLiteMDMTabletRepository) UpdateDevice(device *models.MDMTablet) error {
	query := `
		UPDATE mdm_devices SET
			number = ?, name = ?, description = ?, imei = ?, phone = ?,
			model = ?, manufacturer = ?, os_version = ?, sdk_version = ?,
			app_version = ?, app_version_code = ?, carrier = ?,
			last_lat = ?, last_lng = ?, last_location_time = ?, last_seen = ?,
			status = ?, configuration_id = ?, group_id = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`
	device.UpdatedAt = nowUnix()
	_, err := r.db.Exec(query,
		device.Number, device.Name, device.Description, device.IMEI, device.Phone,
		device.Model, device.Manufacturer, device.OSVersion, device.SDKVersion,
		device.AppVersion, device.AppVersionCode, device.Carrier,
		device.LastLat, device.LastLng, device.LastLocationTime, device.LastSeen,
		device.Status, device.ConfigurationID, device.GroupID, device.Metadata,
		device.UpdatedAt, device.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}
	return nil
}

// DeleteDevice 删除设备（软删除，标记为retired）
func (r *SQLiteMDMTabletRepository) DeleteDevice(id string) error {
	query := `UPDATE mdm_devices SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, string(models.MDMTabletStatusRetired), nowUnix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}
	return nil
}

// ListDevices 获取设备列表
func (r *SQLiteMDMTabletRepository) ListDevices(tenantID string, limit, offset int) ([]*models.MDMTablet, int64, error) {
	var devices []*models.MDMTablet
	query := `SELECT * FROM mdm_devices WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&devices, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM mdm_devices WHERE tenant_id = ?`
	err = r.db.Get(&total, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	return devices, total, nil
}

// SearchDevices 搜索设备
func (r *SQLiteMDMTabletRepository) SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMTablet, int64, error) {
	var devices []*models.MDMTablet
	var args []interface{}
	var conditions []string

	conditions = append(conditions, "tenant_id = ?")
	args = append(args, filter.TenantID)

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = ?"))
		args = append(args, filter.Status)
	}

	if filter.GroupID != "" {
		conditions = append(conditions, fmt.Sprintf("group_id = ?"))
		args = append(args, filter.GroupID)
	}

	if filter.ConfigurationID != "" {
		conditions = append(conditions, fmt.Sprintf("configuration_id = ?"))
		args = append(args, filter.ConfigurationID)
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(name LIKE ? OR number LIKE ? OR imei LIKE ?)"))
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	if filter.HasLocation != nil {
		if *filter.HasLocation {
			conditions = append(conditions, "(last_lat IS NOT NULL AND last_lng IS NOT NULL)")
		} else {
			conditions = append(conditions, "(last_lat IS NULL OR last_lng IS NULL)")
		}
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mdm_devices WHERE %s", whereClause)
	err := r.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	// Get paginated results
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf("SELECT * FROM mdm_devices WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?", whereClause)
	args = append(args, limit, offset)
	err = r.db.Select(&devices, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search devices: %w", err)
	}

	return devices, total, nil
}

// CreateGroup 创建设备分组
func (r *SQLiteMDMTabletRepository) CreateGroup(group *models.MDMTabletGroup) error {
	query := `
		INSERT INTO device_groups (id, name, parent_id, description, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := nowUnix()
	_, err := r.db.Exec(query, group.ID, group.Name, group.ParentID, group.Description, group.TenantID, now, now)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	group.CreatedAt = now
	group.UpdatedAt = now
	return nil
}

// UpdateGroup 更新设备分组
func (r *SQLiteMDMTabletRepository) UpdateGroup(group *models.MDMTabletGroup) error {
	query := `
		UPDATE device_groups SET name = ?, parent_id = ?, description = ?, updated_at = ?
		WHERE id = ?
	`
	group.UpdatedAt = nowUnix()
	_, err := r.db.Exec(query, group.Name, group.ParentID, group.Description, group.UpdatedAt, group.ID)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}
	return nil
}

// DeleteGroup 删除设备分组
func (r *SQLiteMDMTabletRepository) DeleteGroup(id string) error {
	query := `DELETE FROM device_groups WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

// ListGroups 获取设备分组列表
func (r *SQLiteMDMTabletRepository) ListGroups(tenantID string) ([]*models.MDMTabletGroup, error) {
	var groups []*models.MDMTabletGroup
	query := `SELECT * FROM device_groups WHERE tenant_id = ? ORDER BY name`
	err := r.db.Select(&groups, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	return groups, nil
}

// AddTag 添加设备标签
func (r *SQLiteMDMTabletRepository) AddTag(tag *models.MDMTabletTag) error {
	query := `
		INSERT INTO device_tags (id, device_id, tag, value, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	tag.CreatedAt = nowUnix()
	_, err := r.db.Exec(query, tag.ID, tag.DeviceID, tag.Tag, tag.Value, tag.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add tag: %w", err)
	}
	return nil
}

// RemoveTag 移除设备标签
func (r *SQLiteMDMTabletRepository) RemoveTag(deviceID, tag string) error {
	query := `DELETE FROM device_tags WHERE device_id = ? AND tag = ?`
	_, err := r.db.Exec(query, deviceID, tag)
	if err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}
	return nil
}

// GetDeviceTags 获取设备标签
func (r *SQLiteMDMTabletRepository) GetDeviceTags(deviceID string) ([]*models.MDMTabletTag, error) {
	var tags []*models.MDMTabletTag
	query := `SELECT * FROM device_tags WHERE device_id = ? ORDER BY created_at DESC`
	err := r.db.Select(&tags, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	return tags, nil
}

// UpdateLocation 更新设备位置
func (r *SQLiteMDMTabletRepository) UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error {
	query := `
		UPDATE mdm_devices SET last_lat = ?, last_lng = ?, last_location_time = ?, updated_at = ?
		WHERE id = ?
	`
	now := nowUnix()
	_, err := r.db.Exec(query, lat, lng, timestamp, now, deviceID)
	if err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}
	return nil
}

// GetDeviceLocation 获取设备位置
func (r *SQLiteMDMTabletRepository) GetDeviceLocation(deviceID string) (*models.GPSData, error) {
	var device models.MDMTablet
	query := `SELECT last_lat, last_lng, last_location_time FROM mdm_devices WHERE id = ?`
	err := r.db.Get(&device, query, deviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	if device.LastLat == nil || device.LastLng == nil {
		return nil, fmt.Errorf("device has no location data")
	}

	return &models.GPSData{
		Lat:       *device.LastLat,
		Lng:       *device.LastLng,
		Timestamp: 0,
	}, nil
}

// RecordEvent 记录设备事件
func (r *SQLiteMDMTabletRepository) RecordEvent(event *models.MDMTabletEvent) error {
	query := `
		INSERT INTO device_events (id, device_id, event_type, event_data, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	event.CreatedAt = nowUnix()
	_, err := r.db.Exec(query, event.ID, event.DeviceID, event.EventType, event.EventData, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}
	return nil
}

// GetDeviceEvents 获取设备事件
func (r *SQLiteMDMTabletRepository) GetDeviceEvents(deviceID string, limit int) ([]*models.MDMTabletEvent, error) {
	var events []*models.MDMTabletEvent
	query := `SELECT * FROM device_events WHERE device_id = ? ORDER BY created_at DESC LIMIT ?`
	err := r.db.Select(&events, query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	return events, nil
}

// BulkUpdateStatus 批量更新设备状态
func (r *SQLiteMDMTabletRepository) BulkUpdateStatus(deviceIDs []string, status string) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(deviceIDs))
	for i := range deviceIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"UPDATE mdm_devices SET status = ?, updated_at = ? WHERE id IN (%s)",
		strings.Join(placeholders, ","),
	)
	now := nowUnix()
	finalArgs := make([]interface{}, 0, len(deviceIDs)+2)
	finalArgs = append(finalArgs, status, now)
	for _, id := range deviceIDs {
		finalArgs = append(finalArgs, id)
	}

	_, err := r.db.Exec(query, finalArgs...)
	if err != nil {
		return fmt.Errorf("failed to bulk update status: %w", err)
	}
	return nil
}

// BulkAssignGroup 批量分配设备到分组
func (r *SQLiteMDMTabletRepository) BulkAssignGroup(deviceIDs []string, groupID string) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(deviceIDs))
	for i := range deviceIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"UPDATE mdm_devices SET group_id = ?, updated_at = ? WHERE id IN (%s)",
		strings.Join(placeholders, ","),
	)
	now := nowUnix()
	finalArgs := make([]interface{}, 0, len(deviceIDs)+2)
	finalArgs = append(finalArgs, groupID, now)
	for _, id := range deviceIDs {
		finalArgs = append(finalArgs, id)
	}

	_, err := r.db.Exec(query, finalArgs...)
	if err != nil {
		return fmt.Errorf("failed to bulk assign group: %w", err)
	}
	return nil
}

// nowUnix 返回当前时间戳
func nowUnix() int64 {
	return time.Now().Unix()
}
