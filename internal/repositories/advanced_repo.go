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

// AdvancedRepository 高级功能仓库接口
type AdvancedRepository interface {
	InitSchema(ctx context.Context) error

	// DevicePhoto 设备拍照
	CreateDevicePhoto(photo *models.DevicePhoto) error
	GetDevicePhoto(id string) (*models.DevicePhoto, error)
	ListDevicePhotos(tenantID, deviceID string, limit, offset int) ([]*models.DevicePhoto, int64, error)
	DeleteDevicePhoto(id string) error

	// Contact 联系人
	CreateContact(contact *models.Contact) error
	GetContact(id string) (*models.Contact, error)
	UpdateContact(contact *models.Contact) error
	DeleteContact(id string) error
	ListContacts(tenantID, deviceID string, limit, offset int) ([]*models.Contact, int64, error)
	SearchContacts(tenantID, keyword string, limit, offset int) ([]*models.Contact, int64, error)

	// LDAP配置
	CreateLDAPConfig(config *models.LDAPConfig) error
	GetLDAPConfig(id string) (*models.LDAPConfig, error)
	GetLDAPConfigByTenant(tenantID string) (*models.LDAPConfig, error)
	UpdateLDAPConfig(config *models.LDAPConfig) error
	DeleteLDAPConfig(id string) error
	ListLDAPConfigs(tenantID string) ([]*models.LDAPConfig, error)
	TestLDAPConnection(config *models.LDAPConfig) (bool, string, error)
	SyncLDAPUsers(configID string) ([]*models.LDAPUser, error)
	ListLDAPUsers(tenantID string, limit, offset int) ([]*models.LDAPUser, int64, error)

	// WhiteLabel配置
	CreateWhiteLabelConfig(config *models.WhiteLabelConfig) error
	GetWhiteLabelConfig(id string) (*models.WhiteLabelConfig, error)
	GetWhiteLabelConfigByTenant(tenantID string) (*models.WhiteLabelConfig, error)
	UpdateWhiteLabelConfig(config *models.WhiteLabelConfig) error
	DeleteWhiteLabelConfig(id string) error
}

// SQLiteAdvancedRepository SQLite实现
type SQLiteAdvancedRepository struct {
	db *sqlx.DB
}

