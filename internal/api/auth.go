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

// HandleRegister 注册设备
// @Summary 注册设备
// @Description 设备注册，获取访问令牌和证书
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.DeviceRegistration true "设备注册信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/auth/register [post]
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

// HandleRefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v2/auth/refresh [post]
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

// HandleValidateDevice 验证设备
// @Summary 验证设备
// @Description 验证设备是否已注册且有效
// @Tags 认证
// @Produce json
// @Param deviceId path string true "设备ID"
// @Success 200 {object} ValidateDeviceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v2/auth/validate/{deviceId} [get]
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

// HandleRevokeDevice 吊销设备
// @Summary 吊销设备
// @Description 吊销指定设备的访问权限
// @Tags 认证
// @Produce json
// @Param deviceId path string true "设备ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v2/auth/device/{deviceId} [delete]
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

// HandleGetToken 获取令牌
// @Summary 获取访问令牌
// @Description 根据租户ID和设备密钥获取访问令牌(仅用于测试)
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body TokenRequest true "令牌请求"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v2/auth/token [post]
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

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"xxx"`
}

// TokenRequest 获取令牌请求
type TokenRequest struct {
	TenantID  string `json:"tenant_id" example:"tenant001"`
	DeviceKey string `json:"device_key" example:"device-key-xxx"`
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token,omitempty" example:"xxx"`
	ExpiresAt   int64  `json:"expires_at" example:"1700000000000"`
	TokenType   string `json:"token_type" example:"Bearer"`
}

// ValidateDeviceResponse 设备验证响应
type ValidateDeviceResponse struct {
	DeviceID string `json:"device_id" example:"device001"`
	Valid    bool   `json:"valid" example:"true"`
}
