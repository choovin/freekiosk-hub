package services

import (
	"testing"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// MockUserRepository 测试用的模拟用户仓库
type MockUserRepository struct {
	users    map[string]*models.User
	roles    map[string]*models.Role
	sessions map[string]*models.UserSession
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:    make(map[string]*models.User),
		roles:    make(map[string]*models.Role),
		sessions: make(map[string]*models.UserSession),
	}
}

func (m *MockUserRepository) InitSchema(ctx interface{}) error {
	return nil
}

func (m *MockUserRepository) Create(user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByID(id string) (*models.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) List(tenantID string, limit, offset int) ([]*models.User, int64, error) {
	var result []*models.User
	for _, user := range m.users {
		if user.TenantID == tenantID {
			result = append(result, user)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockUserRepository) UpdatePassword(id, passwordHash string) error {
	if user, ok := m.users[id]; ok {
		user.PasswordHash = passwordHash
	}
	return nil
}

func (m *MockUserRepository) UpdateLastLogin(id string, loginAt int64, loginIP string) error {
	if user, ok := m.users[id]; ok {
		user.LastLoginAt = loginAt
		user.LastLoginIP = loginIP
	}
	return nil
}

func (m *MockUserRepository) CreateSession(session *models.UserSession) error {
	m.sessions[session.Token] = session
	return nil
}

func (m *MockUserRepository) GetSession(token string) (*models.UserSession, error) {
	session, ok := m.sessions[token]
	if !ok {
		return nil, nil
	}
	return session, nil
}

func (m *MockUserRepository) DeleteSession(token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *MockUserRepository) DeleteUserSessions(userID string) error {
	for token, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}

func (m *MockUserRepository) CleanExpiredSessions() error {
	return nil
}

func (m *MockUserRepository) CreateRole(role *models.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *MockUserRepository) GetRoleByID(id string) (*models.Role, error) {
	role, ok := m.roles[id]
	if !ok {
		return nil, nil
	}
	return role, nil
}

func (m *MockUserRepository) GetRoleByCode(tenantID, code string) (*models.Role, error) {
	for _, role := range m.roles {
		if role.TenantID == tenantID && role.Code == code {
			return role, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) UpdateRole(role *models.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *MockUserRepository) DeleteRole(id string) error {
	delete(m.roles, id)
	return nil
}

func (m *MockUserRepository) ListRoles(tenantID string) ([]*models.Role, error) {
	var result []*models.Role
	for _, role := range m.roles {
		if role.TenantID == tenantID {
			result = append(result, role)
		}
	}
	return result, nil
}

// Tests

func TestUserService_CreateUser(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, err := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == "" {
		t.Error("User ID should be set")
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", user.Username)
	}

	if user.TenantID != "tenant-1" {
		t.Errorf("Expected tenant_id tenant-1, got %s", user.TenantID)
	}

	if user.Role != "viewer" {
		t.Errorf("Expected role viewer, got %s", user.Role)
	}

	if user.Status != "active" {
		t.Errorf("Expected status active, got %s", user.Status)
	}
}

func TestUserService_CreateUser_DuplicateUsername(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	_, err = svc.CreateUser("tenant-1", "testuser", "password456", "test2@example.com", "viewer")
	if err == nil {
		t.Error("Expected error for duplicate username, got nil")
	}
}

func TestUserService_GetUser(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	created, _ := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	got, err := svc.GetUser(created.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if got == nil {
		t.Fatal("GetUser returned nil")
	}

	if got.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", got.Username)
	}
}

func TestUserService_GetUserByUsername(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	got, err := svc.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if got == nil {
		t.Fatal("GetUserByUsername returned nil")
	}

	if got.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", got.Username)
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	user.Email = "newemail@example.com"
	user.Role = "admin"

	err := svc.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	got, _ := svc.GetUser(user.ID)
	if got.Email != "newemail@example.com" {
		t.Errorf("Expected email newemail@example.com, got %s", got.Email)
	}

	if got.Role != "admin" {
		t.Errorf("Expected role admin, got %s", got.Role)
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	err := svc.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	got, _ := svc.GetUser(user.ID)
	if got != nil {
		t.Error("Expected nil after delete")
	}
}

func TestUserService_ListUsers(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateUser("tenant-1", "user1", "password", "user1@example.com", "viewer")
	_, _ = svc.CreateUser("tenant-1", "user2", "password", "user2@example.com", "viewer")
	_, _ = svc.CreateUser("tenant-2", "user3", "password", "user3@example.com", "viewer")

	users, total, err := svc.ListUsers("tenant-1", 10, 0)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestUserService_Login(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	user, token, err := svc.Login("testuser", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if user == nil {
		t.Fatal("Login returned nil user")
	}

	if token == "" {
		t.Error("Login returned empty token")
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", user.Username)
	}
}

func TestUserService_Login_InvalidPassword(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	_, _, err := svc.Login("testuser", "wrongpassword")
	if err == nil {
		t.Error("Expected error for invalid password, got nil")
	}
}

func TestUserService_Login_InactiveUser(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")
	user.Status = "inactive"
	svc.UpdateUser(user)

	_, _, err := svc.Login("testuser", "password123")
	if err == nil {
		t.Error("Expected error for inactive user, got nil")
	}
}

func TestUserService_Logout(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, token, _ := svc.Login("testuser", "password123")
	_, _ = svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	err := svc.Logout(token)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Error("Expected error after logout")
	}
}

func TestUserService_ValidateToken(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")
	user, token, _ := svc.Login("testuser", "password123")

	validated, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if validated.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, validated.ID)
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "oldpassword", "test@example.com", "viewer")

	err := svc.ChangePassword(user.ID, "oldpassword", "newpassword")
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// Login with new password should work
	_, _, err = svc.Login("testuser", "newpassword")
	if err != nil {
		t.Errorf("Login with new password failed: %v", err)
	}
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "oldpassword", "test@example.com", "viewer")

	err := svc.ChangePassword(user.ID, "wrongpassword", "newpassword")
	if err == nil {
		t.Error("Expected error for wrong old password")
	}
}

func TestUserService_ResetPassword(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "password123", "test@example.com", "viewer")

	err := svc.ResetPassword(user.ID, "resetpassword")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// Login with reset password should work
	_, _, err = svc.Login("testuser", "resetpassword")
	if err != nil {
		t.Errorf("Login with reset password failed: %v", err)
	}
}

func TestUserService_CreateRole(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	role, err := svc.CreateRole("tenant-1", "Test Role", "test_role", "A test role", []string{"read", "write"})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	if role.ID == "" {
		t.Error("Role ID should be set")
	}

	if role.Name != "Test Role" {
		t.Errorf("Expected name 'Test Role', got %s", role.Name)
	}

	if role.TenantID != "tenant-1" {
		t.Errorf("Expected tenant_id tenant-1, got %s", role.TenantID)
	}

	if role.IsSystem {
		t.Error("Custom role should not be a system role")
	}
}

func TestUserService_GetRole(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	created, _ := svc.CreateRole("tenant-1", "Test Role", "test_role", "A test role", []string{"read"})

	got, err := svc.GetRole(created.ID)
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}

	if got == nil {
		t.Fatal("GetRole returned nil")
	}

	if got.Name != "Test Role" {
		t.Errorf("Expected name 'Test Role', got %s", got.Name)
	}
}

func TestUserService_GetRoleByCode(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateRole("tenant-1", "Test Role", "test_role", "A test role", []string{"read"})

	got, err := svc.GetRoleByCode("tenant-1", "test_role")
	if err != nil {
		t.Fatalf("GetRoleByCode failed: %v", err)
	}

	if got == nil {
		t.Fatal("GetRoleByCode returned nil")
	}

	if got.Name != "Test Role" {
		t.Errorf("Expected name 'Test Role', got %s", got.Name)
	}
}

func TestUserService_UpdateRole(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	role, _ := svc.CreateRole("tenant-1", "Test Role", "test_role", "A test role", []string{"read"})

	role.Name = "Updated Role"
	role.Description = "Updated description"

	err := svc.UpdateRole(role)
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	got, _ := svc.GetRole(role.ID)
	if got.Name != "Updated Role" {
		t.Errorf("Expected name 'Updated Role', got %s", got.Name)
	}
}

func TestUserService_DeleteRole(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	role, _ := svc.CreateRole("tenant-1", "Test Role", "test_role", "A test role", []string{"read"})

	err := svc.DeleteRole(role.ID)
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	got, _ := svc.GetRole(role.ID)
	if got != nil {
		t.Error("Expected nil after delete")
	}
}

func TestUserService_ListRoles(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	_, _ = svc.CreateRole("tenant-1", "Role 1", "role_1", "Role 1", []string{"read"})
	_, _ = svc.CreateRole("tenant-1", "Role 2", "role_2", "Role 2", []string{"write"})
	_, _ = svc.CreateRole("tenant-2", "Role 3", "role_3", "Role 3", []string{"admin"})

	roles, err := svc.ListRoles("tenant-1")
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}

	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}
}

