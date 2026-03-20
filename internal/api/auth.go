package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/services"
)

// AuthHandler handles authentication API endpoints
type AuthHandler struct {
	authSvc services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authSvc services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// HandleRegister handles device registration
// POST /api/v2/auth/register
func (h *AuthHandler) HandleRegister(c echo.Context) error {
	var req models.DeviceRegistration
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.TenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}
	if req.DeviceKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_key is required",
		})
	}
	if req.CSRPem == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "csr_pem is required",
		})
	}

	// Process registration
	credentials, err := h.authSvc.RegisterDevice(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"device_id":       credentials.DeviceID,
		"access_token":    credentials.AccessToken,
		"refresh_token":   credentials.RefreshToken,
		"expires_at":      credentials.ExpiresAt.UnixMilli(),
		"certificate_pem": credentials.CertificatePem,
		"serial_number":   credentials.SerialNumber,
		"cert_expires_at": credentials.CertExpiresAt.UnixMilli(),
	})
}

// HandleRefreshToken handles token refresh
// POST /api/v2/auth/refresh
func (h *AuthHandler) HandleRefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "refresh_token is required",
		})
	}

	// Refresh tokens
	tokenPair, err := h.authSvc.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid or expired refresh token",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt.UnixMilli(),
		"token_type":    tokenPair.TokenType,
	})
}

// HandleValidateDevice handles device validation
// GET /api/v2/auth/validate/:deviceId
func (h *AuthHandler) HandleValidateDevice(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id is required",
		})
	}

	valid, err := h.authSvc.ValidateDevice(c.Request().Context(), deviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Device not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"device_id": deviceID,
		"valid":     valid,
	})
}

// HandleRevokeDevice handles device revocation
// DELETE /api/v2/auth/device/:deviceId
func (h *AuthHandler) HandleRevokeDevice(c echo.Context) error {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id is required",
		})
	}

	if err := h.authSvc.RevokeDevice(c.Request().Context(), deviceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Device revoked successfully",
	})
}

// HandleGetToken handles getting a new token (for testing)
// POST /api/v2/auth/token
func (h *AuthHandler) HandleGetToken(c echo.Context) error {
	var req struct {
		TenantID  string `json:"tenant_id"`
		DeviceKey string `json:"device_key"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// This is a simplified endpoint for testing
	// In production, this would require proper authentication

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  uuid.New().String(),
		"refresh_token": uuid.New().String(),
		"expires_at":    time.Now().Add(time.Hour).UnixMilli(),
		"token_type":    "Bearer",
	})
}
