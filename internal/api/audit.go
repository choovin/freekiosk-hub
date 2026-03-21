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