package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/wared2003/freekiosk-hub/ui"

	"github.com/labstack/echo/v4"
)

// MDMTabletHandler MDM平板设备HTTP处理器
type MDMTabletHandler struct {
	mdmService services.MDMTabletService
}

// NewMDMTabletHandler 创建MDM平板设备处理器
func NewMDMTabletHandler(mdmService services.MDMTabletService) *MDMTabletHandler {
	return &MDMTabletHandler{mdmService: mdmService}
}

// HandleListDevices 获取设备列表
func (h *MDMTabletHandler) HandleListDevices(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	devices, total, err := h.mdmService.ListDevices(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices": devices,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleSearchDevices 搜索设备
func (h *MDMTabletHandler) HandleSearchDevices(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	filter := &models.DeviceSearchFilter{
		TenantID: tenantID,
		Status:   c.QueryParam("status"),
		Search:   c.QueryParam("search"),
		GroupID:  c.QueryParam("group_id"),
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	filter.Limit = limit
	filter.Offset = offset

	devices, total, err := h.mdmService.SearchDevices(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices": devices,
		"total":   total,
	})
}

// HandleGetDevice 获取单个设备
func (h *MDMTabletHandler) HandleGetDevice(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	device, err := h.mdmService.GetDevice(id)
	if err != nil {
		if err == repositories.ErrDeviceNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "device not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, device)
}

// HandleCreateDevice 创建设备
func (h *MDMTabletHandler) HandleCreateDevice(c echo.Context) error {
	var device models.MDMTablet
	if err := c.Bind(&device); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if device.TenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	if err := h.mdmService.CreateDevice(&device); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, device)
}

// HandleUpdateDevice 更新设备
func (h *MDMTabletHandler) HandleUpdateDevice(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	var device models.MDMTablet
	if err := c.Bind(&device); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	device.ID = id
	if err := h.mdmService.UpdateDevice(&device); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, device)
}

// HandleDeleteDevice 删除设备
func (h *MDMTabletHandler) HandleDeleteDevice(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	if err := h.mdmService.DeleteDevice(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "device deleted"})
}

// HandleUpdateDeviceStatus 更新设备状态
func (h *MDMTabletHandler) HandleUpdateDeviceStatus(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	status := models.MDMTabletStatus(c.FormValue("status"))
	if status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status is required"})
	}

	if err := h.mdmService.UpdateDeviceStatus(id, status); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "status updated"})
}

// HandleBulkUpdateStatus 批量更新设备状态
func (h *MDMTabletHandler) HandleBulkUpdateStatus(c echo.Context) error {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
		Status    string   `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if len(req.DeviceIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_ids is required"})
	}

	status := models.MDMTabletStatus(req.Status)
	if err := h.mdmService.BulkUpdateStatus(req.DeviceIDs, status); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":       "status updated",
		"affected_count": len(req.DeviceIDs),
	})
}

// HandleListGroups 获取设备分组列表
func (h *MDMTabletHandler) HandleListGroups(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	groups, err := h.mdmService.ListGroups(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, groups)
}

// HandleCreateGroup 创建设备分组
func (h *MDMTabletHandler) HandleCreateGroup(c echo.Context) error {
	var group models.MDMTabletGroup
	if err := c.Bind(&group); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if group.TenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	if err := h.mdmService.CreateGroup(&group); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, group)
}

// HandleUpdateGroup 更新设备分组
func (h *MDMTabletHandler) HandleUpdateGroup(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "group id is required"})
	}

	var group models.MDMTabletGroup
	if err := c.Bind(&group); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	group.ID = id
	if err := h.mdmService.UpdateGroup(&group); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, group)
}

// HandleDeleteGroup 删除设备分组
func (h *MDMTabletHandler) HandleDeleteGroup(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "group id is required"})
	}

	if err := h.mdmService.DeleteGroup(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "group deleted"})
}

// HandleAssignDeviceToGroup 分配设备到分组
func (h *MDMTabletHandler) HandleAssignDeviceToGroup(c echo.Context) error {
	deviceID := c.Param("device_id")
	groupID := c.Param("group_id")

	if deviceID == "" || groupID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id and group_id are required"})
	}

	if err := h.mdmService.AssignDeviceToGroup(deviceID, groupID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "device assigned to group"})
}

// HandleBulkAssignGroup 批量分配设备到分组
func (h *MDMTabletHandler) HandleBulkAssignGroup(c echo.Context) error {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
		GroupID   string   `json:"group_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if len(req.DeviceIDs) == 0 || req.GroupID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_ids and group_id are required"})
	}

	if err := h.mdmService.BulkAssignGroup(req.DeviceIDs, req.GroupID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":        "devices assigned to group",
		"affected_count": len(req.DeviceIDs),
	})
}

