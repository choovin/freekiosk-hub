package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// RemoteControlHandler 远程控制处理器
type RemoteControlHandler struct {
	svc services.RemoteControlService
}

// NewRemoteControlHandler 创建远程控制处理器
func NewRemoteControlHandler(svc services.RemoteControlService) *RemoteControlHandler {
	return &RemoteControlHandler{svc: svc}
}

// RegisterRoutes 注册路由
func (h *RemoteControlHandler) RegisterRoutes(g *echo.Group) {
	// 会话管理
	g.POST("/sessions", h.CreateSession)
	g.GET("/sessions/:id", h.GetSession)
	g.PUT("/sessions/:id", h.UpdateSession)
	g.DELETE("/sessions/:id", h.DeleteSession)
	g.GET("/devices/:deviceId/sessions", h.ListDeviceSessions)
	g.GET("/devices/:deviceId/sessions/active", h.GetActiveSession)

	// 事件记录
	g.POST("/sessions/:sessionId/events", h.RecordEvent)
	g.GET("/sessions/:sessionId/events", h.GetSessionEvents)

	// 屏幕截图
	g.POST("/sessions/:sessionId/screenshots", h.SaveScreenCapture)
	g.GET("/sessions/:sessionId/screenshots", h.GetSessionScreenCaptures)

	// 命令管理
	g.POST("/sessions/:sessionId/commands", h.SendCommand)
	g.PUT("/commands/:id/status", h.UpdateCommandStatus)
	g.GET("/sessions/:sessionId/commands", h.GetSessionCommands)
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	DeviceID    string                   `json:"device_id"`
	TenantID    string                   `json:"tenant_id"`
	InitiatorID string                   `json:"initiator_id"`
	SessionType string                   `json:"session_type"` // view, control
	ICEServers  []map[string]interface{} `json:"ice_servers"`
}

// CreateSession 创建远程会话
func (h *RemoteControlHandler) CreateSession(c echo.Context) error {
	var req CreateSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if req.TenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}
	if req.SessionType == "" {
		req.SessionType = "view"
	}

	session, err := h.svc.CreateSession(req.DeviceID, req.TenantID, req.InitiatorID, req.SessionType, req.ICEServers)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, session)
}

// GetSession 获取会话
func (h *RemoteControlHandler) GetSession(c echo.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session id is required"})
	}

	session, err := h.svc.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if session == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	return c.JSON(http.StatusOK, session)
}

// UpdateSessionRequest 更新会话请求
type UpdateSessionRequest struct {
	Status string `json:"status"`
}

// UpdateSession 更新会话
func (h *RemoteControlHandler) UpdateSession(c echo.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session id is required"})
	}

	var req UpdateSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status is required"})
	}

	err := h.svc.UpdateSession(sessionID, req.Status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "session updated"})
}

// DeleteSession 删除会话
func (h *RemoteControlHandler) DeleteSession(c echo.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session id is required"})
	}

	err := h.svc.DeleteSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "session deleted"})
}

// ListDeviceSessions 获取设备的会话列表
func (h *RemoteControlHandler) ListDeviceSessions(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	sessions, total, err := h.svc.ListDeviceSessions(deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetActiveSession 获取设备的活跃会话
func (h *RemoteControlHandler) GetActiveSession(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	session, err := h.svc.GetActiveSession(deviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if session == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no active session"})
	}

	return c.JSON(http.StatusOK, session)
}

// RecordEventRequest 记录事件请求
type RecordEventRequest struct {
	DeviceID  string `json:"device_id"`
	TenantID  string `json:"tenant_id"`
	EventType string `json:"event_type"`
	Message   string `json:"message"`
}

// RecordEvent 记录会话事件
func (h *RemoteControlHandler) RecordEvent(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	var req RecordEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.EventType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_type is required"})
	}

	err := h.svc.RecordEvent(sessionID, req.DeviceID, req.TenantID, req.EventType, req.Message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "event recorded"})
}

// GetSessionEvents 获取会话的所有事件
func (h *RemoteControlHandler) GetSessionEvents(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	events, err := h.svc.GetSessionEvents(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, events)
}

// SaveScreenCaptureRequest 保存屏幕截图请求
type SaveScreenCaptureRequest struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	FilePath string `json:"file_path"`
	FileSize int64  `json:"file_size"`
}

// SaveScreenCapture 保存屏幕截图
func (h *RemoteControlHandler) SaveScreenCapture(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	var req SaveScreenCaptureRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.FilePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file_path is required"})
	}

	capture, err := h.svc.SaveScreenCapture(sessionID, req.DeviceID, req.TenantID, req.FilePath, req.FileSize)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, capture)
}

// GetSessionScreenCaptures 获取会话的屏幕截图
func (h *RemoteControlHandler) GetSessionScreenCaptures(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	captures, err := h.svc.GetSessionScreenCaptures(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, captures)
}

// SendCommandRequest 发送命令请求
type SendCommandRequest struct {
	DeviceID    string                 `json:"device_id"`
	TenantID    string                 `json:"tenant_id"`
	CommandType string                 `json:"command_type"`
	Params      map[string]interface{} `json:"params"`
}

// SendCommand 发送远程命令
func (h *RemoteControlHandler) SendCommand(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	var req SendCommandRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.CommandType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "command_type is required"})
	}

	cmd, err := h.svc.SendCommand(sessionID, req.DeviceID, req.TenantID, req.CommandType, req.Params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, cmd)
}

// UpdateCommandStatusRequest 更新命令状态请求
type UpdateCommandStatusRequest struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}

// UpdateCommandStatus 更新命令状态
func (h *RemoteControlHandler) UpdateCommandStatus(c echo.Context) error {
	cmdID := c.Param("id")
	if cmdID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "command id is required"})
	}

	var req UpdateCommandStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status is required"})
	}

	err := h.svc.UpdateCommandStatus(cmdID, req.Status, req.Response)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "command status updated"})
}

// GetSessionCommands 获取会话的所有命令
func (h *RemoteControlHandler) GetSessionCommands(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	commands, err := h.svc.GetSessionCommands(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, commands)
}

// ParseICEServers 解析ICE服务器配置
func ParseICEServers(iceServersJSON string) []map[string]interface{} {
	if iceServersJSON == "" {
		return nil
	}
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(iceServersJSON), &result); err != nil {
		return nil
	}
	return result
}

// Ensure RemoteSession implements response properly
var _ interface{} = models.RemoteSession{}
