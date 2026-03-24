package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// FieldTripHandler handles field trip API endpoints
type FieldTripHandler struct {
	Repo          *repositories.FieldTripRepository
	SigningPubKey string
	BcastSvc      *services.BroadcastService
	MqttBrokerURL string // MQTT broker URL for tablets
	MqttPort      int    // MQTT broker port
}

// NewFieldTripHandler creates a new FieldTripHandler
func NewFieldTripHandler(repo *repositories.FieldTripRepository, signingPubKey string, bcastSvc *services.BroadcastService, mqttBrokerURL string, mqttPort int) *FieldTripHandler {
	return &FieldTripHandler{
		Repo:          repo,
		SigningPubKey: signingPubKey,
		BcastSvc:      bcastSvc,
		MqttBrokerURL: mqttBrokerURL,
		MqttPort:      mqttPort,
	}
}

// generateGroupKey generates a secure random group key using crypto/rand
func generateGroupKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateAPIKey generates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// hashAPIKey hashes an API key using SHA256 for storage
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// validateApiKey validates the provided API key against the expected hash
func (h *FieldTripHandler) validateApiKey(c echo.Context, expectedHash string) error {
	providedKey := c.Request().Header.Get("X-Api-Key")
	if providedKey == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing API key")
	}
	providedHash := hashAPIKey(providedKey)
	if providedHash != expectedHash {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
	}
	return nil
}

// CreateGroup creates a new field trip group
// @Summary 创建分组
// @Description 创建新的野外考察分组
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param request body CreateGroupInput true "创建分组请求"
// @Success 201 {object} models.FieldTripGroup
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/groups [post]
func (h *FieldTripHandler) CreateGroup(c echo.Context) error {
	var input struct {
		Name           string `json:"name" validate:"required"`
		BroadcastSound string `json:"broadcast_sound"`
		UpdatePolicy   string `json:"update_policy"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	groupKey, err := generateGroupKey()
	if err != nil {
		slog.Error("Failed to generate group key", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate group key")
	}

	now := time.Now().Unix()
	group := &models.FieldTripGroup{
		ID:              uuid.New().String(),
		Name:            input.Name,
		GroupKey:        groupKey,
		BroadcastSound:  input.BroadcastSound,
		UpdatePolicy:    input.UpdatePolicy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if group.BroadcastSound == "" {
		group.BroadcastSound = "default"
	}
	if group.UpdatePolicy == "" {
		group.UpdatePolicy = "manual"
	}

	if err := h.Repo.CreateGroup(group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create group")
	}

	return c.JSON(http.StatusCreated, group)
}

// ListGroups lists all field trip groups
// @Summary 列出分组
// @Description 获取所有野外考察分组
// @Tags 野外考察
// @Produce json
// @Success 200 {array} models.FieldTripGroup
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/groups [get]
func (h *FieldTripHandler) ListGroups(c echo.Context) error {
	groups, err := h.Repo.ListGroups()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list groups")
	}
	return c.JSON(http.StatusOK, groups)
}

// DeleteGroup deletes a field trip group
// @Summary 删除分组
// @Description 删除指定的野外考察分组
// @Tags 野外考察
// @Produce json
// @Param id path string true "分组ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/groups/{id} [delete]
func (h *FieldTripHandler) DeleteGroup(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing group id")
	}

	// Check if group exists
	_, err := h.Repo.GetGroupByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	if err := h.Repo.DeleteGroup(id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete group")
	}

	return c.NoContent(http.StatusNoContent)
}

// CreateDevice creates a new field trip device and returns QR payload
// @Summary 创建设备
// @Description 创建新的野外考察设备并返回二维码信息
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param request body CreateDeviceInput true "创建设备请求"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices [post]
func (h *FieldTripHandler) CreateDevice(c echo.Context) error {
	var input struct {
		Name     string `json:"name" validate:"required"`
		GroupKey string `json:"group_key" validate:"required"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.GroupKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and group_key are required")
	}

	// Verify group exists
	group, err := h.Repo.GetGroupByKey(input.GroupKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		slog.Error("Failed to generate API key", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate API key")
	}

	now := time.Now().Unix()
	device := &models.FieldTripDevice{
		ID:            uuid.New().String(),
		Name:          input.Name,
		GroupID:       group.ID,
		ApiKeyHash:    hashAPIKey(apiKey),
		HubURL:        h.Repo.GetHubURL(),
		Status:        "pending",
		SigningPubKey: h.SigningPubKey,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.Repo.CreateDevice(device); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create device")
	}

	// Cache the plaintext API key for QR PDF export (expires in 30 days)
	if err := h.Repo.CacheAPIKey(device.ID, apiKey); err != nil {
		slog.Warn("Failed to cache API key", "device_id", device.ID, "error", err)
	}

	// Return QR payload with sensitive data
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"device_id": device.ID,
		"name":      device.Name,
		"group_id":  group.ID,
		"group_key": input.GroupKey,
		"api_key":   apiKey,
		"hub_url":   device.HubURL,
	})
}

