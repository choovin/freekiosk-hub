package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// ExportHandler handles QR code PDF export
type ExportHandler struct {
	Repo *repositories.FieldTripRepository
}

// NewExportHandler creates a new ExportHandler
func NewExportHandler(repo *repositories.FieldTripRepository) *ExportHandler {
	return &ExportHandler{Repo: repo}
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
	pngPath  string
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

	// Create temp directory for QR images
	tempDir, err := os.MkdirTemp("", "freekiosk-qr-*")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create temp directory")
	}
	defer os.RemoveAll(tempDir)

	// Generate QR payload for each device
	deviceQRs := make([]deviceQR, 0, len(devices))
	for i, device := range devices {
		// Create QR payload - note: API key is not stored in plaintext
		// The QR code format follows BindRequest structure
		// For full functionality, the system would need to store encrypted plaintext keys
		payload := QRPayload{
			DeviceID: device.ID,
			GroupKey: group.GroupKey,
			APIKey:   "[RECREATE_KEY]", // Placeholder - plaintext key not stored
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

		// Write to temp file
		pngPath := filepath.Join(tempDir, fmt.Sprintf("qr_%d.png", i))
		if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
			continue
		}

		deviceQRs = append(deviceQRs, deviceQR{
			device:   &devices[i], // Take address of array element to avoid range loop issues
			groupKey: group.GroupKey,
			payload:  payloadStr,
			pngPath:  pngPath,
		})
	}

	if len(deviceQRs) == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate QR codes")
	}

	// Generate PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// Add pages based on layout
	switch layout {
	case "6up":
		h.add6UpPage(pdf, deviceQRs, group.Name)
	case "4up":
		h.add4UpPage(pdf, deviceQRs, group.Name)
	case "1up":
		h.add1UpPage(pdf, deviceQRs, group.Name)
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

// add6UpPage adds a page with 6 QR codes (2 rows x 3 columns)
func (h *ExportHandler) add6UpPage(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string) {
	pdf.AddPage()

	// A4 page: 210mm x 297mm
	// Margins: 15mm each side
	// Usable width: 180mm
	// 3 columns: 60mm each
	// Usable height: 267mm (297 - 15*2)
	// 2 rows: ~133mm each

	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	cellWidth := 60.0
	cellHeight := 130.0
	qrSize := 40.0
	colCount := 3

	// Title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetXY(margin, margin)
	pdf.CellFormat(180, 10, fmt.Sprintf("Group: %s (6-up layout)", groupName), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	startY := pdf.GetY()

	for i, dqr := range deviceQRs {
		if i >= 6 {
			break
		}

		col := i % colCount
		row := i / colCount

		x := margin + float64(col)*cellWidth
		y := startY + float64(row)*cellHeight

		// Add QR code centered in cell
		qrX := x + (cellWidth-qrSize)/2
		pdf.ImageOptions(dqr.pngPath, qrX, y+5, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

		// Add device name below QR
		pdf.SetXY(x, y+qrSize+8)
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(cellWidth, 5, dqr.device.Name, "", 1, "C", false, 0, "")
	}

	// Add footer
	pdf.SetY(pageHeight - 15)
	pdf.SetFont("Arial", "I", 7)
	pdf.CellFormat(0, 5, "FreeKiosk Field Trip - QR Code Export", "", 1, "C", false, 0, "")
}

// add4UpPage adds a page with 4 QR codes (2 rows x 2 columns)
func (h *ExportHandler) add4UpPage(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string) {
	pdf.AddPage()

	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	cellWidth := 90.0
	cellHeight := 130.0
	qrSize := 60.0
	colCount := 2

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(margin, margin)
	pdf.CellFormat(180, 10, fmt.Sprintf("Group: %s (4-up layout)", groupName), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	startY := pdf.GetY()

	for i, dqr := range deviceQRs {
		if i >= 4 {
			break
		}

		col := i % colCount
		row := i / colCount

		x := margin + float64(col)*cellWidth
		y := startY + float64(row)*cellHeight

		// Add device name above QR
		pdf.SetXY(x, y)
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(cellWidth, 6, dqr.device.Name, "", 1, "C", false, 0, "")

		// Add QR code centered in cell
		qrX := x + (cellWidth-qrSize)/2
		pdf.ImageOptions(dqr.pngPath, qrX, y+8, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	// Add footer
	pdf.SetY(pageHeight - 15)
	pdf.SetFont("Arial", "I", 7)
	pdf.CellFormat(0, 5, "FreeKiosk Field Trip - QR Code Export", "", 1, "C", false, 0, "")
}

// add1UpPage adds a page with 1 large QR code per page
func (h *ExportHandler) add1UpPage(pdf *gofpdf.Fpdf, deviceQRs []deviceQR, groupName string) {
	margin := 15.0
	pageWidth, pageHeight := pdf.GetPageSize()
	_ = pageWidth

	for _, dqr := range deviceQRs {
		pdf.AddPage()

		// Title with device name
		pdf.SetFont("Arial", "B", 20)
		pdf.SetXY(margin, margin)
		pdf.CellFormat(180, 15, fmt.Sprintf("Device: %s", dqr.device.Name), "", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 12)
		pdf.SetXY(margin, margin+20)
		pdf.CellFormat(180, 8, fmt.Sprintf("Group: %s", groupName), "", 1, "C", false, 0, "")

		pdf.SetXY(margin, margin+30)
		pdf.CellFormat(180, 8, fmt.Sprintf("Device ID: %s", dqr.device.ID), "", 1, "C", false, 0, "")

		// Large QR code centered
		qrSize := 100.0
		x := (210 - qrSize) / 2
		y := 70.0

		pdf.ImageOptions(dqr.pngPath, x, y, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

		// Note about API key
		pdf.SetXY(margin, y+qrSize+10)
		pdf.SetFont("Arial", "I", 9)
		pdf.CellFormat(180, 5, "Note: Scan this QR code with FreeKiosk app to register device.", "", 1, "C", false, 0, "")

		// Add footer
		pdf.SetY(pageHeight - 15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 5, "FreeKiosk Field Trip - QR Code Export", "", 1, "C", false, 0, "")
	}
}
