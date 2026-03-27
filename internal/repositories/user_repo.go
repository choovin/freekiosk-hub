package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	InitSchema(ctx interface{}) error
	// 用户管理
	Create(user *models.User) error
	GetByID(id string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.User, int64, error)
	UpdatePassword(id, passwordHash string) error
	UpdateLastLogin(id string, loginAt int64, loginIP string) error

	// 角色管理
	CreateRole(role *models.Role) error
	GetRoleByID(id string) (*models.Role, error)
	GetRoleByCode(tenantID, code string) (*models.Role, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error
	ListRoles(tenantID string) ([]*models.Role, error)

	// 会话管理
	CreateSession(session *models.UserSession) error
	GetSession(token string) (*models.UserSession, error)
	DeleteSession(token string) error
	DeleteUserSessions(userID string) error
	CleanExpiredSessions() error
}

// SQLiteUserRepository SQLite实现
type SQLiteUserRepository struct {
	db *sqlx.DB
}

// NewSQLiteUserRepository 创建用户仓库
func NewSQLiteUserRepository(db interface{}) *SQLiteUserRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteUserRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteUserRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			real_name TEXT,
			role TEXT NOT NULL DEFAULT 'viewer',
			status TEXT NOT NULL DEFAULT 'active',
			last_login_at INTEGER DEFAULT 0,
			last_login_ip TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			description TEXT,
			permissions TEXT,
			is_system INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(tenant_id, code)
		);

		CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);

		CREATE TABLE IF NOT EXISTS user_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_sessions_token ON user_sessions(token);
		CREATE INDEX IF NOT EXISTS idx_sessions_user ON user_sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires ON user_sessions(expires_at);
	`
	_, err := r.db.Exec(schema)
	return err
}

// Create 创建用户
func (r *SQLiteUserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, tenant_id, username, password_hash, email, phone, real_name, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		user.ID, user.TenantID, user.Username, user.PasswordHash,
		user.Email, user.Phone, user.RealName, user.Role, user.Status, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

// GetByID 获取用户
func (r *SQLiteUserRepository) GetByID(id string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = ?`
	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetByUsername 获取用户
func (r *SQLiteUserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE username = ?`
	err := r.db.Get(&user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetByEmail 获取用户
func (r *SQLiteUserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = ?`
	err := r.db.Get(&user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// Update 更新用户
func (r *SQLiteUserRepository) Update(user *models.User) error {
	query := `
		UPDATE users SET
			email = ?, phone = ?, real_name = ?, role = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	user.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		user.Email, user.Phone, user.RealName, user.Role, user.Status, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// Delete 删除用户
func (r *SQLiteUserRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// List 获取用户列表
func (r *SQLiteUserRepository) List(tenantID string, limit, offset int) ([]*models.User, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var users []*models.User
	query := `SELECT * FROM users WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&users, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM users WHERE tenant_id = ?`
	r.db.Get(&total, countQuery, tenantID)

	return users, total, nil
}

// UpdatePassword 更新密码
func (r *SQLiteUserRepository) UpdatePassword(id, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, passwordHash, time.Now().Unix(), id)
	return err
}

// UpdateLastLogin 更新最后登录信息
func (r *SQLiteUserRepository) UpdateLastLogin(id string, loginAt int64, loginIP string) error {
	query := `UPDATE users SET last_login_at = ?, last_login_ip = ? WHERE id = ?`
	_, err := r.db.Exec(query, loginAt, loginIP, id)
	return err
}

// CreateRole 创建角色
func (r *SQLiteUserRepository) CreateRole(role *models.Role) error {
	query := `
		INSERT INTO roles (id, tenant_id, name, code, description, permissions, is_system, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		role.ID, role.TenantID, role.Name, role.Code,
		role.Description, role.Permissions, role.IsSystem, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	role.CreatedAt = now
	role.UpdatedAt = now
	return nil
}

// GetRoleByID 获取角色
func (r *SQLiteUserRepository) GetRoleByID(id string) (*models.Role, error) {
	var role models.Role
	query := `SELECT * FROM roles WHERE id = ?`
	err := r.db.Get(&role, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// GetRoleByCode 获取角色
func (r *SQLiteUserRepository) GetRoleByCode(tenantID, code string) (*models.Role, error) {
	var role models.Role
	query := `SELECT * FROM roles WHERE tenant_id = ? AND code = ?`
	err := r.db.Get(&role, query, tenantID, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// UpdateRole 更新角色
func (r *SQLiteUserRepository) UpdateRole(role *models.Role) error {
	query := `
		UPDATE roles SET name = ?, description = ?, permissions = ?, updated_at = ?
		WHERE id = ? AND is_system = 0
	`
	role.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		role.Name, role.Description, role.Permissions, role.UpdatedAt, role.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

// DeleteRole 删除角色
func (r *SQLiteUserRepository) DeleteRole(id string) error {
	query := `DELETE FROM roles WHERE id = ? AND is_system = 0`
	_, err := r.db.Exec(query, id)
	return err
}

// ListRoles 获取角色列表
func (r *SQLiteUserRepository) ListRoles(tenantID string) ([]*models.Role, error) {
	var roles []*models.Role
	query := `SELECT * FROM roles WHERE tenant_id = ? ORDER BY created_at DESC`
	err := r.db.Select(&roles, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

// CreateSession 创建会话
func (r *SQLiteUserRepository) CreateSession(session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	session.CreatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSession 获取会话
func (r *SQLiteUserRepository) GetSession(token string) (*models.UserSession, error) {
	var session models.UserSession
	query := `SELECT * FROM user_sessions WHERE token = ?`
	err := r.db.Get(&session, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

// DeleteSession 删除会话
func (r *SQLiteUserRepository) DeleteSession(token string) error {
	query := `DELETE FROM user_sessions WHERE token = ?`
	_, err := r.db.Exec(query, token)
	return err
}

// DeleteUserSessions 删除用户的所有会话
func (r *SQLiteUserRepository) DeleteUserSessions(userID string) error {
	query := `DELETE FROM user_sessions WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

// CleanExpiredSessions 清理过期会话
func (r *SQLiteUserRepository) CleanExpiredSessions() error {
	query := `DELETE FROM user_sessions WHERE expires_at < ?`
	_, err := r.db.Exec(query, time.Now().Unix())
	return err
}
