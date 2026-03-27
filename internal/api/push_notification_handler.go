package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// PushNotificationHandler 推送通知处理器
type PushNotificationHandler struct {
	svc services.PushNotificationService
}

// NewPushNotificationHandler 创建推送通知处理器
func NewPushNotificationHandler(svc services.PushNotificationService) *PushNotificationHandler {
	return &PushNotificationHandler{svc: svc}
}

// RegisterRoutes 注册路由
func (h *PushNotificationHandler) RegisterRoutes(g *echo.Group) {
	// 通知管理
	g.POST("/notifications", h.CreateNotification)
	g.GET("/notifications", h.ListNotifications)
	g.GET("/notifications/:id", h.GetNotification)
	g.PUT("/notifications/:id", h.UpdateNotification)
	g.DELETE("/notifications/:id", h.DeleteNotification)

	// 发送
	g.POST("/notifications/send/device", h.SendToDevice)
	g.POST("/notifications/send/group", h.SendToGroup)
	g.POST("/notifications/send/all", h.SendToAll)
	g.POST("/notifications/schedule", h.ScheduleNotification)

	// 回执
	g.GET("/notifications/:id/receipts", h.GetReceipts)
	g.GET("/devices/:deviceId/notifications", h.ListDeviceNotifications)
	g.GET("/devices/:deviceId/receipts", h.GetDeviceReceipts)
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	Title    string                `json:"title"`
	Content  string                `json:"content"`
	Type     string                `json:"type"`
	Priority string                `json:"priority"`
	Actions  []models.PushAction  `json:"actions"`
}

// CreateNotification 创建通知
func (h *PushNotificationHandler) CreateNotification(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	notification, err := h.svc.CreateNotification(tenantID, req.Title, req.Content, req.Type, req.Priority, req.Actions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// GetNotification 获取通知
func (h *PushNotificationHandler) GetNotification(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notification id is required"})
	}

	notification, err := h.svc.GetNotification(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if notification == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
	}

	return c.JSON(http.StatusOK, notification)
}

// UpdateNotificationRequest 更新通知请求
type UpdateNotificationRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Priority string `json:"priority"`
	Type     string `json:"type"`
}

// UpdateNotification 更新通知
func (h *PushNotificationHandler) UpdateNotification(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notification id is required"})
	}

	notification, err := h.svc.GetNotification(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if notification == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
	}

	var req UpdateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Title != "" {
		notification.Title = req.Title
	}
	if req.Content != "" {
		notification.Content = req.Content
	}
	if req.Priority != "" {
		notification.Priority = req.Priority
	}
	if req.Type != "" {
		notification.Type = req.Type
	}

	if err := h.svc.UpdateNotification(notification); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notification)
}

// DeleteNotification 删除通知
func (h *PushNotificationHandler) DeleteNotification(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notification id is required"})
	}

	if err := h.svc.DeleteNotification(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "notification deleted"})
}

// ListNotifications 获取通知列表
func (h *PushNotificationHandler) ListNotifications(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	notifications, total, err := h.svc.ListNotifications(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// SendToDeviceRequest 发送到设备请求
type SendToDeviceRequest struct {
	DeviceID string `json:"device_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Type     string `json:"type"`
}

// SendToDevice 发送通知到设备
func (h *PushNotificationHandler) SendToDevice(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req SendToDeviceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	notification, err := h.svc.SendToDevice(tenantID, req.DeviceID, req.Title, req.Content, req.Type)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// SendToGroupRequest 发送到设备组请求
type SendToGroupRequest struct {
	GroupID string `json:"group_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// SendToGroup 发送通知到设备组
func (h *PushNotificationHandler) SendToGroup(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req SendToGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.GroupID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "group_id is required"})
	}
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	notification, err := h.svc.SendToGroup(tenantID, req.GroupID, req.Title, req.Content, req.Type)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// SendToAllRequest 广播通知请求
type SendToAllRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// SendToAll 广播通知到所有设备
func (h *PushNotificationHandler) SendToAll(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req SendToAllRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	notification, err := h.svc.SendToAll(tenantID, req.Title, req.Content, req.Type)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// ScheduleNotificationRequest 计划发送通知请求
type ScheduleNotificationRequest struct {
	Title        string `json:"title"`
	Content      string `json:"content"`
	Type         string `json:"type"`
	ScheduledAt  int64  `json:"scheduled_at"`
}

// ScheduleNotification 计划发送通知
func (h *PushNotificationHandler) ScheduleNotification(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req ScheduleNotificationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}
	if req.ScheduledAt == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "scheduled_at is required"})
	}

	notification, err := h.svc.ScheduleNotification(tenantID, req.Title, req.Content, req.Type, req.ScheduledAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// GetReceipts 获取通知的所有回执
func (h *PushNotificationHandler) GetReceipts(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notification id is required"})
	}

	receipts, err := h.svc.GetReceipts(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, receipts)
}

// ListDeviceNotifications 获取设备的通知列表
func (h *PushNotificationHandler) ListDeviceNotifications(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	notifications, total, err := h.svc.ListDeviceNotifications(deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetDeviceReceipts 获取设备的回执列表
func (h *PushNotificationHandler) GetDeviceReceipts(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	receipts, total, err := h.svc.GetDeviceReceipts(deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"receipts": receipts,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}
