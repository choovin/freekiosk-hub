package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// GeofenceHandler 地理围栏HTTP处理器
type GeofenceHandler struct {
	svc services.GeofenceService
}

// NewGeofenceHandler 创建地理围栏处理器
func NewGeofenceHandler(svc services.GeofenceService) *GeofenceHandler {
	return &GeofenceHandler{svc: svc}
}

// HandleCreate 创建地理围栏
func (h *GeofenceHandler) HandleCreate(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		FenceType     string  `json:"fence_type"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		Radius         float64 `json:"radius"`
		Coordinates    string  `json:"coordinates"`
		IsActive       bool    `json:"is_active"`
		AlertOnEnter   bool    `json:"alert_on_enter"`
		AlertOnExit    bool    `json:"alert_on_exit"`
		TimeRestriction string  `json:"time_restriction"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	gf := &models.Geofence{
		Name:           req.Name,
		Description:    req.Description,
		TenantID:       tenantID,
		FenceType:      req.FenceType,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Radius:         req.Radius,
		Coordinates:    req.Coordinates,
		IsActive:       req.IsActive,
		AlertOnEnter:   req.AlertOnEnter,
		AlertOnExit:    req.AlertOnExit,
		TimeRestriction: req.TimeRestriction,
	}

	if err := h.svc.Create(gf); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, gf)
}

// HandleGet 获取单个地理围栏
func (h *GeofenceHandler) HandleGet(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence id is required"})
	}

	gf, err := h.svc.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if gf == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "geofence not found"})
	}

	return c.JSON(http.StatusOK, gf)
}

// HandleUpdate 更新地理围栏
func (h *GeofenceHandler) HandleUpdate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence id is required"})
	}

	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		FenceType     string  `json:"fence_type"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		Radius         float64 `json:"radius"`
		Coordinates    string  `json:"coordinates"`
		IsActive       bool    `json:"is_active"`
		AlertOnEnter   bool    `json:"alert_on_enter"`
		AlertOnExit    bool    `json:"alert_on_exit"`
		TimeRestriction string  `json:"time_restriction"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	gf, err := h.svc.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if gf == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "geofence not found"})
	}

	gf.Name = req.Name
	gf.Description = req.Description
	gf.FenceType = req.FenceType
	gf.Latitude = req.Latitude
	gf.Longitude = req.Longitude
	gf.Radius = req.Radius
	gf.Coordinates = req.Coordinates
	gf.IsActive = req.IsActive
	gf.AlertOnEnter = req.AlertOnEnter
	gf.AlertOnExit = req.AlertOnExit
	gf.TimeRestriction = req.TimeRestriction

	if err := h.svc.Update(gf); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, gf)
}

// HandleDelete 删除地理围栏
func (h *GeofenceHandler) HandleDelete(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence id is required"})
	}

	if err := h.svc.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "geofence deleted"})
}

// HandleList 获取地理围栏列表
func (h *GeofenceHandler) HandleList(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	geofences, total, err := h.svc.List(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"geofences": geofences,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// HandleListActive 获取激活的地理围栏列表
func (h *GeofenceHandler) HandleListActive(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	geofences, err := h.svc.ListActive(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"geofences": geofences,
	})
}

// HandleAssignToDevice 分配围栏到设备
func (h *GeofenceHandler) HandleAssignToDevice(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req struct {
		GeofenceID string `json:"geofence_id"`
		DeviceID   string `json:"device_id"`
		AssignedBy string `json:"assigned_by"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.GeofenceID == "" || req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence_id and device_id are required"})
	}

	if err := h.svc.AssignToDevice(req.GeofenceID, req.DeviceID, tenantID, req.AssignedBy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "geofence assigned to device"})
}

// HandleUnassignFromDevice 取消分配
func (h *GeofenceHandler) HandleUnassignFromDevice(c echo.Context) error {
	var req struct {
		GeofenceID string `json:"geofence_id"`
		DeviceID   string `json:"device_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.GeofenceID == "" || req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence_id and device_id are required"})
	}

	if err := h.svc.UnassignFromDevice(req.GeofenceID, req.DeviceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "geofence unassigned from device"})
}

// HandleGetDeviceGeofences 获取设备的围栏
func (h *GeofenceHandler) HandleGetDeviceGeofences(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	geofences, err := h.svc.GetDeviceGeofences(deviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"geofences": geofences,
	})
}

// HandleGetDeviceEvents 获取设备的围栏事件
func (h *GeofenceHandler) HandleGetDeviceEvents(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	events, total, err := h.svc.GetDeviceEvents(deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HandleGetGeofenceEvents 获取围栏的事件
func (h *GeofenceHandler) HandleGetGeofenceEvents(c echo.Context) error {
	geofenceID := c.Param("id")
	if geofenceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "geofence_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	events, total, err := h.svc.GetGeofenceEvents(geofenceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HandleCheckLocation 检查设备位置
func (h *GeofenceHandler) HandleCheckLocation(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	events, err := h.svc.CheckDeviceLocation(deviceID, req.Latitude, req.Longitude)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
	})
}
