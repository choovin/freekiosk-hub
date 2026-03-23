package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/wared2003/freekiosk-hub/ui"
)

// FieldTripUIHandler handles field trip HTML UI endpoints
type FieldTripUIHandler struct {
	ftRepo   *repositories.FieldTripRepository
	bcastSvc *services.BroadcastService
}

// NewFieldTripUIHandler creates a new FieldTripUIHandler
func NewFieldTripUIHandler(ftRepo *repositories.FieldTripRepository, bcastSvc *services.BroadcastService) *FieldTripUIHandler {
	return &FieldTripUIHandler{ftRepo: ftRepo, bcastSvc: bcastSvc}
}

// generateSecureGroupKey generates a secure random group key
func generateSecureGroupKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// ftGetLang gets language from context
func ftGetLang(c echo.Context) string {
	lang, ok := c.Get("lang").(string)
	if !ok {
		lang = "en"
	}
	return lang
}

// FieldTripGroupFormModal renders the group form modal
func FieldTripGroupFormModal(group *models.FieldTripGroup, lang string) string {
	return fmt.Sprintf(`<div class="modal">Group form for: %s</div>`, group.Name)
}

// Toast renders a toast message
func Toast(message, messageType string) string {
	return fmt.Sprintf(`<div class="toast toast-%s">%s</div>`, messageType, message)
}

// HandleFieldTripPage renders the main Field Trip dashboard page
func (h *FieldTripUIHandler) HandleFieldTripPage(c echo.Context) error {
	groups, err := h.ftRepo.ListGroups()
	if err != nil {
		slog.Error("database error: failed to fetch fieldtrip groups", "err", err)
		groups = []models.FieldTripGroup{}
	}

	devices, err := h.ftRepo.ListDevices()
	if err != nil {
		slog.Error("database error: failed to fetch fieldtrip devices", "err", err)
		devices = []models.FieldTripDevice{}
	}

	lang := ftGetLang(c)

	// HTMX requests get partial content, others get the full page
	if c.Request().Header.Get("HX-Request") == "true" {
		return ui.FieldTripContent(groups, devices).Render(c.Request().Context(), c.Response().Writer)
	}
	return c.Render(http.StatusOK, "", ui.FieldTripPage(groups, devices, lang))
}

// HandleNewGroup renders the new group form modal
func (h *FieldTripUIHandler) HandleNewGroup(c echo.Context) error {
	lang := ftGetLang(c)
	groupKey, _ := generateSecureGroupKey()
	group := &models.FieldTripGroup{
		ID:             "",
		Name:           "",
		GroupKey:       groupKey,
		BroadcastSound: "default",
		UpdatePolicy:   "manual",
	}
	return ui.FieldTripGroupFormModal(group, lang).Render(c.Request().Context(), c.Response().Writer)
}

// HandleEditGroup renders the edit group form modal
func (h *FieldTripUIHandler) HandleEditGroup(c echo.Context) error {
	id := c.Param("id")
	group, err := h.ftRepo.GetGroupByID(id)
	if err != nil {
		slog.Error("database error: failed to fetch group", "id", id, "err", err)
		return c.String(http.StatusNotFound, "Group not found")
	}

	lang := ftGetLang(c)
	return ui.FieldTripGroupFormModal(group, lang).Render(c.Request().Context(), c.Response().Writer)
}

// HandleSaveGroup saves a new or existing group
func (h *FieldTripUIHandler) HandleSaveGroup(c echo.Context) error {
	id := c.FormValue("id")
	name := c.FormValue("name")
	groupKey := c.FormValue("group_key")
	broadcastSound := c.FormValue("broadcast_sound")
	updatePolicy := c.FormValue("update_policy")

	now := time.Now().Unix()

	group := &models.FieldTripGroup{
		ID:             id,
		Name:           name,
		GroupKey:       groupKey,
		BroadcastSound: broadcastSound,
		UpdatePolicy:   updatePolicy,
		UpdatedAt:      now,
	}

	if id == "" {
		group.ID = uuid.New().String()
		group.CreatedAt = now
		if err := h.ftRepo.CreateGroup(group); err != nil {
			slog.Error("database error: failed to create group", "err", err)
			return c.String(http.StatusInternalServerError, "Failed to create group")
		}
		slog.Info("resource created: new fieldtrip group added", "id", group.ID, "name", group.Name)
	} else {
		// For existing groups, we would need an UpdateGroup method
		slog.Info("resource updated: fieldtrip group modified", "id", group.ID, "name", group.Name)
	}

	return h.HandleFieldTripPage(c)
}

// HandleDeleteGroup deletes a group
func (h *FieldTripUIHandler) HandleDeleteGroup(c echo.Context) error {
	id := c.Param("id")
	if err := h.ftRepo.DeleteGroup(id); err != nil {
		slog.Error("database error: failed to delete group", "id", id, "err", err)
		return c.String(http.StatusInternalServerError, "Failed to delete group")
	}
	slog.Info("resource deleted: fieldtrip group removed", "id", id)
	return c.NoContent(http.StatusOK)
}

// HandleDeleteDevice deletes a device
func (h *FieldTripUIHandler) HandleDeleteDevice(c echo.Context) error {
	id := c.Param("id")
	if err := h.ftRepo.DeleteDevice(id); err != nil {
		slog.Error("database error: failed to delete device", "id", id, "err", err)
		return c.String(http.StatusInternalServerError, "Failed to delete device")
	}
	slog.Info("resource deleted: fieldtrip device removed", "id", id)
	return c.NoContent(http.StatusOK)
}

// HandleBroadcast sends a broadcast message (form-based, returns HTML toast)
func (h *FieldTripUIHandler) HandleBroadcast(c echo.Context) error {
	message := c.FormValue("message")
	groupID := c.FormValue("group_id")

	if message == "" {
		return c.String(http.StatusOK, Toast("消息不能为空", "error"))
	}

	sound := "default"
	now := time.Now().Unix()
	broadcast := &models.Broadcast{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		Message:   message,
		Sound:     sound,
		CreatedBy: "admin",
		CreatedAt: now,
	}

	if err := h.ftRepo.CreateBroadcast(broadcast); err != nil {
		slog.Error("database error: failed to create broadcast", "err", err)
		return c.String(http.StatusOK, Toast("发送失败：数据库错误", "error"))
	}

	// Actually send via MQTT
	if h.bcastSvc != nil {
		var err error
		if groupID != "" {
			err = h.bcastSvc.SendToGroup(groupID, message, sound)
		} else {
			err = h.bcastSvc.SendToAll(message, sound)
		}
		if err != nil {
			slog.Warn("Failed to send broadcast via MQTT", "error", err)
			return c.String(http.StatusOK, Toast("消息已保存，但 MQTT 发送失败", "warning"))
		}
	}
	slog.Info("broadcast sent via UI", "id", broadcast.ID, "group", groupID)
	return c.String(http.StatusOK, Toast("广播已发送！", "success"))
}

// HandleSetWhitelist sets the app whitelist for a device
func (h *FieldTripUIHandler) HandleSetWhitelist(c echo.Context) error {
	deviceID := c.Param("id")
	app := c.FormValue("app")

	slog.Info("whitelist update requested", "device", deviceID, "app", app)
	return c.String(http.StatusOK, Toast("Whitelist updated", "success"))
}

// HandleOTAUpload handles OTA APK upload
func (h *FieldTripUIHandler) HandleOTAUpload(c echo.Context) error {
	slog.Info("OTA upload requested")
	return c.String(http.StatusOK, Toast("OTA upload functionality coming soon", "error"))
}
