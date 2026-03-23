package api

import (
	"net/http"
	"strconv"

	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/labstack/echo/v4"
)

// TenantHandler 租户 API 处理器
type TenantHandler struct {
	tenantSvc services.TenantService
}

// NewTenantHandler 创建租户处理器
func NewTenantHandler(tenantSvc services.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: tenantSvc}
}

// HandleCreateTenant 创建租户
// @Summary 创建租户
// @Description 创建新的租户
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param request body services.CreateTenantRequest true "创建租户请求"
// @Success 201 {object} models.Tenant
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants [post]
func (h *TenantHandler) HandleCreateTenant(c echo.Context) error {
	var req services.CreateTenantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	tenant, err := h.tenantSvc.CreateTenant(c.Request().Context(), &req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, tenant)
}

// HandleGetTenant 获取租户
// @Summary 获取租户
// @Description 根据租户ID获取租户详情
// @Tags 租户管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Success 200 {object} models.Tenant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId} [get]
func (h *TenantHandler) HandleGetTenant(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	tenant, err := h.tenantSvc.GetTenant(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	return c.JSON(http.StatusOK, tenant)
}

// HandleUpdateTenant 更新租户
// @Summary 更新租户
// @Description 更新租户信息
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param request body services.UpdateTenantRequest true "更新租户请求"
// @Success 200 {object} models.Tenant
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId} [put]
func (h *TenantHandler) HandleUpdateTenant(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	var req services.UpdateTenantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	tenant, err := h.tenantSvc.UpdateTenant(c.Request().Context(), tenantID, &req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, tenant)
}

// HandleDeleteTenant 删除租户
// @Summary 删除租户
// @Description 删除指定的租户
// @Tags 租户管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId} [delete]
func (h *TenantHandler) HandleDeleteTenant(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	err := h.tenantSvc.DeleteTenant(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// HandleListTenants 列出租户
// @Summary 列出租户
// @Description 获取租户列表
// @Tags 租户管理
// @Produce json
// @Param limit query int false "每页数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants [get]
func (h *TenantHandler) HandleListTenants(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 20
	}

	tenants, total, err := h.tenantSvc.ListTenants(c.Request().Context(), limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tenants": tenants,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleGetQuota 获取配额
// @Summary 获取租户配额
// @Description 获取指定租户的配额信息
// @Tags 租户管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Success 200 {object} models.TenantQuota
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/quota [get]
func (h *TenantHandler) HandleGetQuota(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	quota, err := h.tenantSvc.GetQuota(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	return c.JSON(http.StatusOK, quota)
}

// HandleUpdateQuota 更新配额
// @Summary 更新租户配额
// @Description 更新指定租户的配额信息
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param request body models.TenantQuota true "配额信息"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/quota [put]
func (h *TenantHandler) HandleUpdateQuota(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	var quota models.TenantQuota
	if err := c.Bind(&quota); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	err := h.tenantSvc.UpdateQuota(c.Request().Context(), tenantID, &quota)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "quota updated"})
}
