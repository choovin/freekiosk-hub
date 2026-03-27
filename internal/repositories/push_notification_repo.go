package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// PushNotificationRepository 推送通知仓库接口
type PushNotificationRepository interface {
	InitSchema(ctx interface{}) error
	// 通知管理
	Create(notification *models.PushNotification) error
	GetByID(id string) (*models.PushNotification, error)
	Update(notification *models.PushNotification) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.PushNotification, int64, error)
	ListByDevice(deviceID string, limit, offset int) ([]*models.PushNotification, int64, error)
	ListScheduled(tenantID string) ([]*models.PushNotification, error)
	ListByGroup(groupID string, limit, offset int) ([]*models.PushNotification, int64, error)

	// 回执管理
	SaveReceipt(receipt *models.PushNotificationReceipt) error
	GetReceipts(notificationID string) ([]*models.PushNotificationReceipt, error)
	UpdateReceipt(receipt *models.PushNotificationReceipt) error
	GetDeviceReceipts(deviceID string, limit, offset int) ([]*models.PushNotificationReceipt, int64, error)
}

// SQLitePushNotificationRepository SQLite实现
type SQLitePushNotificationRepository struct {
	db *sqlx.DB
}

// NewSQLitePushNotificationRepository 创建推送通知仓库
func NewSQLitePushNotificationRepository(db interface{}) *SQLitePushNotificationRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLitePushNotificationRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLitePushNotificationRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS push_notifications (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT,
			group_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			priority TEXT NOT NULL DEFAULT 'normal',
			type TEXT NOT NULL DEFAULT 'info',
			actions TEXT,
			scheduled_at INTEGER DEFAULT 0,
			expired_at INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			sent_at INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_push_notifications_tenant ON push_notifications(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_push_notifications_device ON push_notifications(device_id);
		CREATE INDEX IF NOT EXISTS idx_push_notifications_group ON push_notifications(group_id);
		CREATE INDEX IF NOT EXISTS idx_push_notifications_status ON push_notifications(status);
		CREATE INDEX IF NOT EXISTS idx_push_notifications_scheduled ON push_notifications(scheduled_at);

		CREATE TABLE IF NOT EXISTS push_notification_receipts (
			id TEXT PRIMARY KEY,
			notification_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'sent',
			delivered_at INTEGER DEFAULT 0,
			read_at INTEGER DEFAULT 0,
			error_message TEXT,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (notification_id) REFERENCES push_notifications(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_push_receipts_notification ON push_notification_receipts(notification_id);
		CREATE INDEX IF NOT EXISTS idx_push_receipts_device ON push_notification_receipts(device_id);
	`
	_, err := r.db.Exec(schema)
	return err
}

// Create 创建推送通知
func (r *SQLitePushNotificationRepository) Create(notification *models.PushNotification) error {
	query := `
		INSERT INTO push_notifications (
			id, tenant_id, device_id, group_id, title, content, priority, type,
			actions, scheduled_at, expired_at, status, sent_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		notification.ID, notification.TenantID, notification.DeviceID, notification.GroupID,
		notification.Title, notification.Content, notification.Priority, notification.Type,
		notification.Actions, notification.ScheduledAt, notification.ExpiredAt,
		notification.Status, notification.SentAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create push notification: %w", err)
	}
	notification.CreatedAt = now
	notification.UpdatedAt = now
	return nil
}

// GetByID 获取推送通知
func (r *SQLitePushNotificationRepository) GetByID(id string) (*models.PushNotification, error) {
	var notification models.PushNotification
	query := `SELECT * FROM push_notifications WHERE id = ?`
	err := r.db.Get(&notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get push notification: %w", err)
	}
	return &notification, nil
}

// Update 更新推送通知
func (r *SQLitePushNotificationRepository) Update(notification *models.PushNotification) error {
	query := `
		UPDATE push_notifications SET
			device_id = ?, group_id = ?, title = ?, content = ?,
			priority = ?, type = ?, actions = ?, scheduled_at = ?,
			expired_at = ?, status = ?, sent_at = ?, updated_at = ?
		WHERE id = ?
	`
	notification.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		notification.DeviceID, notification.GroupID, notification.Title, notification.Content,
		notification.Priority, notification.Type, notification.Actions, notification.ScheduledAt,
		notification.ExpiredAt, notification.Status, notification.SentAt,
		notification.UpdatedAt, notification.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update push notification: %w", err)
	}
	return nil
}

// Delete 删除推送通知
func (r *SQLitePushNotificationRepository) Delete(id string) error {
	query := `DELETE FROM push_notifications WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// List 获取租户的通知列表
func (r *SQLitePushNotificationRepository) List(tenantID string, limit, offset int) ([]*models.PushNotification, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var notifications []*models.PushNotification
	query := `SELECT * FROM push_notifications WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&notifications, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list push notifications: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM push_notifications WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return notifications, total, nil
}

// ListByDevice 获取设备的通知列表
func (r *SQLitePushNotificationRepository) ListByDevice(deviceID string, limit, offset int) ([]*models.PushNotification, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var notifications []*models.PushNotification
	query := `
		SELECT * FROM push_notifications
		WHERE device_id = ? AND status IN ('sent', 'delivered', 'read')
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`
	err := r.db.Select(&notifications, query, deviceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list device push notifications: %w", err)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM push_notifications
		WHERE device_id = ? AND status IN ('sent', 'delivered', 'read')
	`
	r.db.Get(&total, countQuery, deviceID)

	return notifications, total, nil
}

// ListScheduled 获取计划发送的通知
func (r *SQLitePushNotificationRepository) ListScheduled(tenantID string) ([]*models.PushNotification, error) {
	var notifications []*models.PushNotification
	now := time.Now().Unix()
	query := `
		SELECT * FROM push_notifications
		WHERE tenant_id = ? AND status = 'pending' AND scheduled_at > 0 AND scheduled_at <= ?
		ORDER BY scheduled_at ASC
	`
	err := r.db.Select(&notifications, query, tenantID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled notifications: %w", err)
	}
	return notifications, nil
}

// ListByGroup 获取设备组的通知列表
func (r *SQLitePushNotificationRepository) ListByGroup(groupID string, limit, offset int) ([]*models.PushNotification, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var notifications []*models.PushNotification
	query := `SELECT * FROM push_notifications WHERE group_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&notifications, query, groupID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list group push notifications: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM push_notifications WHERE group_id = ?`
	r.db.Get(&total, countQuery, groupID)

	return notifications, total, nil
}

// SaveReceipt 保存回执
func (r *SQLitePushNotificationRepository) SaveReceipt(receipt *models.PushNotificationReceipt) error {
	query := `
		INSERT INTO push_notification_receipts (id, notification_id, device_id, status, delivered_at, read_at, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		receipt.ID, receipt.NotificationID, receipt.DeviceID, receipt.Status,
		receipt.DeliveredAt, receipt.ReadAt, receipt.ErrorMessage, now,
	)
	if err != nil {
		return fmt.Errorf("failed to save push receipt: %w", err)
	}
	receipt.CreatedAt = now
	return nil
}

// GetReceipts 获取通知的所有回执
func (r *SQLitePushNotificationRepository) GetReceipts(notificationID string) ([]*models.PushNotificationReceipt, error) {
	var receipts []*models.PushNotificationReceipt
	query := `SELECT * FROM push_notification_receipts WHERE notification_id = ?`
	err := r.db.Select(&receipts, query, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts: %w", err)
	}
	return receipts, nil
}

// UpdateReceipt 更新回执
func (r *SQLitePushNotificationRepository) UpdateReceipt(receipt *models.PushNotificationReceipt) error {
	query := `
		UPDATE push_notification_receipts SET
			status = ?, delivered_at = ?, read_at = ?, error_message = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		receipt.Status, receipt.DeliveredAt, receipt.ReadAt, receipt.ErrorMessage, receipt.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update receipt: %w", err)
	}
	return nil
}

// GetDeviceReceipts 获取设备的回执列表
func (r *SQLitePushNotificationRepository) GetDeviceReceipts(deviceID string, limit, offset int) ([]*models.PushNotificationReceipt, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var receipts []*models.PushNotificationReceipt
	query := `SELECT * FROM push_notification_receipts WHERE device_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&receipts, query, deviceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get device receipts: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM push_notification_receipts WHERE device_id = ?`
	r.db.Get(&total, countQuery, deviceID)

	return receipts, total, nil
}
