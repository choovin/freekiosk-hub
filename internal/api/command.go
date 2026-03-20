package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// CommandHandler 命令 API 处理器
type CommandHandler struct {
	cmdSvc   services.CommandService
	statusSvc services.DeviceStatusService
}

// NewCommandHandler 创建命令处理器
func NewCommandHandler(cmdSvc services.CommandService, statusSvc services.DeviceStatusService) *CommandHandler {
	return &CommandHandler{
		cmdSvc:   cmdSvc,
		statusSvc: statusSvc,
	}
}

// HandleSendCommand 发送命令到设备
// POST /api/v2/tenants/:tenantId/devices/:deviceId/commands
func (h *CommandHandler) HandleSendCommand(c echo.Context) error {
	tenantID := c.Param("tenantId")
	deviceID := c.Param("deviceId")

	if tenantID == "" || deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id and device_id are required",
		})
	}

	var req struct {
		Type     string                 `json:"type"`
		Params   map[string]interface{} `json:"params,omitempty"`
		Timeout  int                    `json:"timeout,omitempty"`
		Priority int                    `json:"priority,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "type is required",
		})
	}

	// 创建命令
	cmd := &models.Command{
		ID:       uuid.New().String(),
		Type:     models.CommandType(req.Type),
		Params:   req.Params,
		Timeout:  req.Timeout,
		Priority: req.Priority,
	}

	// 发送命令
	result, err := h.cmdSvc.SendCommand(c.Request().Context(), tenantID, deviceID, cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"command_id": cmd.ID,
		"success":    result.Success,
		"result":     result.Result,
		"error":      result.Error,
		"duration":   result.Duration,
	})
}

// HandleSendBatchCommand 发送批量命令
// POST /api/v2/tenants/:tenantId/commands/batch
func (h *CommandHandler) HandleSendBatchCommand(c echo.Context) error {
	tenantID := c.Param("tenantId")

	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	var req struct {
		DeviceIDs []string               `json:"device_ids,omitempty"`
		GroupIDs  []string               `json:"group_ids,omitempty"`
		All       bool                   `json:"all,omitempty"`
		Type      string                 `json:"type"`
		Params    map[string]interface{} `json:"params,omitempty"`
		Timeout   int                    `json:"timeout,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "type is required",
		})
	}

	// 创建命令
	cmd := &models.Command{
		ID:      uuid.New().String(),
		Type:    models.CommandType(req.Type),
		Params:  req.Params,
		Timeout: req.Timeout,
	}

	// 创建目标
	target := &models.CommandTarget{
		DeviceIDs: req.DeviceIDs,
		GroupIDs:  req.GroupIDs,
		All:       req.All,
	}

	// 发送批量命令
	result, err := h.cmdSvc.SendBatchCommand(c.Request().Context(), tenantID, target, cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusAccepted, result)
}

// HandleGetCommandHistory 获取命令历史
// GET /api/v2/tenants/:tenantId/devices/:deviceId/commands/history
func (h *CommandHandler) HandleGetCommandHistory(c echo.Context) error {
	tenantID := c.Param("tenantId")
	deviceID := c.Param("deviceId")

	if tenantID == "" || deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id and device_id are required",
		})
	}

	limit := 50
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil {
			limit = parsed
		}
	}

	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil {
			offset = parsed
		}
	}

	records, total, err := h.cmdSvc.GetCommandHistory(c.Request().Context(), tenantID, deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"commands": records,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// HandleGetCommandByID 获取命令详情
// GET /api/v2/tenants/:tenantId/commands/:commandId
func (h *CommandHandler) HandleGetCommandByID(c echo.Context) error {
	commandID := c.Param("commandId")

	if commandID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "command_id is required",
		})
	}

	record, err := h.cmdSvc.GetCommandByID(c.Request().Context(), commandID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Command not found",
		})
	}

	return c.JSON(http.StatusOK, record)
}

// HandleCancelCommand 取消命令
// DELETE /api/v2/tenants/:tenantId/commands/:commandId
func (h *CommandHandler) HandleCancelCommand(c echo.Context) error {
	commandID := c.Param("commandId")

	if commandID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "command_id is required",
		})
	}

	if err := h.cmdSvc.CancelCommand(c.Request().Context(), commandID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Command canceled",
	})
}

// StatusHandler 设备状态 API 处理器
type StatusHandler struct {
	statusSvc services.DeviceStatusService
}

// NewStatusHandler 创建状态处理器
func NewStatusHandler(statusSvc services.DeviceStatusService) *StatusHandler {
	return &StatusHandler{
		statusSvc: statusSvc,
	}
}

// HandleGetDeviceStatus 获取设备状态
// GET /api/v2/tenants/:tenantId/devices/:deviceId/status
func (h *StatusHandler) HandleGetDeviceStatus(c echo.Context) error {
	deviceID := c.Param("deviceId")

	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id is required",
		})
	}

	status, err := h.statusSvc.GetDeviceStatus(c.Request().Context(), deviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Device not found",
		})
	}

	return c.JSON(http.StatusOK, status)
}

// HandleGetAllDeviceStatuses 获取所有设备状态
// GET /api/v2/tenants/:tenantId/devices/status
func (h *StatusHandler) HandleGetAllDeviceStatuses(c echo.Context) error {
	tenantID := c.Param("tenantId")

	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	statuses, err := h.statusSvc.GetAllDeviceStatuses(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// 计算在线/离线数量
	onlineCount := 0
	offlineCount := 0
	for _, s := range statuses {
		if s.Online {
			onlineCount++
		} else {
			offlineCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices":      statuses,
		"total":        len(statuses),
		"online_count": onlineCount,
		"offline_count": offlineCount,
		"timestamp":    time.Now(),
	})
}

func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
