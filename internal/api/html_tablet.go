package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/dtos"
	"github.com/wared2003/freekiosk-hub/internal/i18n"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/wared2003/freekiosk-hub/ui"

	"github.com/labstack/echo/v4"
)

type HtmlTabletHandler struct {
	tabletRepo   repositories.TabletRepository
	reportRepo   repositories.ReportRepository
	groupRepo    repositories.GroupRepository
	kService     services.KioskService
	mediaService services.MediaService
	mqttService  services.MQTTServiceInterface
}

func NewHtmlTabletHandler(tr repositories.TabletRepository, rr repositories.ReportRepository, gr repositories.GroupRepository, ks services.KioskService, mes services.MediaService, ms services.MQTTServiceInterface) *HtmlTabletHandler {
	return &HtmlTabletHandler{tabletRepo: tr, reportRepo: rr, groupRepo: gr, kService: ks, mediaService: mes, mqttService: ms}
}

func (h *HtmlTabletHandler) HandleDetails(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "ID invalide")
	}

	tablet, err := h.tabletRepo.GetByID(id)
	if err != nil {
		return c.String(http.StatusNotFound, "Tablette non trouvée")
	}

	lastReport, _ := h.reportRepo.GetLatestByTablet(id, true)

	history, _ := h.reportRepo.GetHistory(id, 30)

	groups, _ := h.groupRepo.GetGroupsByTablet(id)

	td := dtos.TabletDisplay{
		Tablet:     *tablet,
		LastReport: lastReport,
		Groups:     groups,
	}

	lang := getLang(c)
	t := func(key string) string { return i18n.TL(lang, key) }

	if c.Request().Header.Get("HX-Request") != "true" {
		return c.Render(http.StatusOK, "", ui.TabletDetails(&td, history, true, lang))
	}

	// 2. Si c'est un refresh auto du SSE (on ajoute ?refresh=true dans le hx-get du template)
	if c.QueryParam("refresh") == "true" {
		return c.Render(http.StatusOK, "", ui.TabletUIInner(&td, history, t))
	}

	return c.Render(http.StatusOK, "", ui.TabletDetails(&td, history, false, lang))
}

