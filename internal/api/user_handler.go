package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// UserHandler 用户处理器
type UserHandler struct {
	svc services.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(svc services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(g *echo.Group) {
	// 用户管理
	g.POST("/users", h.CreateUser)
	g.GET("/users", h.ListUsers)
	g.GET("/users/:id", h.GetUser)
	g.PUT("/users/:id", h.UpdateUser)
	g.DELETE("/users/:id", h.DeleteUser)
	g.POST("/users/:id/password", h.ChangePassword)
	g.POST("/users/:id/reset-password", h.ResetPassword)

	// 认证
	g.POST("/auth/login", h.Login)
	g.POST("/auth/logout", h.Logout)
	g.GET("/auth/me", h.Me)
	g.POST("/auth/validate", h.ValidateToken)

	// 角色管理
	g.POST("/roles", h.CreateRole)
	g.GET("/roles", h.ListRoles)
	g.GET("/roles/:id", h.GetRole)
	g.PUT("/roles/:id", h.UpdateRole)
	g.DELETE("/roles/:id", h.DeleteRole)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
	}
	if req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password is required"})
	}

	user, err := h.svc.CreateUser(tenantID, req.Username, req.Password, req.Email, req.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, user)
}

// GetUser 获取用户
func (h *UserHandler) GetUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user id is required"})
	}

	user, err := h.svc.GetUser(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	RealName string `json:"real_name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user id is required"})
	}

	user, err := h.svc.GetUser(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.RealName != "" {
		user.RealName = req.RealName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := h.svc.UpdateUser(user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user id is required"})
	}

	if err := h.svc.DeleteUser(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "user deleted"})
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	users, total, err := h.svc.ListUsers(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user id is required"})
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.OldPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "old_password is required"})
	}
	if req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new_password is required"})
	}

	if err := h.svc.ChangePassword(id, req.OldPassword, req.NewPassword); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "password changed"})
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// ResetPassword 重置密码
func (h *UserHandler) ResetPassword(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user id is required"})
	}

	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new_password is required"})
	}

	if err := h.svc.ResetPassword(id, req.NewPassword); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "password reset"})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 登录
func (h *UserHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
	}
	if req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password is required"})
	}

	user, token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// Logout 登出
func (h *UserHandler) Logout(c echo.Context) error {
	token := c.Request().Header.Get("Authorization")
	if token != "" {
		h.svc.Logout(token)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// Me 获取当前用户
func (h *UserHandler) Me(c echo.Context) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}

	user, err := h.svc.ValidateToken(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

// ValidateTokenRequest 验证令牌请求
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateToken 验证令牌
func (h *UserHandler) ValidateToken(c echo.Context) error {
	var req ValidateTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := h.svc.ValidateToken(req.Token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// CreateRole 创建角色
func (h *UserHandler) CreateRole(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "code is required"})
	}

	role, err := h.svc.CreateRole(tenantID, req.Name, req.Code, req.Description, req.Permissions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, role)
}

// GetRole 获取角色
func (h *UserHandler) GetRole(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role id is required"})
	}

	role, err := h.svc.GetRole(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "role not found"})
	}

	return c.JSON(http.StatusOK, role)
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRole 更新角色
func (h *UserHandler) UpdateRole(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role id is required"})
	}

	role, err := h.svc.GetRole(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "role not found"})
	}

	var req UpdateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Permissions != nil {
		role.Permissions = mustMarshalJSON(req.Permissions)
	}

	if err := h.svc.UpdateRole(role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, role)
}

// DeleteRole 删除角色
func (h *UserHandler) DeleteRole(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role id is required"})
	}

	if err := h.svc.DeleteRole(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role deleted"})
}

// ListRoles 获取角色列表
func (h *UserHandler) ListRoles(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	roles, err := h.svc.ListRoles(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, roles)
}

func mustMarshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
