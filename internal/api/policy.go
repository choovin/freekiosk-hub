package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// PolicyHandler 安全策略 API 处理器
type PolicyHandler struct {
	policySvc services.PolicyService
}

// NewPolicyHandler 创建策略处理器
func NewPolicyHandler(policySvc services.PolicyService) *PolicyHandler {
	return &PolicyHandler{policySvc: policySvc}
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Settings    *models.SecurityPolicySettings `json:"settings,omitempty"`
	AppWhitelist []models.AppWhitelistEntry `json:"app_whitelist,omitempty"`
}

// UpdatePolicyRequest 更新策略请求
type UpdatePolicyRequest struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Settings    *models.SecurityPolicySettings `json:"settings,omitempty"`
	AppWhitelist []models.AppWhitelistEntry `json:"app_whitelist,omitempty"`
}

// AddAppRequest 添加应用到白名单请求
type AddAppRequest struct {
	PackageName       string `json:"package_name"`
	AppName          string `json:"app_name"`
	AutoLaunch       bool   `json:"auto_launch"`
	AllowNotifications bool   `json:"allow_notifications"`
	DefaultShortcut   bool   `json:"default_shortcut"`
}

// AssignPolicyRequest 分配策略请求
type AssignPolicyRequest struct {
	DeviceIDs []string `json:"device_ids"`
	GroupIDs  []string `json:"group_ids"`
}

// CreatePolicy 创建安全策略
// @Summary 创建安全策略
// @Description 为租户创建新的安全策略
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param request body CreatePolicyRequest true "创建策略请求"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies [post]
func (h *PolicyHandler) CreatePolicy(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant_id is required")
	}

	var req CreatePolicyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	svcReq := &services.CreatePolicyRequest{
		Name:        req.Name,
		Description: req.Description,
		Settings:    req.Settings,
		AppWhitelist: req.AppWhitelist,
	}

	policy, err := h.policySvc.CreatePolicy(c.Request().Context(), tenantID, svcReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, policy)
}

// GetPolicy 获取策略
// @Summary 获取策略详情
// @Description 根据策略ID获取策略详细信息
// @Tags 策略管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Success 200 {object} models.SecurityPolicy
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId} [get]
func (h *PolicyHandler) GetPolicy(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}

	policy, err := h.policySvc.GetPolicy(c.Request().Context(), policyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "policy not found")
	}

	return c.JSON(http.StatusOK, policy)
}

// ListPolicies 列出租户的所有策略
// @Summary 列出租户策略
// @Description 获取指定租户下的所有策略列表
// @Tags 策略管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies [get]
func (h *PolicyHandler) ListPolicies(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant_id is required")
	}

	policies, err := h.policySvc.ListPolicies(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"policies": policies,
		"total":    len(policies),
	})
}

// UpdatePolicy 更新策略
// @Summary 更新策略
// @Description 更新指定策略的信息和设置
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Param request body UpdatePolicyRequest true "更新策略请求"
// @Success 200 {object} models.SecurityPolicy
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId} [put]
func (h *PolicyHandler) UpdatePolicy(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}

	var req UpdatePolicyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// 获取现有策略
	policy, err := h.policySvc.GetPolicy(c.Request().Context(), policyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "policy not found")
	}

	// 更新字段
	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Description != "" {
		policy.Description = req.Description
	}
	if req.Settings != nil {
		policy.Settings = *req.Settings
	}
	if req.AppWhitelist != nil {
		policy.AppWhitelist = req.AppWhitelist
	}

	if err := h.policySvc.UpdatePolicy(c.Request().Context(), policy); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, policy)
}

// DeletePolicy 删除策略
// @Summary 删除策略
// @Description 删除指定的策略
// @Tags 策略管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId} [delete]
func (h *PolicyHandler) DeletePolicy(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}

	if err := h.policySvc.DeletePolicy(c.Request().Context(), policyID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "policy deleted successfully",
	})
}

// AssignPolicy 分配策略给设备
// @Summary 分配策略
// @Description 将策略分配给指定的设备
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Param request body AssignPolicyRequest true "分配策略请求"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId}/assign [post]
func (h *PolicyHandler) AssignPolicy(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}

	var req AssignPolicyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// TODO: 实现批量分配逻辑
	// 目前只支持单个设备分配
	if len(req.DeviceIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "device_ids is required")
	}

	deviceID := req.DeviceIDs[0]
	if err := h.policySvc.AssignPolicy(c.Request().Context(), policyID, deviceID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":   "policy assigned successfully",
		"policyId": policyID,
		"deviceId": deviceID,
	})
}

// AddAppToWhitelist 添加应用到白名单
// @Summary 添加应用到白名单
// @Description 将指定应用添加到策略的白名单中
// @Tags 白名单管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Param request body AddAppRequest true "添加应用请求"
// @Success 201 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId}/whitelist [post]
func (h *PolicyHandler) AddAppToWhitelist(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}

	var req AddAppRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.PackageName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "package_name is required")
	}

	entry := models.AppWhitelistEntry{
		PackageName:        req.PackageName,
		AppName:           req.AppName,
		AutoLaunch:        req.AutoLaunch,
		AllowNotifications: req.AllowNotifications,
		DefaultShortcut:   req.DefaultShortcut,
	}

	if err := h.policySvc.AddAppToWhitelist(c.Request().Context(), policyID, entry); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message":      "app added to whitelist",
		"policyId":    policyID,
		"packageName": req.PackageName,
	})
}

// RemoveAppFromWhitelist 从白名单移除应用
// @Summary 从白名单移除应用
// @Description 将指定应用从策略的白名单中移除
// @Tags 白名单管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param policyId path string true "策略ID"
// @Param packageName path string true "应用包名"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/policies/{policyId}/whitelist/{packageName} [delete]
func (h *PolicyHandler) RemoveAppFromWhitelist(c echo.Context) error {
	policyID := c.Param("policyId")
	packageName := c.Param("packageName")

	if policyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "policy_id is required")
	}
	if packageName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "package_name is required")
	}

	if err := h.policySvc.RemoveAppFromWhitelist(c.Request().Context(), policyID, packageName); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":      "app removed from whitelist",
		"policyId":    policyID,
		"packageName": packageName,
	})
}

// GetDeviceWhitelist 获取设备的白名单
// @Summary 获取设备白名单
// @Description 获取指定设备的应用白名单
// @Tags 白名单管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param deviceId path string true "设备ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/{deviceId}/whitelist [get]
func (h *PolicyHandler) GetDeviceWhitelist(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "device_id is required")
	}

	whitelist, err := h.policySvc.GetDeviceWhitelist(c.Request().Context(), deviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deviceId":  deviceID,
		"whitelist": whitelist,
		"total":    len(whitelist),
	})
}

// GetDevicePolicy 获取设备的策略
// @Summary 获取设备策略
// @Description 获取指定设备当前应用的策略
// @Tags 策略管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param deviceId path string true "设备ID"
// @Success 200 {object} models.SecurityPolicy
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/{deviceId}/policy [get]
func (h *PolicyHandler) GetDevicePolicy(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "device_id is required")
	}

	policy, err := h.policySvc.GetDevicePolicy(c.Request().Context(), deviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, policy)
}
