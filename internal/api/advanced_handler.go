// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// AdvancedHandler 高级功能API处理器
type AdvancedHandler struct {
	svc services.AdvancedService
}

// NewAdvancedHandler 创建高级功能处理器
func NewAdvancedHandler(svc services.AdvancedService) *AdvancedHandler {
	return &AdvancedHandler{svc: svc}
}

// ===== DevicePhoto Handlers =====

// CreateDevicePhotoRequest 创建设备拍照请求
type CreateDevicePhotoRequest struct {
	DeviceID    string `json:"device_id"`
	URL         string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	FileSize    int64  `json:"file_size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Description string `json:"description"`
}

// CreateDevicePhoto 创建设备拍照记录
func (h *AdvancedHandler) CreateDevicePhoto(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateDevicePhotoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}

	photo, err := h.svc.CreateDevicePhoto(tenantID, req.DeviceID, req.URL, req.ThumbnailURL, req.FileSize, req.Width, req.Height, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, photo)
}

// GetDevicePhoto 获取设备拍照记录
func (h *AdvancedHandler) GetDevicePhoto(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	photo, err := h.svc.GetDevicePhoto(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if photo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "photo not found"})
	}

	return c.JSON(http.StatusOK, photo)
}

// ListDevicePhotos 获取设备拍照列表
func (h *AdvancedHandler) ListDevicePhotos(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	deviceID := c.QueryParam("device_id")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	photos, total, err := h.svc.ListDevicePhotos(tenantID, deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"photos": photos,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteDevicePhoto 删除设备拍照记录
func (h *AdvancedHandler) DeleteDevicePhoto(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if err := h.svc.DeleteDevicePhoto(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "photo deleted"})
}

// ===== Contact Handlers =====

// CreateContactRequest 创建联系人请求
type CreateContactRequest struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	JobTitle string `json:"job_title"`
	Address  string `json:"address"`
	Note     string `json:"note"`
}

// CreateContact 创建联系人
func (h *AdvancedHandler) CreateContact(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateContactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	contact, err := h.svc.CreateContact(tenantID, req.DeviceID, req.Name, req.Phone, req.Email, req.Company, req.JobTitle, req.Address, req.Note)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, contact)
}

// GetContact 获取联系人
func (h *AdvancedHandler) GetContact(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	contact, err := h.svc.GetContact(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if contact == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "contact not found"})
	}

	return c.JSON(http.StatusOK, contact)
}

// UpdateContactRequest 更新联系人请求
type UpdateContactRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	JobTitle string `json:"job_title"`
	Address  string `json:"address"`
	Note     string `json:"note"`
	Starred  *bool  `json:"starred"`
}

// UpdateContact 更新联系人
func (h *AdvancedHandler) UpdateContact(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	contact, err := h.svc.GetContact(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if contact == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "contact not found"})
	}

	var req UpdateContactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		contact.Name = req.Name
	}
	if req.Phone != "" {
		contact.Phone = req.Phone
	}
	if req.Email != "" {
		contact.Email = req.Email
	}
	if req.Company != "" {
		contact.Company = req.Company
	}
	if req.JobTitle != "" {
		contact.JobTitle = req.JobTitle
	}
	if req.Address != "" {
		contact.Address = req.Address
	}
	if req.Note != "" {
		contact.Note = req.Note
	}
	if req.Starred != nil {
		contact.Starred = *req.Starred
	}

	if err := h.svc.UpdateContact(contact); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, contact)
}

// DeleteContact 删除联系人
func (h *AdvancedHandler) DeleteContact(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if err := h.svc.DeleteContact(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "contact deleted"})
}

// ListContacts 获取联系人列表
func (h *AdvancedHandler) ListContacts(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	deviceID := c.QueryParam("device_id")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	contacts, total, err := h.svc.ListContacts(tenantID, deviceID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"contacts": contacts,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// SearchContacts 搜索联系人
func (h *AdvancedHandler) SearchContacts(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	keyword := c.QueryParam("keyword")
	if keyword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "keyword is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	contacts, total, err := h.svc.SearchContacts(tenantID, keyword, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"contacts": contacts,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// ===== LDAPConfig Handlers =====

// CreateLDAPConfigRequest 创建LDAP配置请求
type CreateLDAPConfigRequest struct {
	Name         string `json:"name"`
	Server       string `json:"server"`
	Port         int    `json:"port"`
	UseSSL       bool   `json:"use_ssl"`
	BaseDN       string `json:"base_dn"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	UserFilter   string `json:"user_filter"`
	GroupFilter  string `json:"group_filter"`
	SyncInterval int    `json:"sync_interval"`
}

