package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// ConfigurationRepository 配置档案仓库接口
type ConfigurationRepository interface {
	InitSchema(ctx interface{}) error
	Create(cfg *models.ConfigurationProfile) error
	GetByID(id string) (*models.ConfigurationProfile, error)
	Update(cfg *models.ConfigurationProfile) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.ConfigurationProfile, int64, error)
	AssignToDevice(deviceID, configID, tenantID, assignedBy string) error
	UnassignFromDevice(deviceID, configID string) error
	GetDeviceConfiguration(deviceID string) (*models.ConfigurationProfile, error)
	GetDeviceConfigurations(deviceID string) ([]*models.ConfigurationProfile, error)
}

// SQLiteConfigurationRepository SQLite实现
type SQLiteConfigurationRepository struct {
	db *sqlx.DB
}

// NewSQLiteConfigurationRepository 创建配置档案仓库
func NewSQLiteConfigurationRepository(db interface{}) *SQLiteConfigurationRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteConfigurationRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteConfigurationRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS configuration_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			tenant_id TEXT NOT NULL,
			password_min_length INTEGER DEFAULT 4,
			password_require_number INTEGER DEFAULT 0,
			password_require_special INTEGER DEFAULT 0,
			password_expire_days INTEGER DEFAULT 0,
			app_whitelist TEXT,
			app_blacklist TEXT,
			allow_install_unknown_apps INTEGER DEFAULT 1,
			allowed_hours_start TEXT DEFAULT '00:00',
			allowed_hours_end TEXT DEFAULT '23:59',
			allowed_days TEXT,
			device_timeout INTEGER DEFAULT 30,
			enable_gps INTEGER DEFAULT 1,
			enable_camera INTEGER DEFAULT 1,
			enable_usb INTEGER DEFAULT 1,
			settings_json TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_config_tenant ON configuration_profiles(tenant_id);

		CREATE TABLE IF NOT EXISTS device_configurations (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			configuration_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			assigned_by TEXT,
			assigned_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (configuration_id) REFERENCES configuration_profiles(id) ON DELETE CASCADE,
			FOREIGN KEY (device_id) REFERENCES mdm_devices(id) ON DELETE CASCADE,
			UNIQUE(device_id, configuration_id)
		);

		CREATE INDEX IF NOT EXISTS idx_device_config_device ON device_configurations(device_id);
		CREATE INDEX IF NOT EXISTS idx_device_config_config ON device_configurations(configuration_id);
	`
	_, err := r.db.Exec(schema)
	return err
}

// Create 创键配置档案
func (r *SQLiteConfigurationRepository) Create(cfg *models.ConfigurationProfile) error {
	// 序列化数组为JSON
	appWhitelistJSON, _ := json.Marshal(cfg.AppWhitelist)
	appBlacklistJSON, _ := json.Marshal(cfg.AppBlacklist)
	allowedDaysJSON, _ := json.Marshal(cfg.AllowedDays)

	query := `
		INSERT INTO configuration_profiles (
			id, name, description, tenant_id,
			password_min_length, password_require_number, password_require_special, password_expire_days,
			app_whitelist, app_blacklist, allow_install_unknown_apps,
			allowed_hours_start, allowed_hours_end, allowed_days,
			device_timeout, enable_gps, enable_camera, enable_usb,
			settings_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		cfg.ID, cfg.Name, cfg.Description, cfg.TenantID,
		cfg.PasswordMinLength, boolToInt(cfg.PasswordRequireNumber),
		boolToInt(cfg.PasswordRequireSpecial), cfg.PasswordExpireDays,
		string(appWhitelistJSON), string(appBlacklistJSON), boolToInt(cfg.AllowInstallUnknownApps),
		cfg.AllowedHoursStart, cfg.AllowedHoursEnd, string(allowedDaysJSON),
		cfg.DeviceTimeout, boolToInt(cfg.EnableGPS), boolToInt(cfg.EnableCamera), boolToInt(cfg.EnableUSB),
		cfg.SettingsJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	return nil
}

// GetByID 根据ID获取
func (r *SQLiteConfigurationRepository) GetByID(id string) (*models.ConfigurationProfile, error) {
	var cfg models.ConfigurationProfile
	query := `SELECT * FROM configuration_profiles WHERE id = ?`
	err := r.db.Get(&cfg, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration not found")
		}
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}
	r.parseJSONFields(&cfg)
	return &cfg, nil
}