// BulkCreateDevices creates multiple devices at once
// @Summary 批量创建设备
// @Description 批量创建多个野外考察设备
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param request body BulkCreateDevicesInput true "批量创建设备请求"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/bulk [post]
func (h *FieldTripHandler) BulkCreateDevices(c echo.Context) error {
	var input struct {
		GroupID string `json:"group_id"`
		Count   int    `json:"count"`
		Prefix  string `json:"prefix"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if input.GroupID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group_id is required")
	}
	if input.Count <= 0 || input.Count > 200 {
		return echo.NewHTTPError(http.StatusBadRequest, "count must be between 1 and 200")
	}
	if input.Prefix == "" {
		input.Prefix = "平板-"
	}

	// Verify group exists
	group, err := h.Repo.GetGroupByID(input.GroupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	var devices []map[string]interface{}
	for i := 1; i <= input.Count; i++ {
		name := fmt.Sprintf("%s%02d", input.Prefix, i)

		apiKey, err := generateAPIKey()
		if err != nil {
			slog.Error("Failed to generate API key", "error", err)
			continue
		}

		now := time.Now().Unix()
		device := &models.FieldTripDevice{
			ID:            uuid.New().String(),
			Name:          name,
			GroupID:       group.ID,
			ApiKeyHash:    hashAPIKey(apiKey),
			HubURL:        h.Repo.GetHubURL(),
			Status:        "pending",
			SigningPubKey: h.SigningPubKey,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := h.Repo.CreateDevice(device); err != nil {
			slog.Error("Failed to create device", "name", name, "error", err)
			continue
		}

		// Cache the plaintext API key for QR PDF export
		if err := h.Repo.CacheAPIKey(device.ID, apiKey); err != nil {
			slog.Warn("Failed to cache API key", "device_id", device.ID, "error", err)
		}

		devices = append(devices, map[string]interface{}{
			"device_id": device.ID,
			"name":      device.Name,
			"api_key":   apiKey,
			"group_key": group.GroupKey,
			"hub_url":   device.HubURL,
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"count":   len(devices),
		"devices": devices,
	})
}

// ListDevices lists all field trip devices
// @Summary 列出设备
// @Description 获取所有野外考察设备
// @Tags 野外考察
// @Produce json
// @Success 200 {array} models.FieldTripDevice
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices [get]
func (h *FieldTripHandler) ListDevices(c echo.Context) error {
	devices, err := h.Repo.ListDevices()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list devices")
	}
	return c.JSON(http.StatusOK, devices)
}

// DeleteDevice deletes a field trip device
// @Summary 删除设备
// @Description 删除指定的野外考察设备
// @Tags 野外考察
// @Produce json
// @Param id path string true "设备ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/{id} [delete]
func (h *FieldTripHandler) DeleteDevice(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	// Check if device exists
	_, err := h.Repo.GetDeviceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	if err := h.Repo.DeleteDevice(id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete device")
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateDevice updates a field trip device (name only)
// @Summary 更新设备
// @Description 更新野外考察设备信息（仅名称）
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body UpdateDeviceInput true "更新设备请求"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/{id} [patch]
func (h *FieldTripHandler) UpdateDevice(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	// Check if device exists
	_, err := h.Repo.GetDeviceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	now := time.Now().Unix()
	if err := h.Repo.UpdateDeviceName(id, input.Name, now); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update device")
	}

	return c.NoContent(http.StatusNoContent)
}

// BindDevice handles tablet QR code binding
// @Summary 绑定设备
// @Description 处理平板二维码绑定
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param request body models.BindRequest true "绑定请求"
// @Success 200 {object} models.BindResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/bind [post]
func (h *FieldTripHandler) BindDevice(c echo.Context) error {
	var req models.BindRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.DeviceID == "" || req.GroupKey == "" || req.ApiKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "device_id, group_key, and api_key are required")
	}

	// Verify device exists
	device, err := h.Repo.GetDeviceByID(req.DeviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	// Verify API key matches
	providedHash := hashAPIKey(req.ApiKey)
	if providedHash != device.ApiKeyHash {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
	}

	// Verify group key matches
	group, err := h.Repo.GetGroupByKey(req.GroupKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	// Update device with group association and status
	now := time.Now().Unix()
	device.GroupID = group.ID
	device.Status = "active"
	device.HubURL = req.HubURL
	device.UpdatedAt = now

	// Update in database
	if err := h.Repo.SetDeviceStatus(device.ID, "active", now); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to bind device")
	}

	resp := models.BindResponse{
		DeviceID:       device.ID,
		GroupID:        group.ID,
		GroupName:      group.Name,
		DeviceName:     device.Name,
		SigningPubKey:  h.SigningPubKey,
		BroadcastSound: group.BroadcastSound,
		UpdatePolicy:   group.UpdatePolicy,
		MqttBrokerURL:  h.MqttBrokerURL,
		MqttPort:       h.MqttPort,
	}

	return c.JSON(http.StatusOK, resp)
}

// ReportLocation reports GPS location from a device
// @Summary 报告位置
// @Description 设备上报GPS位置信息
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body models.GPSReport true "GPS报告"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/{id}/location [post]
func (h *FieldTripHandler) ReportLocation(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	var report models.GPSReport
	if err := c.Bind(&report); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	report.DeviceID = id
	now := time.Now().Unix()

	// Update device location
	if err := h.Repo.UpdateDeviceLocation(id, report.Lat, report.Lng, now); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update location")
	}

	// Insert GPS log
	if err := h.Repo.InsertGPSLog(id, report.Lat, report.Lng, report.Accuracy, report.Timestamp, now); err != nil {
		slog.Warn("Failed to insert GPS log", "device_id", id, "error", err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetLocationHistory returns GPS history for a device
// @Summary 获取位置历史
// @Description 获取设备的位置历史记录
// @Tags 野外考察
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/{id}/location/history [get]
func (h *FieldTripHandler) GetLocationHistory(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	history, err := h.Repo.GetGPSHistory(id, 100)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get location history")
	}

	return c.JSON(http.StatusOK, history)
}

// PollCommands returns pending commands for a device
// @Summary 拉取命令
// @Description 设备拉取待处理的命令
// @Tags 野外考察
// @Produce json
// @Param device_id query string true "设备ID"
// @Success 200 {object} models.CommandPoll
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/commands [get]
func (h *FieldTripHandler) PollCommands(c echo.Context) error {
	deviceID := c.QueryParam("device_id")
	if deviceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "device_id is required")
	}

	// Validate device exists
	device, err := h.Repo.GetDeviceByID(deviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	// Validate API key
	if err := h.validateApiKey(c, device.ApiKeyHash); err != nil {
		return err
	}

	// Pop pending commands
	cmds, err := h.Repo.PopPendingCommands(deviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get commands")
	}

	// Build response
	resp := models.CommandPoll{}
	for _, cmd := range cmds {
		switch cmd.CommandType {
		case "whitelist":
			var whitelist []string
			if err := json.Unmarshal([]byte(cmd.Payload), &whitelist); err == nil {
				resp.Whitelist = whitelist
			}
		case "ota":
			resp.OTAURL = cmd.Payload
		case "broadcast":
			resp.Broadcast = cmd.Payload
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// SetWhitelist sets the app whitelist for a device
// @Summary 设置白名单
// @Description 设置设备的应用白名单
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body SetWhitelistInput true "白名单请求"
// @Success 202 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/devices/{id}/whitelist [post]
func (h *FieldTripHandler) SetWhitelist(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	var input struct {
		Whitelist []string `json:"whitelist"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate device exists
	_, err := h.Repo.GetDeviceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	// Serialize whitelist to JSON
	payload, err := json.Marshal(input.Whitelist)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to serialize whitelist")
	}

	cmd := &models.PendingCommand{
		ID:           uuid.New().String(),
		DeviceID:     id,
		CommandType:  "whitelist",
		Payload:      string(payload),
		Status:       "pending",
		CreatedAt:    time.Now().Unix(),
	}

	if err := h.Repo.PushPendingCommand(cmd); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set whitelist")
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "pending"})
}