// CreateLDAPConfig 创建LDAP配置
func (h *AdvancedHandler) CreateLDAPConfig(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateLDAPConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.Server == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "server is required"})
	}
	if req.BaseDN == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "base_dn is required"})
	}

	config, err := h.svc.CreateLDAPConfig(tenantID, req.Name, req.Server, req.Port, req.UseSSL, req.BaseDN, req.BindDN, req.BindPassword, req.UserFilter, req.GroupFilter, req.SyncInterval)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, config)
}

// GetLDAPConfig 获取LDAP配置
func (h *AdvancedHandler) GetLDAPConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	config, err := h.svc.GetLDAPConfig(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "ldap config not found"})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateLDAPConfigRequest 更新LDAP配置请求
type UpdateLDAPConfigRequest struct {
	Name         string `json:"name"`
	Server       string `json:"server"`
	Port         int    `json:"port"`
	UseSSL       *bool  `json:"use_ssl"`
	BaseDN       string `json:"base_dn"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	UserFilter   string `json:"user_filter"`
	GroupFilter  string `json:"group_filter"`
	SyncInterval int    `json:"sync_interval"`
	Enabled      *bool  `json:"enabled"`
}

// UpdateLDAPConfig 更新LDAP配置
func (h *AdvancedHandler) UpdateLDAPConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	config, err := h.svc.GetLDAPConfig(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "ldap config not found"})
	}

	var req UpdateLDAPConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Server != "" {
		config.Server = req.Server
	}
	if req.Port > 0 {
		config.Port = req.Port
	}
	if req.UseSSL != nil {
		config.UseSSL = *req.UseSSL
	}
	if req.BaseDN != "" {
		config.BaseDN = req.BaseDN
	}
	if req.BindDN != "" {
		config.BindDN = req.BindDN
	}
	if req.BindPassword != "" {
		config.BindPassword = req.BindPassword
	}
	if req.UserFilter != "" {
		config.UserFilter = req.UserFilter
	}
	if req.GroupFilter != "" {
		config.GroupFilter = req.GroupFilter
	}
	if req.SyncInterval > 0 {
		config.SyncInterval = req.SyncInterval
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.svc.UpdateLDAPConfig(config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, config)
}

// DeleteLDAPConfig 删除LDAP配置
func (h *AdvancedHandler) DeleteLDAPConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if err := h.svc.DeleteLDAPConfig(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "ldap config deleted"})
}

// ListLDAPConfigs 获取LDAP配置列表
func (h *AdvancedHandler) ListLDAPConfigs(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	configs, err := h.svc.ListLDAPConfigs(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, configs)
}

// TestLDAPConnection 测试LDAP连接
func (h *AdvancedHandler) TestLDAPConnection(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	config, err := h.svc.GetLDAPConfig(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "ldap config not found"})
	}

	success, message, err := h.svc.TestLDAPConnection(config)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": success,
		"message": message,
	})
}

// SyncLDAPUsers 同步LDAP用户
func (h *AdvancedHandler) SyncLDAPUsers(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	users, err := h.svc.SyncLDAPUsers(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// ListLDAPUsers 获取LDAP用户列表
func (h *AdvancedHandler) ListLDAPUsers(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	users, total, err := h.svc.ListLDAPUsers(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// ===== WhiteLabelConfig Handlers =====

// CreateWhiteLabelConfigRequest 创建白标配置请求
type CreateWhiteLabelConfigRequest struct {
	Name           string `json:"name"`
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor      string `json:"text_color"`
	CustomCSS      string `json:"custom_css"`
	CustomJS       string `json:"custom_js"`
	FooterText     string `json:"footer_text"`
	LoginBgURL     string `json:"login_bg_url"`
	Enabled        bool   `json:"enabled"`
}

// CreateWhiteLabelConfig 创建白标配置
func (h *AdvancedHandler) CreateWhiteLabelConfig(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req CreateWhiteLabelConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	input := &services.WhiteLabelInput{
		LogoURL:         req.LogoURL,
		FaviconURL:      req.FaviconURL,
		PrimaryColor:    req.PrimaryColor,
		SecondaryColor:  req.SecondaryColor,
		AccentColor:     req.AccentColor,
		BackgroundColor: req.BackgroundColor,
		TextColor:      req.TextColor,
		CustomCSS:      req.CustomCSS,
		CustomJS:       req.CustomJS,
		FooterText:     req.FooterText,
		LoginBgURL:      req.LoginBgURL,
		Enabled:        req.Enabled,
	}

	config, err := h.svc.CreateWhiteLabelConfig(tenantID, req.Name, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, config)
}

// GetWhiteLabelConfig 获取白标配置
func (h *AdvancedHandler) GetWhiteLabelConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	config, err := h.svc.GetWhiteLabelConfig(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "white label config not found"})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateWhiteLabelConfigRequest 更新白标配置请求
type UpdateWhiteLabelConfigRequest struct {
	Name           string `json:"name"`
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor      string `json:"text_color"`
	CustomCSS      string `json:"custom_css"`
	CustomJS       string `json:"custom_js"`
	FooterText     string `json:"footer_text"`
	LoginBgURL     string `json:"login_bg_url"`
	Enabled        *bool  `json:"enabled"`
}

// UpdateWhiteLabelConfig 更新白标配置
func (h *AdvancedHandler) UpdateWhiteLabelConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	config, err := h.svc.GetWhiteLabelConfig(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "white label config not found"})
	}

	var req UpdateWhiteLabelConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		config.Name = req.Name
	}
	if req.LogoURL != "" {
		config.LogoURL = req.LogoURL
	}
	if req.FaviconURL != "" {
		config.FaviconURL = req.FaviconURL
	}
	if req.PrimaryColor != "" {
		config.PrimaryColor = req.PrimaryColor
	}
	if req.SecondaryColor != "" {
		config.SecondaryColor = req.SecondaryColor
	}
	if req.AccentColor != "" {
		config.AccentColor = req.AccentColor
	}
	if req.BackgroundColor != "" {
		config.BackgroundColor = req.BackgroundColor
	}
	if req.TextColor != "" {
		config.TextColor = req.TextColor
	}
	if req.CustomCSS != "" {
		config.CustomCSS = req.CustomCSS
	}
	if req.CustomJS != "" {
		config.CustomJS = req.CustomJS
	}
	if req.FooterText != "" {
		config.FooterText = req.FooterText
	}
	if req.LoginBgURL != "" {
		config.LoginBgURL = req.LoginBgURL
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.svc.UpdateWhiteLabelConfig(config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, config)
}

// DeleteWhiteLabelConfig 删除白标配置
func (h *AdvancedHandler) DeleteWhiteLabelConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if err := h.svc.DeleteWhiteLabelConfig(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "white label config deleted"})
}

// GetWhiteLabelConfigByTenant 根据租户获取白标配置
func (h *AdvancedHandler) GetWhiteLabelConfigByTenant(c echo.Context) error {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	config, err := h.svc.GetWhiteLabelConfigByTenant(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "white label config not found"})
	}

	return c.JSON(http.StatusOK, config)
}

// GenerateUUID 生成UUID (for testing)
func GenerateUUID() string {
	return uuid.New().String()
}
