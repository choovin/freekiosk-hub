package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// ExportHandler handles QR code PDF export
type ExportHandler struct {
	Repo        *repositories.FieldTripRepository
	ServerPort  string
}

// NewExportHandler creates a new ExportHandler
func NewExportHandler(repo *repositories.FieldTripRepository, serverPort string) *ExportHandler {
	return &ExportHandler{Repo: repo, ServerPort: serverPort}
}

// QRPayload represents the JSON payload stored in QR codes
type QRPayload struct {
	DeviceID string `json:"device_id"`
	GroupKey string `json:"group_key"`
	APIKey   string `json:"api_key"`
	HubURL   string `json:"hub_url"`
}

// deviceQR holds a device and its generated QR code data
type deviceQR struct {
	device   *models.FieldTripDevice
	groupKey string
	payload  string
	pngData  []byte
}

// HandleExportPDF generates a PDF with QR codes for all devices in a group
// GET /api/v2/fieldtrip/groups/:id/export?layout=6up|4up|1up
func (h *ExportHandler) HandleExportPDF(c echo.Context) error {
	groupID := c.Param("id")
	if groupID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing group id")
	}

	layout := c.QueryParam("layout")
	if layout == "" {
		layout = "6up"
	}

	// Validate layout
	if layout != "6up" && layout != "4up" && layout != "1up" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid layout, must be 6up, 4up, or 1up")
	}

	// Get group
	group, err := h.Repo.GetGroupByID(groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	// Get devices in group
	devices, err := h.Repo.ListDevicesByGroup(groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get devices")
	}

	if len(devices) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no devices in this group")
	}

	// Generate QR payload for each device
	deviceQRs := make([]deviceQR, 0, len(devices))
	for i, device := range devices {
		// Retrieve cached plaintext API key for QR code
		apiKey := "[KEY_NOT_FOUND]"
		if cachedKey, err := h.Repo.GetCachedAPIKey(device.ID); err == nil {
			apiKey = cachedKey
		}

		payload := QRPayload{
			DeviceID: device.ID,
			GroupKey: group.GroupKey,
			APIKey:   apiKey,
			HubURL:   device.HubURL,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}

		payloadStr := string(payloadBytes)

		// Generate QR code PNG
		pngData, err := qrcode.Encode(payloadStr, qrcode.Medium, 256)
		if err != nil {
			continue
		}

		deviceQRs = append(deviceQRs, deviceQR{
			device:   &devices[i], // Take address of array element to avoid range loop issues
			groupKey: group.GroupKey,
			payload:  payloadStr,
			pngData:  pngData,
		})
	}

	if len(deviceQRs) == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate QR codes")
	}

	// Generate PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// Calculate total pages needed
	var perPage int
	switch layout {
	case "6up":
		perPage = 6
	case "4up":
		perPage = 4
	case "1up":
		perPage = 1
	}
	totalPages := (len(deviceQRs) + perPage - 1) / perPage

	// Add pages based on layout
	switch layout {
	case "6up":
		h.add6UpPages(pdf, deviceQRs, group.Name, totalPages)
	case "4up":
		h.add4UpPages(pdf, deviceQRs, group.Name, totalPages)
	case "1up":
		h.add1UpPages(pdf, deviceQRs, group.Name, totalPages)
	}

	// Output to buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate PDF")
	}

	// Set response headers
	filename := fmt.Sprintf("group-%s-qr-%s.pdf", group.Name, layout)
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Response().Write(buf.Bytes())

	return nil
}

// HandleGetGroupQR returns the group QR payload as JSON
// GET /api/v2/fieldtrip/groups/:id/qr
func (h *ExportHandler) HandleGetGroupQR(c echo.Context) error {
	groupID := c.Param("id")
	if groupID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing group id")
	}

	group, err := h.Repo.GetGroupByID(groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	payload := map[string]interface{}{
		"type":       "group",
		"hub_url":    h.Repo.GetHubURL(),
		"group_id":   group.ID,
		"group_key":  group.GroupKey,
		"group_name": group.Name,
	}

	return c.JSON(http.StatusOK, payload)
}

