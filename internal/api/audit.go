// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package api

import (
	"net/http"
	"strconv"

	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/labstack/echo/v4"
)

// AuditHandler 审计日志 API 处理器
type AuditHandler struct {
	auditSvc *services.AuditService
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler(auditSvc *services.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// HandleQueryAuditLogs 查询审计日志
// @Summary 查询审计日志
// @Description 查询指定租户的审计日志记录
// @Tags 审计日志
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param limit query int false "每页数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Param actor_id query string false "操作者ID"
// @Param action query string false "操作类型"
// @Param resource_type query string false "资源类型"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/audit-logs [get]
func (h *AuditHandler) HandleQueryAuditLogs(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "tenantId is required")
	}

	// 解析查询参数
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	actorID := c.QueryParam("actor_id")
	action := c.QueryParam("action")
	resourceType := c.QueryParam("resource_type")

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	filter := &services.AuditLogFilter{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		Limit:        limit,
		Offset:       offset,
	}

	logs, total, err := h.auditSvc.Query(c.Request().Context(), filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}