// Update 更新配置档案
func (r *SQLiteConfigurationRepository) Update(cfg *models.ConfigurationProfile) error {
	appWhitelistJSON, _ := json.Marshal(cfg.AppWhitelist)
	appBlacklistJSON, _ := json.Marshal(cfg.AppBlacklist)
	allowedDaysJSON, _ := json.Marshal(cfg.AllowedDays)

	query := `
		UPDATE configuration_profiles SET
			name = ?, description = ?,
			password_min_length = ?, password_require_number = ?, password_require_special = ?, password_expire_days = ?,
			app_whitelist = ?, app_blacklist = ?, allow_install_unknown_apps = ?,
			allowed_hours_start = ?, allowed_hours_end = ?, allowed_days = ?,
			device_timeout = ?, enable_gps = ?, enable_camera = ?, enable_usb = ?,
			settings_json = ?, updated_at = ?
		WHERE id = ?
	`
	cfg.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		cfg.Name, cfg.Description,
		cfg.PasswordMinLength, boolToInt(cfg.PasswordRequireNumber),
		boolToInt(cfg.PasswordRequireSpecial), cfg.PasswordExpireDays,
		string(appWhitelistJSON), string(appBlacklistJSON), boolToInt(cfg.AllowInstallUnknownApps),
		cfg.AllowedHoursStart, cfg.AllowedHoursEnd, string(allowedDaysJSON),
		cfg.DeviceTimeout, boolToInt(cfg.EnableGPS), boolToInt(cfg.EnableCamera), boolToInt(cfg.EnableUSB),
		cfg.SettingsJSON, cfg.UpdatedAt, cfg.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}
	return nil
}

// Delete 删除配置档案
func (r *SQLiteConfigurationRepository) Delete(id string) error {
	query := `DELETE FROM configuration_profiles WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// List 获取配置档案列表
func (r *SQLiteConfigurationRepository) List(tenantID string, limit, offset int) ([]*models.ConfigurationProfile, int64, error) {
	var configs []*models.ConfigurationProfile
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT * FROM configuration_profiles WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&configs, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configurations: %w", err)
	}

	// 解析JSON字段
	for _, cfg := range configs {
		r.parseJSONFields(cfg)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM configuration_profiles WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return configs, total, nil
}

// AssignToDevice 分配配置到设备
func (r *SQLiteConfigurationRepository) AssignToDevice(deviceID, configID, tenantID, assignedBy string) error {
	id := fmt.Sprintf("dc-%d", time.Now().UnixNano())
	query := `
		INSERT OR REPLACE INTO device_configurations (id, device_id, configuration_id, tenant_id, assigned_by, assigned_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query, id, deviceID, configID, tenantID, assignedBy, now, now)
	return err
}

// UnassignFromDevice 取消分配
func (r *SQLiteConfigurationRepository) UnassignFromDevice(deviceID, configID string) error {
	query := `DELETE FROM device_configurations WHERE device_id = ? AND configuration_id = ?`
	_, err := r.db.Exec(query, deviceID, configID)
	return err
}

// GetDeviceConfiguration 获取设备当前配置
func (r *SQLiteConfigurationRepository) GetDeviceConfiguration(deviceID string) (*models.ConfigurationProfile, error) {
	var configID string
	query := `SELECT configuration_id FROM device_configurations WHERE device_id = ? ORDER BY assigned_at DESC LIMIT 1`
	err := r.db.Get(&configID, query, deviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没有配置
		}
		return nil, err
	}
	return r.GetByID(configID)
}

// GetDeviceConfigurations 获取设备所有配置
func (r *SQLiteConfigurationRepository) GetDeviceConfigurations(deviceID string) ([]*models.ConfigurationProfile, error) {
	var configs []*models.ConfigurationProfile
	query := `
		SELECT c.* FROM configuration_profiles c
		JOIN device_configurations dc ON c.id = dc.configuration_id
		WHERE dc.device_id = ?
		ORDER BY dc.assigned_at DESC
	`
	err := r.db.Select(&configs, query, deviceID)
	if err != nil {
		return nil, err
	}
	for _, cfg := range configs {
		r.parseJSONFields(cfg)
	}
	return configs, nil
}

// 辅助方法
func (r *SQLiteConfigurationRepository) parseJSONFields(cfg *models.ConfigurationProfile) {
	if cfg.AppWhitelist == nil {
		var whitelist []string
		json.Unmarshal([]byte(cfg.AppWhitelistJSON), &whitelist)
		cfg.AppWhitelist = whitelist
	}
	if cfg.AppBlacklist == nil {
		var blacklist []string
		json.Unmarshal([]byte(cfg.AppBlacklistJSON), &blacklist)
		cfg.AppBlacklist = blacklist
	}
	if cfg.AllowedDays == nil {
		var days []int
		json.Unmarshal([]byte(cfg.AllowedDaysJSON), &days)
		cfg.AllowedDays = days
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseIntArray(strs []string) []int {
	var result []int
	for _, s := range strs {
		var v int
		fmt.Sscanf(s, "%d", &v)
		result = append(result, v)
	}
	return result
}
