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
// @Summary 发送设备命令
// @Description 向指定设备发送控制命令（如屏幕开关、导航、音频等）
// @Tags 命令管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param deviceId path string true "设备ID"
// @Param command body CommandRequest true "命令请求"
// @Success 200 {object} CommandResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/{deviceId}/commands [post]
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
// @Summary 发送批量命令
// @Description 向多个设备或分组发送批量命令
// @Tags 命令管理
// @Accept json
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param request body BatchCommandRequest true "批量命令请求"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/commands/batch [post]
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
// @Summary 获取命令历史
// @Description 获取指定设备的命令历史记录
// @Tags 命令管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param deviceId path string true "设备ID"
// @Param limit query int false "每页数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/{deviceId}/commands/history [get]
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
// @Summary 获取命令详情
// @Description 根据命令ID获取命令详细信息
// @Tags 命令管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param commandId path string true "命令ID"
// @Success 200 {object} models.Command
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/commands/{commandId} [get]
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
// @Summary 取消命令
// @Description 取消指定的待执行命令
// @Tags 命令管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param commandId path string true "命令ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/commands/{commandId} [delete]
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
// @Summary 获取设备状态
// @Description 获取指定设备的当前状态信息
// @Tags 状态管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Param deviceId path string true "设备ID"
// @Success 200 {object} models.DeviceStatus
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/{deviceId}/status [get]
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
// @Summary 获取所有设备状态
// @Description 获取指定租户下所有设备的状态概览
// @Tags 状态管理
// @Produce json
// @Param tenantId path string true "租户ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/tenants/{tenantId}/devices/status [get]
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

// CommandRequest 发送命令请求
type CommandRequest struct {
	Type     string                 `json:"type" example:"setScreen" validate:"required"`
	Params   map[string]interface{} `json:"params,omitempty"`
	Timeout  int                    `json:"timeout,omitempty" example:"30"`
	Priority int                    `json:"priority,omitempty" example:"0"`
}

// CommandResponse 发送命令响应
type CommandResponse struct {
	CommandID string                 `json:"command_id" example:"cmd-uuid-1234"`
	Success   bool                   `json:"success" example:"true"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty" example:""`
	Duration  int64                 `json:"duration" example:"150"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// BatchCommandRequest 批量命令请求
type BatchCommandRequest struct {
	DeviceIDs []string               `json:"device_ids,omitempty"`
	GroupIDs  []string               `json:"group_ids,omitempty"`
	All       bool                   `json:"all,omitempty"`
	Type      string                 `json:"type" example:"setScreen"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timeout   int                    `json:"timeout,omitempty" example:"30"`
}