func TestUserService_HasPermission(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "testuser", "password", "test@example.com", "viewer")

	// Create a role with specific permissions
	role, _ := svc.CreateRole("tenant-1", "Manager", "manager", "Manager role", []string{"read", "write", "delete"})
	user.Role = "manager"
	repo.Update(user)

	// Need to create the role in the repo as well for the permission check
	repo.roles[role.ID] = role

	hasRead, err := svc.HasPermission(user.ID, "read")
	if err != nil {
		t.Fatalf("HasPermission failed: %v", err)
	}
	if !hasRead {
		t.Error("Expected to have read permission")
	}

	hasDelete, err := svc.HasPermission(user.ID, "delete")
	if err != nil {
		t.Fatalf("HasPermission failed: %v", err)
	}
	if !hasDelete {
		t.Error("Expected to have delete permission")
	}

	hasAdmin, err := svc.HasPermission(user.ID, "admin")
	if err != nil {
		t.Fatalf("HasPermission failed: %v", err)
	}
	if hasAdmin {
		t.Error("Expected to NOT have admin permission")
	}
}

func TestUserService_HasPermission_AdminRole(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	user, _ := svc.CreateUser("tenant-1", "adminuser", "password", "admin@example.com", "admin")

	// Admin should have all permissions
	hasAny, err := svc.HasPermission(user.ID, "anything")
	if err != nil {
		t.Fatalf("HasPermission failed: %v", err)
	}
	if !hasAny {
		t.Error("Admin should have any permission")
	}
}
