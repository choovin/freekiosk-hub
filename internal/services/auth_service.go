package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// AuthService handles device authentication and registration
type AuthService interface {
	RegisterDevice(ctx context.Context, req *models.DeviceRegistration) (*models.DeviceCredentials, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error)
	ValidateDevice(ctx context.Context, deviceID string) (bool, error)
	RevokeDevice(ctx context.Context, deviceID string) error
}

type authService struct {
	deviceRepo    repositories.DeviceRepository
	tenantRepo    repositories.TenantRepository
	certRepo      repositories.CertificateRepository
	refreshRepo   repositories.RefreshTokenRepository
	caService     CAService
	jwtService    JWTService
	integritySvc  IntegrityService
}

// NewAuthService creates a new authentication service
func NewAuthService(
	deviceRepo repositories.DeviceRepository,
	tenantRepo repositories.TenantRepository,
	certRepo repositories.CertificateRepository,
	refreshRepo repositories.RefreshTokenRepository,
	caService CAService,
	jwtService JWTService,
	integritySvc IntegrityService,
) AuthService {
	return &authService{
		deviceRepo:    deviceRepo,
		tenantRepo:    tenantRepo,
		certRepo:      certRepo,
		refreshRepo:   refreshRepo,
		caService:     caService,
		jwtService:    jwtService,
		integritySvc:  integritySvc,
	}
}

// RegisterDevice handles device registration flow
func (s *authService) RegisterDevice(ctx context.Context, req *models.DeviceRegistration) (*models.DeviceCredentials, error) {
	// 1. Validate tenant exists
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	if tenant.Status != string(models.TenantStatusActive) {
		return nil, fmt.Errorf("tenant is not active")
	}

	// 2. Check if device already exists
	existingDevice, err := s.deviceRepo.GetByTenantAndKey(ctx, req.TenantID, req.DeviceKey)
	if err == nil && existingDevice != nil {
		// Device already registered, check status
		if existingDevice.Status == string(models.DeviceStatusActive) {
			return nil, fmt.Errorf("device already registered and active")
		}
		// Reactivate if previously disabled
	}

	// 3. Validate CSR
	csr, err := s.caService.ValidateCSR(req.CSRPem)
	if err != nil {
		return nil, fmt.Errorf("invalid CSR: %w", err)
	}

	// 4. Verify Play Integrity token (if provided)
	if req.IntegrityToken != "" {
		verdict, err := s.integritySvc.VerifyToken(ctx, req.IntegrityToken, req.Nonce)
		if err != nil {
			// Log warning but don't fail registration
			// In production, you might want to enforce this
			fmt.Printf("Integrity verification failed: %v\n", err)
		} else {
			// Check integrity verdict
			valid, _ := CheckDeviceIntegrity(verdict)
			if !valid {
				// Log for review
				fmt.Printf("Device failed integrity check: %s\n", req.DeviceKey)
			}
		}
	}

	// 5. Create or update device
	deviceID := uuid.New().String()
	device := &models.Device{
		ID:         deviceID,
		TenantID:   req.TenantID,
		DeviceKey:  req.DeviceKey,
		Name:       req.DeviceInfo.Model,
		Status:     string(models.DeviceStatusActive),
		DeviceInfo: req.DeviceInfo.ToMap(),
	}

	if existingDevice != nil {
		device.ID = existingDevice.ID
		device.CreatedAt = existingDevice.CreatedAt
		if err := s.deviceRepo.Update(ctx, device); err != nil {
			return nil, fmt.Errorf("failed to update device: %w", err)
		}
	} else {
		if err := s.deviceRepo.Create(ctx, device); err != nil {
			return nil, fmt.Errorf("failed to create device: %w", err)
		}
	}

	// 6. Sign certificate
	cert, err := s.caService.SignCertificate(ctx, csr, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	// 7. Generate tokens
	accessToken, expiresAt, err := s.jwtService.GenerateAccessToken(deviceID, req.TenantID, req.DeviceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	tokenHash := s.jwtService.HashToken(refreshToken)
	refreshTokenRecord := &models.RefreshToken{
		ID:        uuid.New().String(),
		DeviceID:  deviceID,
		TokenHash: tokenHash,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}
	if err := s.refreshRepo.Create(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// 8. Return credentials
	return &models.DeviceCredentials{
		DeviceID:       deviceID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresAt:      expiresAt,
		CertificatePem: cert.CertificatePem,
		SerialNumber:   cert.SerialNumber,
		CertExpiresAt:  cert.ExpiresAt,
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	return s.jwtService.RefreshTokens(ctx, refreshToken)
}

// ValidateDevice checks if a device is valid and active
func (s *authService) ValidateDevice(ctx context.Context, deviceID string) (bool, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return false, err
	}

	if device.Status != string(models.DeviceStatusActive) {
		return false, nil
	}

	// Update last seen
	if err := s.deviceRepo.UpdateLastSeen(ctx, deviceID, time.Now()); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to update last seen: %v\n", err)
	}

	return true, nil
}

// RevokeDevice revokes all tokens and certificates for a device
func (s *authService) RevokeDevice(ctx context.Context, deviceID string) error {
	// Revoke all refresh tokens
	if err := s.refreshRepo.RevokeAllForDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}

	// Revoke certificates
	cert, err := s.certRepo.GetByDeviceID(ctx, deviceID)
	if err == nil {
		if err := s.certRepo.Revoke(ctx, cert.ID, "device_revoked"); err != nil {
			return fmt.Errorf("failed to revoke certificate: %w", err)
		}
	}

	// Update device status
	if err := s.deviceRepo.UpdateStatus(ctx, deviceID, string(models.DeviceStatusDisabled)); err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	return nil
}