// NewSQLiteAdvancedRepository 创建高级功能仓库
func NewSQLiteAdvancedRepository(db interface{}) *SQLiteAdvancedRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteAdvancedRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteAdvancedRepository) InitSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS device_photos (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			url TEXT NOT NULL,
			thumbnail_url TEXT,
			file_size INTEGER DEFAULT 0,
			width INTEGER DEFAULT 0,
			height INTEGER DEFAULT 0,
			captured_at TEXT NOT NULL,
			description TEXT,
			created_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_device_photos_tenant ON device_photos(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_device_photos_device ON device_photos(device_id);
		CREATE INDEX IF NOT EXISTS idx_device_photos_captured ON device_photos(captured_at);

		CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT,
			email TEXT,
			company TEXT,
			job_title TEXT,
			address TEXT,
			note TEXT,
			starred INTEGER DEFAULT 0,
			frequency INTEGER DEFAULT 0,
			last_contact INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_contacts_tenant ON contacts(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_contacts_device ON contacts(device_id);
		CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);
		CREATE INDEX IF NOT EXISTS idx_contacts_phone ON contacts(phone);

		CREATE TABLE IF NOT EXISTS ldap_configs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			server TEXT NOT NULL,
			port INTEGER DEFAULT 389,
			use_ssl INTEGER DEFAULT 0,
			base_dn TEXT NOT NULL,
			bind_dn TEXT,
			bind_password TEXT,
			user_filter TEXT,
			group_filter TEXT,
			sync_interval INTEGER DEFAULT 60,
			enabled INTEGER DEFAULT 0,
			last_sync_at INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_ldap_configs_tenant ON ldap_configs(tenant_id);

		CREATE TABLE IF NOT EXISTS ldap_users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			ldap_config_id TEXT NOT NULL,
			username TEXT NOT NULL,
			display_name TEXT,
			email TEXT,
			phone TEXT,
			department TEXT,
			job_title TEXT,
			groups TEXT,
			dn TEXT,
			synced_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_ldap_users_tenant ON ldap_users(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_ldap_users_config ON ldap_users(ldap_config_id);
		CREATE INDEX IF NOT EXISTS idx_ldap_users_username ON ldap_users(username);

		CREATE TABLE IF NOT EXISTS white_label_configs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			logo_url TEXT,
			favicon_url TEXT,
			primary_color TEXT DEFAULT '#1976D2',
			secondary_color TEXT DEFAULT '#424242',
			accent_color TEXT DEFAULT '#FF5722',
			background_color TEXT DEFAULT '#FFFFFF',
			text_color TEXT DEFAULT '#212121',
			custom_css TEXT,
			custom_js TEXT,
			footer_text TEXT,
			login_bg_url TEXT,
			enabled INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_white_label_configs_tenant ON white_label_configs(tenant_id);
	`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

// DevicePhoto Methods

func (r *SQLiteAdvancedRepository) CreateDevicePhoto(photo *models.DevicePhoto) error {
	query := `
		INSERT INTO device_photos (id, tenant_id, device_id, url, thumbnail_url, file_size, width, height, captured_at, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		photo.ID, photo.TenantID, photo.DeviceID, photo.URL, photo.ThumbnailURL,
		photo.FileSize, photo.Width, photo.Height, photo.CapturedAt, photo.Description, now)
	if err != nil {
		return fmt.Errorf("failed to create device photo: %w", err)
	}
	photo.CreatedAt = now
	return nil
}

func (r *SQLiteAdvancedRepository) GetDevicePhoto(id string) (*models.DevicePhoto, error) {
	var photo models.DevicePhoto
	query := `SELECT * FROM device_photos WHERE id = ?`
	err := r.db.Get(&photo, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device photo: %w", err)
	}
	return &photo, nil
}

func (r *SQLiteAdvancedRepository) ListDevicePhotos(tenantID, deviceID string, limit, offset int) ([]*models.DevicePhoto, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var photos []*models.DevicePhoto
	query := `SELECT * FROM device_photos WHERE tenant_id = ?`
	countQuery := `SELECT COUNT(*) FROM device_photos WHERE tenant_id = ?`
	args := []interface{}{tenantID}

	if deviceID != "" {
		query += ` AND device_id = ?`
		countQuery += ` AND device_id = ?`
		args = append(args, deviceID)
	}

	query += ` ORDER BY captured_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	err := r.db.Select(&photos, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list device photos: %w", err)
	}

	var total int64
	r.db.Get(&total, countQuery, args[:len(args)-2]...)

	return photos, total, nil
}

func (r *SQLiteAdvancedRepository) DeleteDevicePhoto(id string) error {
	query := `DELETE FROM device_photos WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Contact Methods

func (r *SQLiteAdvancedRepository) CreateContact(contact *models.Contact) error {
	query := `
		INSERT INTO contacts (id, tenant_id, device_id, name, phone, email, company, job_title, address, note, starred, frequency, last_contact, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		contact.ID, contact.TenantID, contact.DeviceID, contact.Name, contact.Phone, contact.Email,
		contact.Company, contact.JobTitle, contact.Address, contact.Note, contact.Starred, contact.Frequency,
		contact.LastContact, now, now)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}
	contact.CreatedAt = now
	contact.UpdatedAt = now
	return nil
}

func (r *SQLiteAdvancedRepository) GetContact(id string) (*models.Contact, error) {
	var contact models.Contact
	query := `SELECT * FROM contacts WHERE id = ?`
	err := r.db.Get(&contact, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return &contact, nil
}

func (r *SQLiteAdvancedRepository) UpdateContact(contact *models.Contact) error {
	query := `
		UPDATE contacts SET name = ?, phone = ?, email = ?, company = ?, job_title = ?,
		address = ?, note = ?, starred = ?, frequency = ?, last_contact = ?, updated_at = ?
		WHERE id = ?`
	contact.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		contact.Name, contact.Phone, contact.Email, contact.Company, contact.JobTitle,
		contact.Address, contact.Note, contact.Starred, contact.Frequency, contact.LastContact,
		contact.UpdatedAt, contact.ID)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}
	return nil
}

func (r *SQLiteAdvancedRepository) DeleteContact(id string) error {
	query := `DELETE FROM contacts WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteAdvancedRepository) ListContacts(tenantID, deviceID string, limit, offset int) ([]*models.Contact, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var contacts []*models.Contact
	query := `SELECT * FROM contacts WHERE tenant_id = ?`
	countQuery := `SELECT COUNT(*) FROM contacts WHERE tenant_id = ?`
	args := []interface{}{tenantID}

	if deviceID != "" {
		query += ` AND device_id = ?`
		countQuery += ` AND device_id = ?`
		args = append(args, deviceID)
	}

	query += ` ORDER BY name ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	err := r.db.Select(&contacts, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list contacts: %w", err)
	}

	var total int64
	r.db.Get(&total, countQuery, args[:len(args)-2]...)

	return contacts, total, nil
}

func (r *SQLiteAdvancedRepository) SearchContacts(tenantID, keyword string, limit, offset int) ([]*models.Contact, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var contacts []*models.Contact
	searchPattern := "%" + keyword + "%"
	query := `SELECT * FROM contacts WHERE tenant_id = ? AND (name LIKE ? OR phone LIKE ? OR email LIKE ? OR company LIKE ?)`
	countQuery := `SELECT COUNT(*) FROM contacts WHERE tenant_id = ? AND (name LIKE ? OR phone LIKE ? OR email LIKE ? OR company LIKE ?)`
	args := []interface{}{tenantID, searchPattern, searchPattern, searchPattern, searchPattern}
	countArgs := []interface{}{tenantID, searchPattern, searchPattern, searchPattern, searchPattern}

	query += ` ORDER BY name ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	err := r.db.Select(&contacts, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search contacts: %w", err)
	}

	var total int64
	r.db.Get(&total, countQuery, countArgs...)

	return contacts, total, nil
}

// LDAPConfig Methods

func (r *SQLiteAdvancedRepository) CreateLDAPConfig(config *models.LDAPConfig) error {
	query := `
		INSERT INTO ldap_configs (id, tenant_id, name, server, port, use_ssl, base_dn, bind_dn, bind_password, user_filter, group_filter, sync_interval, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		config.ID, config.TenantID, config.Name, config.Server, config.Port, config.UseSSL,
		config.BaseDN, config.BindDN, config.BindPassword, config.UserFilter, config.GroupFilter,
		config.SyncInterval, config.Enabled, now, now)
	if err != nil {
		return fmt.Errorf("failed to create ldap config: %w", err)
	}
	config.CreatedAt = now
	config.UpdatedAt = now
	return nil
}

func (r *SQLiteAdvancedRepository) GetLDAPConfig(id string) (*models.LDAPConfig, error) {
	var config models.LDAPConfig
	query := `SELECT * FROM ldap_configs WHERE id = ?`
	err := r.db.Get(&config, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ldap config: %w", err)
	}
	return &config, nil
}

func (r *SQLiteAdvancedRepository) GetLDAPConfigByTenant(tenantID string) (*models.LDAPConfig, error) {
	var config models.LDAPConfig
	query := `SELECT * FROM ldap_configs WHERE tenant_id = ? LIMIT 1`
	err := r.db.Get(&config, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ldap config by tenant: %w", err)
	}
	return &config, nil
}

func (r *SQLiteAdvancedRepository) UpdateLDAPConfig(config *models.LDAPConfig) error {
	query := `
		UPDATE ldap_configs SET name = ?, server = ?, port = ?, use_ssl = ?, base_dn = ?,
		bind_dn = ?, bind_password = ?, user_filter = ?, group_filter = ?, sync_interval = ?,
		enabled = ?, last_sync_at = ?, updated_at = ?
		WHERE id = ?`
	config.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		config.Name, config.Server, config.Port, config.UseSSL, config.BaseDN,
		config.BindDN, config.BindPassword, config.UserFilter, config.GroupFilter,
		config.SyncInterval, config.Enabled, config.LastSyncAt, config.UpdatedAt, config.ID)
	if err != nil {
		return fmt.Errorf("failed to update ldap config: %w", err)
	}
	return nil
}

func (r *SQLiteAdvancedRepository) DeleteLDAPConfig(id string) error {
	query := `DELETE FROM ldap_configs WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteAdvancedRepository) ListLDAPConfigs(tenantID string) ([]*models.LDAPConfig, error) {
	var configs []*models.LDAPConfig
	query := `SELECT * FROM ldap_configs WHERE tenant_id = ? ORDER BY created_at DESC`
	err := r.db.Select(&configs, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list ldap configs: %w", err)
	}
	return configs, nil
}

func (r *SQLiteAdvancedRepository) TestLDAPConnection(config *models.LDAPConfig) (bool, string, error) {
	// Note: In production, this would actually test the LDAP connection
	// For now, return success as a placeholder
	return true, "LDAP connection test not implemented in demo mode", nil
}

func (r *SQLiteAdvancedRepository) SyncLDAPUsers(configID string) ([]*models.LDAPUser, error) {
	// Placeholder for LDAP user sync - in production would connect to LDAP server
	var users []*models.LDAPUser
	return users, nil
}

func (r *SQLiteAdvancedRepository) ListLDAPUsers(tenantID string, limit, offset int) ([]*models.LDAPUser, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var users []*models.LDAPUser
	query := `SELECT * FROM ldap_users WHERE tenant_id = ? ORDER BY username ASC LIMIT ? OFFSET ?`
	err := r.db.Select(&users, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ldap users: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM ldap_users WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return users, total, nil
}

// WhiteLabelConfig Methods

func (r *SQLiteAdvancedRepository) CreateWhiteLabelConfig(config *models.WhiteLabelConfig) error {
	query := `
		INSERT INTO white_label_configs (id, tenant_id, name, logo_url, favicon_url, primary_color, secondary_color, accent_color, background_color, text_color, custom_css, custom_js, footer_text, login_bg_url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		config.ID, config.TenantID, config.Name, config.LogoURL, config.FaviconURL,
		config.PrimaryColor, config.SecondaryColor, config.AccentColor, config.BackgroundColor,
		config.TextColor, config.CustomCSS, config.CustomJS, config.FooterText, config.LoginBgURL,
		config.Enabled, now, now)
	if err != nil {
		return fmt.Errorf("failed to create white label config: %w", err)
	}
	config.CreatedAt = now
	config.UpdatedAt = now
	return nil
}

func (r *SQLiteAdvancedRepository) GetWhiteLabelConfig(id string) (*models.WhiteLabelConfig, error) {
	var config models.WhiteLabelConfig
	query := `SELECT * FROM white_label_configs WHERE id = ?`
	err := r.db.Get(&config, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get white label config: %w", err)
	}
	return &config, nil
}

func (r *SQLiteAdvancedRepository) GetWhiteLabelConfigByTenant(tenantID string) (*models.WhiteLabelConfig, error) {
	var config models.WhiteLabelConfig
	query := `SELECT * FROM white_label_configs WHERE tenant_id = ? LIMIT 1`
	err := r.db.Get(&config, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get white label config by tenant: %w", err)
	}
	return &config, nil
}

func (r *SQLiteAdvancedRepository) UpdateWhiteLabelConfig(config *models.WhiteLabelConfig) error {
	query := `
		UPDATE white_label_configs SET name = ?, logo_url = ?, favicon_url = ?,
		primary_color = ?, secondary_color = ?, accent_color = ?, background_color = ?,
		text_color = ?, custom_css = ?, custom_js = ?, footer_text = ?, login_bg_url = ?,
		enabled = ?, updated_at = ?
		WHERE id = ?`
	config.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		config.Name, config.LogoURL, config.FaviconURL,
		config.PrimaryColor, config.SecondaryColor, config.AccentColor, config.BackgroundColor,
		config.TextColor, config.CustomCSS, config.CustomJS, config.FooterText, config.LoginBgURL,
		config.Enabled, config.UpdatedAt, config.ID)
	if err != nil {
		return fmt.Errorf("failed to update white label config: %w", err)
	}
	return nil
}

func (r *SQLiteAdvancedRepository) DeleteWhiteLabelConfig(id string) error {
	query := `DELETE FROM white_label_configs WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
