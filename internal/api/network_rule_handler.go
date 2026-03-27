// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// NetworkRuleHandler 网络规则API处理器
type NetworkRuleHandler struct {
	svc services.NetworkRuleService
}

// NewNetworkRuleHandler 创建网络规则处理器
func NewNetworkRuleHandler(svc services.NetworkRuleService) *NetworkRuleHandler {
	return &NetworkRuleHandler{svc: svc}
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type       string `json:"type"`         // domain/ip/url
	Pattern     string `json:"pattern"`       // 匹配模式
	Action     string `json:"action"`       // block/allow/log
	Priority   int    `json:"priority"`     // 优先级
	Enabled    bool   `json:"enabled"`      // 是否启用
	DeviceID   string `json:"device_id"`   // 设备ID，为空则应用于所有设备
}

// CreateRule 创建网络规则
func (h *NetworkRuleHandler) CreateRule(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.Pattern == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "pattern is required"})
	}

	ruleType := models.NetworkRuleType(req.Type)
	if ruleType == "" {
		ruleType = models.NetworkRuleTypeDomain
	}

	action := models.NetworkRuleAction(req.Action)
	if action == "" {
		action = models.NetworkRuleActionBlock
	}

	rule, err := h.svc.CreateRule(tenantID, req.Name, req.Description, ruleType, req.Pattern, action, req.Priority, req.Enabled, req.DeviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, rule)
}

// GetRule 获取规则
func (h *NetworkRuleHandler) GetRule(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "rule id is required"})
	}

	rule, err := h.svc.GetRule(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if rule == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
	}

	return c.JSON(http.StatusOK, rule)
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type       string `json:"type"`
	Pattern     string `json:"pattern"`
	Action     string `json:"action"`
	Priority   int    `json:"priority"`
	Enabled    bool   `json:"enabled"`
	DeviceID   string `json:"device_id"`
}

// UpdateRule 更新规则
func (h *NetworkRuleHandler) UpdateRule(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "rule id is required"})
	}

	rule, err := h.svc.GetRule(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if rule == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
	}

	var req UpdateRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Type != "" {
		rule.Type = models.NetworkRuleType(req.Type)
	}
	if req.Pattern != "" {
		rule.Pattern = req.Pattern
	}
	if req.Action != "" {
		rule.Action = models.NetworkRuleAction(req.Action)
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	rule.Enabled = req.Enabled
	if req.DeviceID != "" {
		rule.DeviceID = req.DeviceID
	}

	if err := h.svc.UpdateRule(rule); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rule)
}

// DeleteRule 删除规则
func (h *NetworkRuleHandler) DeleteRule(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "rule id is required"})
	}

	if err := h.svc.DeleteRule(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "rule deleted"})
}

// ListRules 获取规则列表
func (h *NetworkRuleHandler) ListRules(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	rules, total, err := h.svc.ListRules(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"rules":  rules,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// MatchRuleRequest 规则匹配请求
type MatchRuleRequest struct {
	DeviceID string `json:"device_id"`
	Domain  string `json:"domain"`
	IP      string `json:"ip"`
	URL     string `json:"url"`
}

// MatchRule 检查规则匹配
func (h *NetworkRuleHandler) MatchRule(c echo.Context) error {
	var req MatchRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	rule, err := h.svc.MatchRule(req.DeviceID, req.Domain, req.IP, req.URL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if rule == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"matched": false,
			"rule":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"matched": true,
		"rule":    rule,
	})
}

// ListTrafficLogs 获取流量日志
func (h *NetworkRuleHandler) ListTrafficLogs(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	deviceID := c.QueryParam("device_id")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	logs, total, err := h.svc.ListTrafficLogs(tenantID, deviceID, startDate, endDate, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetTrafficStats 获取流量统计
func (h *NetworkRuleHandler) GetTrafficStats(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	deviceID := c.Param("deviceId")
	date := c.QueryParam("date")
	if date == "" {
		date = "2006-01-02"
	}

	stats, err := h.svc.GetTrafficStats(tenantID, deviceID, date)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, stats)
}

// GetTopDomains 获取流量最大的域名
func (h *NetworkRuleHandler) GetTopDomains(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	deviceID := c.Param("deviceId")
	date := c.QueryParam("date")
	if date == "" {
		date = "2006-01-02"
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 10
	}

	stats, err := h.svc.GetTopDomains(tenantID, deviceID, date, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, stats)
}

// AddToWhitelistRequest 添加白名单请求
type AddToWhitelistRequest struct {
	Name        string `json:"name"`
	Type       string `json:"type"`   // domain/ip/url
	Pattern     string `json:"pattern"` // 匹配模式
	Description string `json:"description"`
	DeviceID   string `json:"device_id"`
}

// AddToWhitelist 添加到白名单
func (h *NetworkRuleHandler) AddToWhitelist(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req AddToWhitelistRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.Pattern == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "pattern is required"})
	}

	entryType := req.Type
	if entryType == "" {
		entryType = "domain"
	}

	err := h.svc.AddToWhitelist(tenantID, req.Name, entryType, req.Pattern, req.Description, req.DeviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "added to whitelist"})
}

// RemoveFromWhitelist 从白名单移除
func (h *NetworkRuleHandler) RemoveFromWhitelist(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if err := h.svc.RemoveFromWhitelist(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "removed from whitelist"})
}

// ListWhitelist 获取白名单
func (h *NetworkRuleHandler) ListWhitelist(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	entries, err := h.svc.ListWhitelist(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, entries)
}