// HandleGetAPKQR returns the APK download URL as a QR code PNG
// GET /api/v2/fieldtrip/apk-qr
func (h *ExportHandler) HandleGetAPKQR(c echo.Context) error {
	apkInfo, err := h.getLatestAPK()
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "no APK available")
	}

	pngData, err := qrcode.Encode(apkInfo.DownloadURL, qrcode.Medium, 256)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate QR code")
	}

	c.Response().Header().Set("Content-Type", "image/png")
	c.Response().Write(pngData)
	return nil
}

// getLatestAPK finds the most recently modified APK file in the apk directory
func (h *ExportHandler) getLatestAPK() (*apkInfo, error) {
	dir := "apk"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("APK directory does not exist: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read APK directory: %w", err)
	}

	var latestFile os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if latestFile == nil || info.ModTime().After(latestFile.ModTime()) {
			latestFile = info
		}
	}

	if latestFile == nil {
		return nil, fmt.Errorf("no APK files found")
	}

	filename := latestFile.Name()
	downloadURL := fmt.Sprintf("http://localhost:%s/apk/%s", h.ServerPort, filename)

	return &apkInfo{
		Filename:    filename,
		DownloadURL: downloadURL,
	}, nil
}

type apkInfo struct {
	Filename    string
	DownloadURL string
}