// HandleAddTag 添加设备标签
func (h *MDMTabletHandler) HandleAddTag(c echo.Context) error {
	var tag models.MDMTabletTag
	if err := c.Bind(&tag); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if tag.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	if err := h.mdmService.AddTag(&tag); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, tag)
}

// HandleRemoveTag 移除设备标签
func (h *MDMTabletHandler) HandleRemoveTag(c echo.Context) error {
	deviceID := c.Param("device_id")
	tagName := c.Param("tag_name")

	if deviceID == "" || tagName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id and tag_name are required"})
	}

	if err := h.mdmService.RemoveTag(deviceID, tagName); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "tag removed"})
}

// HandleGetDeviceTags 获取设备标签
func (h *MDMTabletHandler) HandleGetDeviceTags(c echo.Context) error {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	tags, err := h.mdmService.GetDeviceTags(deviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, tags)
}

// HandleUpdateLocation 更新设备位置
func (h *MDMTabletHandler) HandleUpdateLocation(c echo.Context) error {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	var req struct {
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		LocationTime int64  `json:"location_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	timestamp := req.LocationTime
	if timestamp == 0 {
		timestamp = int64(time.Now().Unix())
	}

	if err := h.mdmService.UpdateLocation(deviceID, req.Lat, req.Lng, timestamp); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "location updated"})
}

// HandleGetDeviceLocation 获取设备位置
func (h *MDMTabletHandler) HandleGetDeviceLocation(c echo.Context) error {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	location, err := h.mdmService.GetDeviceLocation(deviceID)
	if err != nil {
		if err == repositories.ErrDeviceNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "device not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, location)
}

// HandleRecordEvent 记录设备事件
func (h *MDMTabletHandler) HandleRecordEvent(c echo.Context) error {
	var event models.MDMTabletEvent
	if err := c.Bind(&event); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if event.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	if err := h.mdmService.RecordEvent(&event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, event)
}

// HandleGetDeviceEvents 获取设备事件
func (h *MDMTabletHandler) HandleGetDeviceEvents(c echo.Context) error {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}

	events, err := h.mdmService.GetDeviceEvents(deviceID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, events)
}

// HandleGetDeviceByNumber 根据编号获取设备
func (h *MDMTabletHandler) HandleGetDeviceByNumber(c echo.Context) error {
	number := c.Param("number")
	if number == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device number is required"})
	}

	device, err := h.mdmService.GetDeviceByNumber(number)
	if err != nil {
		if err == repositories.ErrDeviceNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "device not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, device)
}

// HandleUnassignDeviceFromGroup 从分组移除设备
func (h *MDMTabletHandler) HandleUnassignDeviceFromGroup(c echo.Context) error {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	if err := h.mdmService.UnassignDeviceFromGroup(deviceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "device unassigned from group"})
}

// HandleMDMTabletsDashboard MDM设备仪表板页面
func (h *MDMTabletHandler) HandleMDMTabletsDashboard(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	devices, total, err := h.mdmService.ListDevices(tenantID, 50, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	groups, err := h.mdmService.ListGroups(tenantID)
	if err != nil {
		groups = []*models.MDMTabletGroup{}
	}

	return c.Render(http.StatusOK, "mdm_dashboard.html", map[string]interface{}{
		"devices": devices,
		"groups":  groups,
		"total":   total,
		"tenant":   tenantID,
	})
}

// HandleMDMTabletDetails MDM设备详情页面
func (h *MDMTabletHandler) HandleMDMTabletDetails(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	device, err := h.mdmService.GetDevice(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "device not found"})
	}

	tags, _ := h.mdmService.GetDeviceTags(id)
	events, _ := h.mdmService.GetDeviceEvents(id, 20)
	location, _ := h.mdmService.GetDeviceLocation(id)

	return c.Render(http.StatusOK, "mdm_tablet_details.html", map[string]interface{}{
		"device":   device,
		"tags":     tags,
		"events":   events,
		"location": location,
	})
}

// HandleMDMTabletModal 设备模态框 (用于HTMX刷新)
func (h *MDMTabletHandler) HandleMDMTabletModal(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return ui.Toast("设备ID不能为空", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	device, err := h.mdmService.GetDevice(id)
	if err != nil {
		return ui.Toast("设备不存在", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	return c.Render(http.StatusOK, "mdm_tablet_row.html", device)
}

// HandleQRCode 生成设备二维码
func (h *MDMTabletHandler) HandleQRCode(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device id is required"})
	}

	device, err := h.mdmService.GetDevice(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "device not found"})
	}

	// 生成二维码内容 (设备绑定URL)
	qrContent := fmt.Sprintf("freekiosk://bind?device=%s&number=%s", device.ID, device.Number)

	return c.JSON(http.StatusOK, map[string]string{
		"device_id":   device.ID,
		"device_name": device.Name,
		"device_number": device.Number,
		"qr_content":  qrContent,
	})
}
