package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// ConfigurationHandler 配置档案HTTP处理器
type ConfigurationHandler struct {
	svc services.ConfigurationService
}

// NewConfigurationHandler 创建配置档案处理器
func NewConfigurationHandler(svc services.ConfigurationService) *ConfigurationHandler {
	return &ConfigurationHandler{svc: svc}
}

// HandleCreate 创建配置档案
func (h *ConfigurationHandler) HandleCreate(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req struct {
		Name                    string   `json:"name"`
		Description             string   `json:"description"`
		PasswordMinLength       int      `json:"password_min_length"`
		PasswordRequireNumber   bool     `json:"password_require_number"`
		PasswordRequireSpecial  bool     `json:"password_require_special"`
		PasswordExpireDays      int      `json:"password_expire_days"`
		AppWhitelist            []string `json:"app_whitelist"`
		AppBlacklist           []string `json:"app_blacklist"`
		AllowInstallUnknownApps bool     `json:"allow_install_unknown_apps"`
		AllowedHoursStart       string   `json:"allowed_hours_start"`
		AllowedHoursEnd         string   `json:"allowed_hours_end"`
		AllowedDays             []int    `json:"allowed_days"`
		DeviceTimeout           int      `json:"device_timeout"`
		EnableGPS               bool     `json:"enable_gps"`
		EnableCamera            bool     `json:"enable_camera"`
		EnableUSB               bool     `json:"enable_usb"`
		SettingsJSON            string   `json:"settings_json"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	cfg := &models.ConfigurationProfile{
		Name:                    req.Name,
		Description:             req.Description,
		TenantID:                tenantID,
		PasswordMinLength:       req.PasswordMinLength,
		PasswordRequireNumber:   req.PasswordRequireNumber,
		PasswordRequireSpecial:  req.PasswordRequireSpecial,
		PasswordExpireDays:      req.PasswordExpireDays,
		AppWhitelist:            req.AppWhitelist,
		AppBlacklist:           req.AppBlacklist,
		AllowInstallUnknownApps: req.AllowInstallUnknownApps,
		AllowedHoursStart:       req.AllowedHoursStart,
		AllowedHoursEnd:         req.AllowedHoursEnd,
		AllowedDays:             req.AllowedDays,
		DeviceTimeout:           req.DeviceTimeout,
		EnableGPS:               req.EnableGPS,
		EnableCamera:            req.EnableCamera,
		EnableUSB:               req.EnableUSB,
		SettingsJSON:            req.SettingsJSON,
	}

	if err := h.svc.Create(cfg); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, cfg)
}

// HandleGet 获取单个配置档案
func (h *ConfigurationHandler) HandleGet(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "configuration id is required"})
	}

	cfg, err := h.svc.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "configuration not found"})
	}

	return c.JSON(http.StatusOK, cfg)
}

// HandleUpdate 更新配置档案
func (h *ConfigurationHandler) HandleUpdate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "configuration id is required"})
	}

	var req struct {
		Name                    string   `json:"name"`
		Description             string   `json:"description"`
		PasswordMinLength       int      `json:"password_min_length"`
		PasswordRequireNumber   bool     `json:"password_require_number"`
		PasswordRequireSpecial  bool     `json:"password_require_special"`
		PasswordExpireDays      int      `json:"password_expire_days"`
		AppWhitelist            []string `json:"app_whitelist"`
		AppBlacklist           []string `json:"app_blacklist"`
		AllowInstallUnknownApps bool     `json:"allow_install_unknown_apps"`
		AllowedHoursStart       string   `json:"allowed_hours_start"`
		AllowedHoursEnd         string   `json:"allowed_hours_end"`
		AllowedDays             []int    `json:"allowed_days"`
		DeviceTimeout           int      `json:"device_timeout"`
		EnableGPS               bool     `json:"enable_gps"`
		EnableCamera            bool     `json:"enable_camera"`
		EnableUSB               bool     `json:"enable_usb"`
		SettingsJSON            string   `json:"settings_json"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	cfg, err := h.svc.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "configuration not found"})
	}

	cfg.Name = req.Name
	cfg.Description = req.Description
	cfg.PasswordMinLength = req.PasswordMinLength
	cfg.PasswordRequireNumber = req.PasswordRequireNumber
	cfg.PasswordRequireSpecial = req.PasswordRequireSpecial
	cfg.PasswordExpireDays = req.PasswordExpireDays
	cfg.AppWhitelist = req.AppWhitelist
	cfg.AppBlacklist = req.AppBlacklist
	cfg.AllowInstallUnknownApps = req.AllowInstallUnknownApps
	cfg.AllowedHoursStart = req.AllowedHoursStart
	cfg.AllowedHoursEnd = req.AllowedHoursEnd
	cfg.AllowedDays = req.AllowedDays
	cfg.DeviceTimeout = req.DeviceTimeout
	cfg.EnableGPS = req.EnableGPS
	cfg.EnableCamera = req.EnableCamera
	cfg.EnableUSB = req.EnableUSB
	cfg.SettingsJSON = req.SettingsJSON

	if err := h.svc.Update(cfg); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, cfg)
}

// HandleDelete 删除配置档案
func (h *ConfigurationHandler) HandleDelete(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "configuration id is required"})
	}

	if err := h.svc.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "configuration deleted"})
}

// HandleList 获取配置档案列表
func (h *ConfigurationHandler) HandleList(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	configs, total, err := h.svc.List(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"configurations": configs,
		"total":          total,
		"limit":          limit,
		"offset":         offset,
	})
}

// HandleAssignToDevice 分配配置到设备
func (h *ConfigurationHandler) HandleAssignToDevice(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req struct {
		DeviceID   string `json:"device_id"`
		ConfigID   string `json:"config_id"`
		AssignedBy string `json:"assigned_by"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.DeviceID == "" || req.ConfigID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id and config_id are required"})
	}

	if err := h.svc.AssignToDevice(req.DeviceID, req.ConfigID, tenantID, req.AssignedBy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "configuration assigned to device"})
}

// HandleUnassignFromDevice 取消分配
func (h *ConfigurationHandler) HandleUnassignFromDevice(c echo.Context) error {
	var req struct {
		DeviceID string `json:"device_id"`
		ConfigID string `json:"config_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.DeviceID == "" || req.ConfigID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id and config_id are required"})
	}

	if err := h.svc.UnassignFromDevice(req.DeviceID, req.ConfigID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "configuration unassigned from device"})
}

// HandleGetDeviceConfiguration 获取设备当前配置
func (h *ConfigurationHandler) HandleGetDeviceConfiguration(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	cfg, err := h.svc.GetDeviceConfiguration(deviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if cfg == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"configuration": nil})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"configuration": cfg})
}

// HandleGetDeviceConfigurations 获取设备所有配置
func (h *ConfigurationHandler) HandleGetDeviceConfigurations(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}

	configs, err := h.svc.GetDeviceConfigurations(deviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"configurations": configs})
}