// add6UpPages adds multiple pages with 6 QR codes each (2 rows x 3 columns)
func (h *ExportHandler) add6UpPages(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string, totalPages int) {
	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	cellWidth := 60.0
	cellHeight := 130.0
	qrSize := 40.0
	colCount := 3

	for pageNum := 0; pageNum < totalPages; pageNum++ {
		pdf.AddPage()

		// Title
		pdf.SetFont("Arial", "B", 14)
		pdf.SetXY(margin, margin)
		pdf.CellFormat(180, 10, fmt.Sprintf("FreeKiosk 研学版 — %s 设备二维码", groupName), "", 1, "C", false, 0, "")

		// Print date
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(margin, margin+6)
		pdf.CellFormat(180, 5, fmt.Sprintf("打印日期: %s", time.Now().Format("2006-01-02")), "", 1, "C", false, 0, "")
		pdf.Ln(5)

		startY := pdf.GetY()

		startIdx := pageNum * 6
		endIdx := startIdx + 6
		if endIdx > len(deviceQRs) {
			endIdx = len(deviceQRs)
		}

		for i := startIdx; i < endIdx; i++ {
			idx := i - startIdx
			dqr := deviceQRs[i]

			col := idx % colCount
			row := idx / colCount

			x := margin + float64(col)*cellWidth
			y := startY + float64(row)*cellHeight

			// Add QR code centered in cell (use in-memory loading)
			qrX := x + (cellWidth-qrSize)/2
			imgName := fmt.Sprintf("qr_%d_%d", pageNum, idx)
			r := bytes.NewReader(dqr.pngData)
			pdf.RegisterImageReader(imgName, "png", r)
			pdf.ImageOptions(imgName, qrX, y+5, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

			// Add device name below QR
			pdf.SetXY(x, y+qrSize+8)
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(cellWidth, 5, dqr.device.Name, "", 1, "C", false, 0, "")
		}

		// Add footer with page number
		pdf.SetY(pageHeight - 15)
		pdf.SetFont("Arial", "I", 7)
		pdf.CellFormat(0, 5, fmt.Sprintf("FreeKiosk 研学版  |  第 %d/%d 页", pageNum+1, totalPages), "", 1, "C", false, 0, "")
	}
}

// add4UpPages adds multiple pages with 4 QR codes each (2 rows x 2 columns)
func (h *ExportHandler) add4UpPages(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string, totalPages int) {
	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	cellWidth := 90.0
	cellHeight := 130.0
	qrSize := 60.0
	colCount := 2

	for pageNum := 0; pageNum < totalPages; pageNum++ {
		pdf.AddPage()

		// Title
		pdf.SetFont("Arial", "B", 16)
		pdf.SetXY(margin, margin)
		pdf.CellFormat(180, 10, fmt.Sprintf("FreeKiosk 研学版 — %s 设备二维码", groupName), "", 1, "C", false, 0, "")

		// Print date
		pdf.SetFont("Arial", "", 9)
		pdf.SetXY(margin, margin+6)
		pdf.CellFormat(180, 5, fmt.Sprintf("打印日期: %s", time.Now().Format("2006-01-02")), "", 1, "C", false, 0, "")
		pdf.Ln(5)

		startY := pdf.GetY()

		startIdx := pageNum * 4
		endIdx := startIdx + 4
		if endIdx > len(deviceQRs) {
			endIdx = len(deviceQRs)
		}

		for i := startIdx; i < endIdx; i++ {
			idx := i - startIdx
			dqr := deviceQRs[i]

			col := idx % colCount
			row := idx / colCount

			x := margin + float64(col)*cellWidth
			y := startY + float64(row)*cellHeight

			// Add device name above QR
			pdf.SetXY(x, y)
			pdf.SetFont("Arial", "B", 11)
			pdf.CellFormat(cellWidth, 6, dqr.device.Name, "", 1, "C", false, 0, "")

			// Add QR code centered in cell (use in-memory loading)
			qrX := x + (cellWidth-qrSize)/2
			imgName := fmt.Sprintf("qr_%d_%d", pageNum, idx)
			r := bytes.NewReader(dqr.pngData)
			pdf.RegisterImageReader(imgName, "png", r)
			pdf.ImageOptions(imgName, qrX, y+8, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}

		// Add footer with page number
		pdf.SetY(pageHeight - 15)
		pdf.SetFont("Arial", "I", 7)
		pdf.CellFormat(0, 5, fmt.Sprintf("FreeKiosk 研学版  |  第 %d/%d 页", pageNum+1, totalPages), "", 1, "C", false, 0, "")
	}
}

// add1UpPages adds multiple pages with 1 large QR code per page
func (h *ExportHandler) add1UpPages(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string, totalPages int) {
	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	for pageNum := 0; pageNum < totalPages; pageNum++ {
		dqr := deviceQRs[pageNum]
		pdf.AddPage()

		// Title with device name
		pdf.SetFont("Arial", "B", 20)
		pdf.SetXY(margin, margin)
		pdf.CellFormat(180, 15, fmt.Sprintf("设备: %s", dqr.device.Name), "", 1, "C", false, 0, "")

		// Print date
		pdf.SetFont("Arial", "", 10)
		pdf.SetXY(margin, margin+18)
		pdf.CellFormat(180, 6, fmt.Sprintf("打印日期: %s", time.Now().Format("2006-01-02")), "", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 12)
		pdf.SetXY(margin, margin+26)
		pdf.CellFormat(180, 8, fmt.Sprintf("分组: %s", groupName), "", 1, "C", false, 0, "")

		pdf.SetXY(margin, margin+36)
		pdf.CellFormat(180, 8, fmt.Sprintf("设备 ID: %s", dqr.device.ID), "", 1, "C", false, 0, "")

		// Large QR code centered (use in-memory loading)
		qrSize := 100.0
		x := (210 - qrSize) / 2
		y := 70.0

		imgName := fmt.Sprintf("qr_%d", pageNum)
		r := bytes.NewReader(dqr.pngData)
		pdf.RegisterImageReader(imgName, "png", r)
		pdf.ImageOptions(imgName, x, y, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

		// Note about API key
		pdf.SetXY(margin, y+qrSize+10)
		pdf.SetFont("Arial", "I", 9)
		pdf.CellFormat(180, 5, "提示: 请使用 FreeKiosk 应用扫描此二维码以注册设备", "", 1, "C", false, 0, "")

		// Add footer with page number
		pdf.SetY(pageHeight - 15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 5, fmt.Sprintf("FreeKiosk 研学版  |  第 %d/%d 页", pageNum+1, totalPages), "", 1, "C", false, 0, "")
	}
}