func (h *HtmlTabletHandler) HandleBeep(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("invalid tablet id", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.Beep(services.Target{TabletID: id})
	if err != nil {
		return ui.Toast("error : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🔔 %s : Beep Send !", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : Error sending Beep ", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

func (h *HtmlTabletHandler) HandleReload(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("invalid tablet id ", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.Reload(services.Target{TabletID: id})
	if err != nil {
		return ui.Toast("Erreur : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🔄 %s : Reloading...", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : error reloading", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

func (h *HtmlTabletHandler) HandleReboot(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("invalid tablet id ", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.Reboot(services.Target{TabletID: id})
	if err != nil {
		return ui.Toast("Erreur : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🔄 %s : Rebooting", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : error reboot failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

func (h *HtmlTabletHandler) HandleNavigateModal(c echo.Context) error {
	idParam := c.Param("id")
	id, _ := strconv.ParseInt(idParam, 10, 64)

	// On peut optionnellement récupérer l'URL actuelle depuis la DB/Cache
	// pour pré-remplir l'input
	currentURL := ""

	return ui.NavigateModal(id, currentURL).Render(c.Request().Context(), c.Response().Writer)
}

func (h *HtmlTabletHandler) HandleNavigate(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("Invalid tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	newURL := c.FormValue("url")
	if newURL == "" {
		return ui.Toast("URL cannot be empty", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	parsedURL, err := url.ParseRequestURI(newURL)
	if err != nil {
		return ui.Toast("Invalid URL format", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ui.Toast("Forbidden protocol: Use HTTP or HTTPS", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.Navigate(services.Target{TabletID: id}, parsedURL.String())
	if err != nil {
		return ui.Toast("Error: "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🌐 %s: URL updated!", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s: Update failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}
	c.Response().Header().Set("HX-Trigger", "update")
	return nil
}

func (h *HtmlTabletHandler) HandleWakeUp(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("invalid tablet id ", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.Wake(services.Target{TabletID: id})
	if err != nil {
		return ui.Toast("Error : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("⏰ %s : Waked up", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : error waking up", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

func (h *HtmlTabletHandler) HandleScreenStatus(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("Invalid tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	statusRaw := c.FormValue("status")
	var shouldBeOn bool
	switch statusRaw {
	case "true", "on":
		shouldBeOn = true
	case "false", "off":
		shouldBeOn = false
	default:
		return ui.Toast("err: invalid request", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.SetScreen(services.Target{TabletID: id}, shouldBeOn)
	if err != nil {
		lang := getLang(c)
		ui.ScreenStatusBox(!shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
		return ui.Toast("Error: "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			lang := getLang(c)
			ui.ScreenStatusBox(shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
			ui.Toast(fmt.Sprintf("%s :screen command send", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			lang := getLang(c)
			ui.ScreenStatusBox(!shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
			ui.Toast(fmt.Sprintf("❌ %s: send screen command failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}
	return nil
}

func (h *HtmlTabletHandler) HandleScreenSaver(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("Invalid tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	statusRaw := c.FormValue("status")
	var shouldBeOn bool
	switch statusRaw {
	case "true", "on":
		shouldBeOn = true
	case "false", "off":
		shouldBeOn = false
	default:
		return ui.Toast("err: invalid request", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	report, err := h.kService.SetScreensaver(services.Target{TabletID: id}, shouldBeOn)
	if err != nil {
		lang := getLang(c)
		ui.ScreensaverStatusBox(!shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
		return ui.Toast("Error: "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			lang := getLang(c)
			ui.ScreensaverStatusBox(shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
			ui.Toast(fmt.Sprintf("%s :screensaver command send", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			lang := getLang(c)
			ui.ScreensaverStatusBox(!shouldBeOn, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
			ui.Toast(fmt.Sprintf("❌ %s: send screensaver command failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}
	return nil
}

func (h *HtmlTabletHandler) HandleSoundModal(c echo.Context) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	sounds, err := h.mediaService.List()
	if err != nil {
		return ui.Toast("Impossible de charger la bibliothèque", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	lang := getLang(c)
	return ui.TabSoundModal(sounds, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
}

func (h *HtmlTabletHandler) HandleUploadSound(c echo.Context) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	file, err := c.FormFile("soundFile")
	if err != nil {
		return ui.Toast("Fichier manquant", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	src, err := file.Open()
	if err != nil {
		return ui.Toast("Erreur lecture fichier", "error").Render(c.Request().Context(), c.Response().Writer)
	}
	defer src.Close()

	_, err = h.mediaService.Upload(file.Filename, src.(io.ReadSeeker))
	if err != nil {
		return ui.Toast(err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	sounds, _ := h.mediaService.List()
	lang := getLang(c)
	return ui.TabSoundList(sounds, id, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
}

func (h *HtmlTabletHandler) HandlePlaySound(c echo.Context) error {
	idParam := c.Param("id")
	id, _ := strconv.ParseInt(idParam, 10, 64)

	soundURL := c.FormValue("soundUrl")
	print("sound-url")
	volume, _ := strconv.Atoi(c.FormValue("volume"))
	loop := c.FormValue("loop") == "on"

	report, err := h.kService.PlayAudio(services.Target{TabletID: id}, soundURL, loop, volume)
	if err != nil {
		return ui.Toast("Erreur : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🔊 %s : Playback started", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : Playback failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}
	return nil
}

func (h *HtmlTabletHandler) HandleStopSound(c echo.Context) error {
	idParam := c.Param("id")
	id, _ := strconv.ParseInt(idParam, 10, 64)

	report, err := h.kService.StopAudio(services.Target{TabletID: id})
	if err != nil {
		return ui.Toast("Erreur : "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🛑 %s : Playback stopped", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s : Failed to stop", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

func (h *HtmlTabletHandler) HandleGtslTTSSound(c echo.Context) error {
	// 1. Get Tablet ID from URL
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return ui.Toast("Invalid Tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	// 2. Get Form Values
	text := c.FormValue("tts_text")
	if text == "" {
		return ui.Toast("Please enter some text", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	lang := c.FormValue("lang")
	if lang == "" {
		lang = "en"
	}

	loop := c.FormValue("loop") == "on"

	volumeStr := c.FormValue("volume")
	volume, _ := strconv.Atoi(volumeStr)
	if volume == 0 {
		volume = 100 // Default volume
	}

	safeText := url.QueryEscape(text)
	safeLang := url.QueryEscape(lang)
	ttsURL := fmt.Sprintf("https://translate.google.com/translate_tts?ie=UTF-8&tl=%s&client=tw-ob&q=%s", safeLang, safeText)

	report, err := h.kService.PlayAudio(services.Target{TabletID: id}, ttsURL, loop, volume)
	if err != nil {
		return ui.Toast("Service Error: "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("🗣️ %s: Announcement sent", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("❌ %s: Delivery failed", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

// HandleSetPinModal 显示 PIN 修改模态框
func (h *HtmlTabletHandler) HandleSetPinModal(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ui.Toast("Invalid tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	tablet, err := h.tabletRepo.GetByID(id)
	if err != nil {
		return ui.Toast("Tablet not found", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	lang := getLang(c)
	return ui.SetPinModal(tablet.ID, tablet.Name, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
}

// HandleSetPin 处理 PIN 修改请求
func (h *HtmlTabletHandler) HandleSetPin(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ui.Toast("Invalid tablet ID", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	newPin := c.FormValue("pin")
	if newPin == "" {
		return ui.Toast("PIN cannot be empty", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	// Validate PIN format (4-digit numeric)
	if len(newPin) != 4 || !isNumeric(newPin) {
		return ui.Toast("PIN must be 4 digits", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	tablet, err := h.tabletRepo.GetByID(id)
	if err != nil {
		return ui.Toast("Tablet not found", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	// Send PIN change command via MQTT
	if h.mqttService != nil && h.mqttService.IsConnected() {
		// Use device ID (tablet name) for MQTT command
		deviceID := tablet.Name
		if deviceID == "" {
			deviceID = fmt.Sprintf("device_%d", tablet.ID)
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
		defer cancel()

		_, err := h.mqttService.SendCommand(ctx, deviceID, models.CommandSetPin, map[string]interface{}{
			"pin": newPin,
		}, 30*time.Second)

		if err != nil {
			slog.Warn("Failed to send PIN change command via MQTT", "error", err)
			return ui.Toast("Failed to send PIN change command", "error").Render(c.Request().Context(), c.Response().Writer)
		}

		slog.Info("PIN change command sent", "tablet_id", tablet.ID, "device_id", deviceID)
		return ui.Toast(fmt.Sprintf("PIN change sent to %s", tablet.Name), "success").Render(c.Request().Context(), c.Response().Writer)
	}

	// MQTT not available - try HTTP fallback
	report, err := h.kService.SendRemoteCommand(services.Target{TabletID: id}, fmt.Sprintf("setPin:%s", newPin))
	if err != nil {
		return ui.Toast("Failed to change PIN: "+err.Error(), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	for _, res := range report.Results {
		if res.Executed {
			ui.Toast(fmt.Sprintf("PIN changed for %s", res.Name), "success").Render(c.Request().Context(), c.Response().Writer)
		} else {
			ui.Toast(fmt.Sprintf("Failed to change PIN for %s", res.Name), "error").Render(c.Request().Context(), c.Response().Writer)
		}
	}

	return nil
}

// HandleBulkSetPinModal 显示批量 PIN 修改模态框
func (h *HtmlTabletHandler) HandleBulkSetPinModal(c echo.Context) error {
	deviceIDs := c.QueryParam("device_ids")
	lang := getLang(c)
	return ui.BulkSetPinModal(deviceIDs, func(key string) string { return i18n.TL(lang, key) }).Render(c.Request().Context(), c.Response().Writer)
}

// HandleBulkSetPin 处理批量 PIN 修改请求
func (h *HtmlTabletHandler) HandleBulkSetPin(c echo.Context) error {
	deviceIDsStr := c.FormValue("device_ids")
	if deviceIDsStr == "" {
		return ui.Toast("No devices selected", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	newPin := c.FormValue("pin")
	if newPin == "" {
		return ui.Toast("PIN cannot be empty", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	// Validate PIN format (4-digit numeric)
	if len(newPin) != 4 || !isNumeric(newPin) {
		return ui.Toast("PIN must be 4 digits", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	// Parse device IDs
	var deviceIDs []int64
	for _, idStr := range strings.Split(deviceIDsStr, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			deviceIDs = append(deviceIDs, id)
		}
	}

	if len(deviceIDs) == 0 {
		return ui.Toast("No valid devices selected", "error").Render(c.Request().Context(), c.Response().Writer)
	}

	successCount := 0
	failCount := 0

	for _, id := range deviceIDs {
		tablet, err := h.tabletRepo.GetByID(id)
		if err != nil {
			failCount++
			continue
		}

		deviceID := tablet.Name
		if deviceID == "" {
			deviceID = fmt.Sprintf("device_%d", tablet.ID)
		}

		if h.mqttService != nil && h.mqttService.IsConnected() {
			ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
			_, err := h.mqttService.SendCommand(ctx, deviceID, models.CommandSetPin, map[string]interface{}{
				"pin": newPin,
			}, 30*time.Second)
			cancel()

			if err == nil {
				successCount++
			} else {
				failCount++
			}
		} else {
			// HTTP fallback
			_, err := h.kService.SendRemoteCommand(services.Target{TabletID: id}, fmt.Sprintf("setPin:%s", newPin))
			if err == nil {
				successCount++
			} else {
				failCount++
			}
		}
	}

	if successCount > 0 {
		ui.Toast(fmt.Sprintf("PIN change sent to %d device(s)", successCount), "success").Render(c.Request().Context(), c.Response().Writer)
	}
	if failCount > 0 {
		ui.Toast(fmt.Sprintf("Failed for %d device(s)", failCount), "error").Render(c.Request().Context(), c.Response().Writer)
	}

	return nil
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
