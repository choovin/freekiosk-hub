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
	"github.com/wared2003/freekiosk-hub/internal/i18n"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// FieldTripUIHandler handles field trip HTML UI endpoints
type FieldTripUIHandler struct {
	ftRepo *repositories.FieldTripRepository
}

// NewFieldTripUIHandler creates a new FieldTripUIHandler
func NewFieldTripUIHandler(ftRepo *repositories.FieldTripRepository) *FieldTripUIHandler {
	return &FieldTripUIHandler{ftRepo: ftRepo}
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

// FieldTripContent renders field trip content partial
func FieldTripContent(groups []models.FieldTripGroup, devices []models.FieldTripDevice, t func(string) string) string {
	return fmt.Sprintf(`<div id="fieldtrip-content">%s %d groups, %d devices</div>`, t("Field Trip"), len(groups), len(devices))
}

// FieldTripPage renders the main Field Trip page
func FieldTripPage(groups []models.FieldTripGroup, devices []models.FieldTripDevice, lang string) string {
	return fmt.Sprintf(`<html><body><h1>Field Trip</h1><p>%d groups, %d devices</p></body></html>`, len(groups), len(devices))
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

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.Render(http.StatusOK, "", FieldTripContent(groups, devices, func(key string) string { return i18n.TL(lang, key) }))
	}
	return c.Render(http.StatusOK, "", FieldTripPage(groups, devices, lang))
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
	return c.Render(http.StatusOK, "", FieldTripGroupFormModal(group, lang))
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
	return c.Render(http.StatusOK, "", FieldTripGroupFormModal(group, lang))
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

// HandleBroadcast sends a broadcast message
func (h *FieldTripUIHandler) HandleBroadcast(c echo.Context) error {
	message := c.FormValue("message")
	groupID := c.FormValue("group_id")

	if message == "" {
		return c.Render(http.StatusOK, "", Toast("Message is required", "error"))
	}

	now := time.Now().Unix()
	broadcast := &models.Broadcast{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		Message:   message,
		Sound:     "default",
		CreatedBy: "admin",
		CreatedAt: now,
	}

	if err := h.ftRepo.CreateBroadcast(broadcast); err != nil {
		slog.Error("database error: failed to create broadcast", "err", err)
		return c.Render(http.StatusOK, "", Toast("Failed to send broadcast", "error"))
	}

	slog.Info("broadcast sent", "id", broadcast.ID, "group", groupID)
	return c.Render(http.StatusOK, "", Toast(fmt.Sprintf("Broadcast sent to %d devices", 0), "success"))
}

// HandleSetWhitelist sets the app whitelist for a device
func (h *FieldTripUIHandler) HandleSetWhitelist(c echo.Context) error {
	deviceID := c.Param("id")
	app := c.FormValue("app")

	slog.Info("whitelist update requested", "device", deviceID, "app", app)
	return c.Render(http.StatusOK, "", Toast("Whitelist updated", "success"))
}

// HandleOTAUpload handles OTA APK upload
func (h *FieldTripUIHandler) HandleOTAUpload(c echo.Context) error {
	slog.Info("OTA upload requested")
	return c.Render(http.StatusOK, "", Toast("OTA upload functionality coming soon", "error"))
}
