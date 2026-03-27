package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// RemoteControlRepository 远程控制仓库接口
type RemoteControlRepository interface {
	InitSchema(ctx interface{}) error
	// 会话管理
	CreateSession(session *models.RemoteSession) error
	GetSession(id string) (*models.RemoteSession, error)
	UpdateSession(session *models.RemoteSession) error
	DeleteSession(id string) error
	ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error)
	GetActiveSession(deviceID string) (*models.RemoteSession, error)

	// 事件记录
	RecordEvent(event *models.RemoteSessionEvent) error
	GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error)

	// 屏幕截图
	SaveScreenCapture(capture *models.ScreenCapture) error
	GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error)

	// 命令记录
	SaveCommand(cmd *models.RemoteCommand) error
	UpdateCommandStatus(id, status, response string) error
	GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error)
}

// SQLiteRemoteControlRepository SQLite实现
type SQLiteRemoteControlRepository struct {
	db *sqlx.DB
}

// NewSQLiteRemoteControlRepository 创建远程控制仓库
func NewSQLiteRemoteControlRepository(db interface{}) *SQLiteRemoteControlRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteRemoteControlRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteRemoteControlRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS remote_sessions (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			initiator_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			session_type TEXT NOT NULL DEFAULT 'view',
			ice_servers TEXT,
			started_at INTEGER,
			ended_at INTEGER,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_remote_sessions_device ON remote_sessions(device_id);
		CREATE INDEX IF NOT EXISTS idx_remote_sessions_tenant ON remote_sessions(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_remote_sessions_status ON remote_sessions(status);

		CREATE TABLE IF NOT EXISTS remote_session_events (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			message TEXT,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (session_id) REFERENCES remote_sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_remote_session_events_session ON remote_session_events(session_id);

		CREATE TABLE IF NOT EXISTS screen_captures (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			mime_type TEXT NOT NULL DEFAULT 'image/png',
			captured_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (session_id) REFERENCES remote_sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_screen_captures_session ON screen_captures(session_id);
		CREATE INDEX IF NOT EXISTS idx_screen_captures_device ON screen_captures(device_id);

		CREATE TABLE IF NOT EXISTS remote_commands (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			command_type TEXT NOT NULL,
			params TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			response TEXT,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (session_id) REFERENCES remote_sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_remote_commands_session ON remote_commands(session_id);
		CREATE INDEX IF NOT EXISTS idx_remote_commands_status ON remote_commands(status);
	`
	_, err := r.db.Exec(schema)
	return err
}

// CreateSession 创建远程会话
func (r *SQLiteRemoteControlRepository) CreateSession(session *models.RemoteSession) error {
	query := `
		INSERT INTO remote_sessions (
			id, device_id, tenant_id, initiator_id, status, session_type,
			ice_servers, started_at, ended_at, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		session.ID, session.DeviceID, session.TenantID, session.InitiatorID,
		session.Status, session.SessionType, session.ICEServers,
		session.StartedAt, session.EndedAt, session.ExpiresAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create remote session: %w", err)
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	return nil
}

// GetSession 获取会话
func (r *SQLiteRemoteControlRepository) GetSession(id string) (*models.RemoteSession, error) {
	var session models.RemoteSession
	query := `SELECT * FROM remote_sessions WHERE id = ?`
	err := r.db.Get(&session, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get remote session: %w", err)
	}
	return &session, nil
}

// UpdateSession 更新会话
func (r *SQLiteRemoteControlRepository) UpdateSession(session *models.RemoteSession) error {
	query := `
		UPDATE remote_sessions SET
			status = ?, session_type = ?, ice_servers = ?,
			started_at = ?, ended_at = ?, expires_at = ?,
			updated_at = ?
		WHERE id = ?
	`
	session.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		session.Status, session.SessionType, session.ICEServers,
		session.StartedAt, session.EndedAt, session.ExpiresAt,
		session.UpdatedAt, session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update remote session: %w", err)
	}
	return nil
}

// DeleteSession 删除会话
func (r *SQLiteRemoteControlRepository) DeleteSession(id string) error {
	query := `DELETE FROM remote_sessions WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// ListDeviceSessions 获取设备的会话列表
func (r *SQLiteRemoteControlRepository) ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var sessions []*models.RemoteSession
	query := `SELECT * FROM remote_sessions WHERE device_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&sessions, query, deviceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list device sessions: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM remote_sessions WHERE device_id = ?`
	r.db.Get(&total, countQuery, deviceID)

	return sessions, total, nil
}

// GetActiveSession 获取设备的活跃会话
func (r *SQLiteRemoteControlRepository) GetActiveSession(deviceID string) (*models.RemoteSession, error) {
	var session models.RemoteSession
	query := `SELECT * FROM remote_sessions WHERE device_id = ? AND status = 'active' ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&session, query, deviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}
	return &session, nil
}

// RecordEvent 记录会话事件
func (r *SQLiteRemoteControlRepository) RecordEvent(event *models.RemoteSessionEvent) error {
	query := `
		INSERT INTO remote_session_events (id, session_id, device_id, tenant_id, event_type, message, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		event.ID, event.SessionID, event.DeviceID, event.TenantID,
		event.EventType, event.Message, event.Timestamp, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record session event: %w", err)
	}
	event.CreatedAt = now
	return nil
}

// GetSessionEvents 获取会话的所有事件
func (r *SQLiteRemoteControlRepository) GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error) {
	var events []*models.RemoteSessionEvent
	query := `SELECT * FROM remote_session_events WHERE session_id = ? ORDER BY timestamp DESC`
	err := r.db.Select(&events, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session events: %w", err)
	}
	return events, nil
}

// SaveScreenCapture 保存屏幕截图
func (r *SQLiteRemoteControlRepository) SaveScreenCapture(capture *models.ScreenCapture) error {
	query := `
		INSERT INTO screen_captures (id, session_id, device_id, tenant_id, file_path, file_size, mime_type, captured_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		capture.ID, capture.SessionID, capture.DeviceID, capture.TenantID,
		capture.FilePath, capture.FileSize, capture.MimeType, capture.CapturedAt, now,
	)
	if err != nil {
		return fmt.Errorf("failed to save screen capture: %w", err)
	}
	capture.CreatedAt = now
	return nil
}

// GetSessionScreenCaptures 获取会话的屏幕截图
func (r *SQLiteRemoteControlRepository) GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error) {
	var captures []*models.ScreenCapture
	query := `SELECT * FROM screen_captures WHERE session_id = ? ORDER BY captured_at DESC`
	err := r.db.Select(&captures, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session screen captures: %w", err)
	}
	return captures, nil
}

// SaveCommand 保存命令
func (r *SQLiteRemoteControlRepository) SaveCommand(cmd *models.RemoteCommand) error {
	query := `
		INSERT INTO remote_commands (id, session_id, device_id, tenant_id, command_type, params, status, response, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		cmd.ID, cmd.SessionID, cmd.DeviceID, cmd.TenantID,
		cmd.CommandType, cmd.Params, cmd.Status, cmd.Response, cmd.Timestamp, now,
	)
	if err != nil {
		return fmt.Errorf("failed to save remote command: %w", err)
	}
	cmd.CreatedAt = now
	return nil
}

// UpdateCommandStatus 更新命令状态
func (r *SQLiteRemoteControlRepository) UpdateCommandStatus(id, status, response string) error {
	query := `UPDATE remote_commands SET status = ?, response = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, response, id)
	return err
}

// GetSessionCommands 获取会话的所有命令
func (r *SQLiteRemoteControlRepository) GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error) {
	var commands []*models.RemoteCommand
	query := `SELECT * FROM remote_commands WHERE session_id = ? ORDER BY timestamp DESC`
	err := r.db.Select(&commands, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session commands: %w", err)
	}
	return commands, nil
}
