package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// UserService 用户服务接口
type UserService interface {
	// 用户管理
	CreateUser(tenantID, username, password, email, role string) (*models.User, error)
	GetUser(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id string) error
	ListUsers(tenantID string, limit, offset int) ([]*models.User, int64, error)
	ChangePassword(userID, oldPassword, newPassword string) error
	ResetPassword(userID, newPassword string) error

	// 认证
	Login(username, password string) (*models.User, string, error)
	Logout(token string) error
	ValidateToken(token string) (*models.User, error)

	// 角色管理
	CreateRole(tenantID, name, code, description string, permissions []string) (*models.Role, error)
	GetRole(id string) (*models.Role, error)
	GetRoleByCode(tenantID, code string) (*models.Role, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error
	ListRoles(tenantID string) ([]*models.Role, error)
	HasPermission(userID string, permission string) (bool, error)
}

// DefaultUserService 默认用户服务实现
type DefaultUserService struct {
	repo UserRepository
}

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.User, int64, error)
	UpdatePassword(id, passwordHash string) error
	UpdateLastLogin(id string, loginAt int64, loginIP string) error
	CreateSession(session *models.UserSession) error
	GetSession(token string) (*models.UserSession, error)
	DeleteSession(token string) error
	DeleteUserSessions(userID string) error
	CreateRole(role *models.Role) error
	GetRoleByID(id string) (*models.Role, error)
	GetRoleByCode(tenantID, code string) (*models.Role, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error
	ListRoles(tenantID string) ([]*models.Role, error)
}

// NewUserService 创建用户服务
func NewUserService(repo UserRepository) *DefaultUserService {
	return &DefaultUserService{repo: repo}
}

// CreateUser 创建用户
func (s *DefaultUserService) CreateUser(tenantID, username, password, email, role string) (*models.User, error) {
	if role == "" {
		role = "viewer"
	}

	// Check if username already exists
	existing, _ := s.repo.GetByUsername(username)
	if existing != nil {
		return nil, fmt.Errorf("username already exists")
	}

	now := time.Now().Unix()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Username:     username,
		PasswordHash: HashPassword(password), // In production, use proper bcrypt
		Email:        email,
		Role:         role,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser 获取用户
func (s *DefaultUserService) GetUser(id string) (*models.User, error) {
	return s.repo.GetByID(id)
}

// GetUserByUsername 获取用户
func (s *DefaultUserService) GetUserByUsername(username string) (*models.User, error) {
	return s.repo.GetByUsername(username)
}

// UpdateUser 更新用户
func (s *DefaultUserService) UpdateUser(user *models.User) error {
	return s.repo.Update(user)
}

// DeleteUser 删除用户
func (s *DefaultUserService) DeleteUser(id string) error {
	// Delete all sessions first
	if err := s.repo.DeleteUserSessions(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// ListUsers 获取用户列表
func (s *DefaultUserService) ListUsers(tenantID string, limit, offset int) ([]*models.User, int64, error) {
	return s.repo.List(tenantID, limit, offset)
}

// ChangePassword 修改密码
func (s *DefaultUserService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	if !CheckPasswordHash(oldPassword, user.PasswordHash) {
		return fmt.Errorf("invalid old password")
	}

	newHash := HashPassword(newPassword)
	return s.repo.UpdatePassword(userID, newHash)
}

// ResetPassword 重置密码
func (s *DefaultUserService) ResetPassword(userID, newPassword string) error {
	newHash := HashPassword(newPassword)
	return s.repo.UpdatePassword(userID, newHash)
}

// Login 登录
func (s *DefaultUserService) Login(username, password string) (*models.User, string, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", fmt.Errorf("invalid username or password")
	}

	if user.Status != "active" {
		return nil, "", fmt.Errorf("account is not active")
	}

	// Verify password
	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid username or password")
	}

	// Create session
	token := GenerateToken()
	session := &models.UserSession{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, "", err
	}

	// Update last login
	s.repo.UpdateLastLogin(user.ID, time.Now().Unix(), "")

	return user, token, nil
}

// Logout 登出
func (s *DefaultUserService) Logout(token string) error {
	return s.repo.DeleteSession(token)
}

// ValidateToken 验证令牌
func (s *DefaultUserService) ValidateToken(token string) (*models.User, error) {
	session, err := s.repo.GetSession(token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	if session.ExpiresAt < time.Now().Unix() {
		s.repo.DeleteSession(token)
		return nil, fmt.Errorf("session expired")
	}

	return s.repo.GetByID(session.UserID)
}

// CreateRole 创建角色
func (s *DefaultUserService) CreateRole(tenantID, name, code, description string, permissions []string) (*models.Role, error) {
	permissionsJSON, _ := json.Marshal(permissions)

	now := time.Now().Unix()
	role := &models.Role{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        name,
		Code:        code,
		Description: description,
		Permissions: string(permissionsJSON),
		IsSystem:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRole(role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetRole 获取角色
func (s *DefaultUserService) GetRole(id string) (*models.Role, error) {
	return s.repo.GetRoleByID(id)
}

// GetRoleByCode 获取角色
func (s *DefaultUserService) GetRoleByCode(tenantID, code string) (*models.Role, error) {
	return s.repo.GetRoleByCode(tenantID, code)
}

// UpdateRole 更新角色
func (s *DefaultUserService) UpdateRole(role *models.Role) error {
	return s.repo.UpdateRole(role)
}

// DeleteRole 删除角色
func (s *DefaultUserService) DeleteRole(id string) error {
	return s.repo.DeleteRole(id)
}

// ListRoles 获取角色列表
func (s *DefaultUserService) ListRoles(tenantID string) ([]*models.Role, error) {
	return s.repo.ListRoles(tenantID)
}

// HasPermission 检查权限
func (s *DefaultUserService) HasPermission(userID string, permission string) (bool, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	// Admin has all permissions
	if user.Role == "admin" {
		return true, nil
	}

	// Look up role by tenant ID and role code
	role, err := s.repo.GetRoleByCode(user.TenantID, user.Role)
	if err != nil || role == nil {
		return false, nil
	}

	var permissions []string
	if err := json.Unmarshal([]byte(role.Permissions), &permissions); err != nil {
		return false, nil
	}

	for _, p := range permissions {
		if p == permission || p == "*" {
			return true, nil
		}
	}

	return false, nil
}

// HashPassword 密码哈希 (简化版，实际应使用bcrypt)
func HashPassword(password string) string {
	// In production, use bcrypt or argon2
	return fmt.Sprintf("hash_%s", password)
}

// CheckPasswordHash 验证密码哈希
func CheckPasswordHash(password, hash string) bool {
	// In production, use bcrypt.CompareHashAndPassword
	return fmt.Sprintf("hash_%s", password) == hash
}

// GenerateToken 生成令牌
func GenerateToken() string {
	return uuid.New().String()
}
