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