// SendBroadcast sends a broadcast message to a group or all devices
// @Summary 发送广播
// @Description 向分组或所有设备发送广播消息
// @Tags 野外考察
// @Accept json
// @Produce json
// @Param request body SendBroadcastInput true "广播请求"
// @Success 201 {object} models.Broadcast
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/fieldtrip/broadcast [post]
func (h *FieldTripHandler) SendBroadcast(c echo.Context) error {
	var input struct {
		GroupID   string `json:"group_id"`
		Message   string `json:"message" validate:"required"`
		Sound     string `json:"sound"`
		CreatedBy string `json:"created_by"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if input.Message == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "message is required")
	}

	if input.Sound == "" {
		input.Sound = "default"
	}
	if input.CreatedBy == "" {
		input.CreatedBy = "system"
	}

	broadcast := &models.Broadcast{
		ID:         uuid.New().String(),
		GroupID:    input.GroupID,
		Message:    input.Message,
		Sound:      input.Sound,
		CreatedBy:  input.CreatedBy,
		CreatedAt:  time.Now().Unix(),
	}

	if err := h.Repo.CreateBroadcast(broadcast); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create broadcast")
	}

	// Send via MQTT if BroadcastService is available
	if h.BcastSvc != nil {
		if input.GroupID != "" {
			if err := h.BcastSvc.SendToGroup(input.GroupID, input.Message, input.Sound); err != nil {
				slog.Warn("Failed to send broadcast via MQTT", "error", err)
			}
		} else {
			if err := h.BcastSvc.SendToAll(input.Message, input.Sound); err != nil {
				slog.Warn("Failed to send broadcast to all via MQTT", "error", err)
			}
		}
	}

	return c.JSON(http.StatusCreated, broadcast)
}

// HandleGetDeviceQR returns a single device's QR code as PNG
// GET /api/v2/fieldtrip/devices/:id/qr
func (h *FieldTripHandler) HandleGetDeviceQR(c echo.Context) error {
	deviceID := c.Param("id")
	if deviceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	// Get device
	device, err := h.Repo.GetDeviceByID(deviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	// Get group
	group, err := h.Repo.GetGroupByID(device.GroupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	// Get cached API key
	apiKey := "[KEY_NOT_FOUND]"
	if cachedKey, err := h.Repo.GetCachedAPIKey(device.ID); err == nil {
		apiKey = cachedKey
	}

	// Create QR payload
	payload := QRPayload{
		DeviceID: device.ID,
		GroupKey: group.GroupKey,
		APIKey:   apiKey,
		HubURL:   device.HubURL,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate QR payload")
	}

	// Generate QR code PNG (256x256)
	pngData, err := qrcode.Encode(string(payloadBytes), qrcode.Medium, 256)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate QR code")
	}

	// Return PNG image
	c.Response().Header().Set("Content-Type", "image/png")
	c.Response().Write(pngData)
	return nil
}

// CreateGroupInput 创建分组输入
type CreateGroupInput struct {
	Name           string `json:"name"`
	BroadcastSound string `json:"broadcast_sound"`
	UpdatePolicy   string `json:"update_policy"`
}

// CreateDeviceInput 创建设备输入
type CreateDeviceInput struct {
	Name     string `json:"name"`
	GroupKey string `json:"group_key"`
}

// BulkCreateDevicesInput 批量创建设备输入
type BulkCreateDevicesInput struct {
	GroupID string `json:"group_id"`
	Count   int    `json:"count"`
	Prefix  string `json:"prefix"`
}

// UpdateDeviceInput 更新设备输入
type UpdateDeviceInput struct {
	Name string `json:"name"`
}

// SetWhitelistInput 设置白名单输入
type SetWhitelistInput struct {
	Whitelist []string `json:"whitelist"`
}

// SendBroadcastInput 发送广播输入
type SendBroadcastInput struct {
	GroupID   string `json:"group_id"`
	Message   string `json:"message"`
	Sound     string `json:"sound"`
	CreatedBy string `json:"created_by"`
}

// ReportDeviceInfo handles device system information reporting from tablet
// POST /api/v2/fieldtrip/devices/:id/info
func (h *FieldTripHandler) ReportDeviceInfo(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	var info models.FieldTripDeviceInfo
	if err := c.Bind(&info); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Serialize to JSON for storage
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to serialize device info")
	}

	now := time.Now().Unix()
	if err := h.Repo.UpdateDeviceInfo(id, string(infoJSON), now); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update device info")
	}

	return c.NoContent(http.StatusNoContent)
}

// GetDeviceInfo returns stored device system information
// GET /api/v2/fieldtrip/devices/:id/info
func (h *FieldTripHandler) GetDeviceInfo(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	infoJSON, err := h.Repo.GetDeviceInfo(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	if infoJSON == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{})
	}

	var info models.FieldTripDeviceInfo
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse device info")
	}

	return c.JSON(http.StatusOK, info)
}

// GetDeviceConfig returns stored device configuration
// GET /api/v2/fieldtrip/devices/:id/config
func (h *FieldTripHandler) GetDeviceConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	configJSON, err := h.Repo.GetDeviceConfig(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	if configJSON == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{})
	}

	var config models.DeviceConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse device config")
	}

	return c.JSON(http.StatusOK, config)
}

// SetDeviceConfig sets device configuration
// POST /api/v2/fieldtrip/devices/:id/config
func (h *FieldTripHandler) SetDeviceConfig(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing device id")
	}

	var config models.DeviceConfig
	if err := c.Bind(&config); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to serialize device config")
	}

	now := time.Now().Unix()
	if err := h.Repo.UpdateDeviceConfig(id, string(configJSON), now); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update device config")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
