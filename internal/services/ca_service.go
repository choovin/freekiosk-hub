package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"log/slog"
)

// CAService handles certificate authority operations
type CAService interface {
	ValidateCSR(csrPem string) (*x509.CertificateRequest, error)
	SignCertificate(ctx context.Context, csr *x509.CertificateRequest, deviceID string) (*models.DeviceCertificate, error)
	RevokeCertificate(ctx context.Context, serialNumber string, reason string) error
	IsCertificateValid(ctx context.Context, serialNumber string) (bool, error)
}

type caService struct {
	caCert         *x509.Certificate
	caKey          interface{}
	certRepo       repositories.CertificateRepository
	validityDays   int
}

// NewCAService creates a new CA service
func NewCAService(caCertPath, caKeyPath string, validityDays int, certRepo repositories.CertificateRepository) (CAService, error) {
	caService := &caService{
		caCert:       nil,
		caKey:        nil,
		certRepo:     certRepo,
		validityDays: validityDays,
	}

	// Try to load CA certificate and key from files
	if caKeyPath != "" {
		if err := caService.loadCAFromFiles(caCertPath, caKeyPath); err != nil {
			slog.Warn("⚠️ Failed to load CA from files, will generate new CA", "error", err)
			// Generate a new CA for development
			if err := caService.generateDevCA(); err != nil {
				return nil, fmt.Errorf("failed to generate development CA: %w", err)
			}
		} else {
			slog.Info("✅ CA loaded from files", "cert", caCertPath, "key", caKeyPath)
		}
	} else {
		// No CA key path provided, generate for development
		slog.Info("ℹ️ No CA key path configured, generating development CA")
		if err := caService.generateDevCA(); err != nil {
			return nil, fmt.Errorf("failed to generate development CA: %w", err)
		}
	}

	return caService, nil
}

// loadCAFromFiles loads CA certificate and key from PEM files
func (s *caService) loadCAFromFiles(certPath, keyPath string) error {
	// Load CA certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	s.caCert = caCert

	// Load CA private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	// Parse the private key based on type
	var key interface{}
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse PKCS1 private key: %w", err)
		}
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse EC private key: %w", err)
		}
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
	default:
		return fmt.Errorf("unsupported private key type: %s", keyBlock.Type)
	}

	s.caKey = key
	return nil
}

// generateDevCA generates a self-signed CA for development
func (s *caService) generateDevCA() error {
	slog.Info("🔐 Generating development CA...")

	// Generate RSA key for CA
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "FreeKiosk Enterprise CA",
			Organization: []string{"FreeKiosk Enterprise"},
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	// Self-sign the CA certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	s.caCert = caCert
	s.caKey = privateKey

	// Encode CA cert to PEM for display
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	slog.Info("✅ Development CA generated", "cert_serial", template.SerialNumber)
	slog.Debug("CA Certificate (first 100 chars)", "pem", string(certPEM[:min(100, len(certPEM))]))

	return nil
}

// ValidateCSR parses and validates a CSR
func (s *caService) ValidateCSR(csrPem string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(csrPem))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "CERTIFICATE REQUEST" {
		return nil, fmt.Errorf("invalid PEM type: %s", block.Type)
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSR: %w", err)
	}

	// Verify CSR signature
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature verification failed: %w", err)
	}

	return csr, nil
}

// SignCertificate signs a CSR and returns a device certificate
func (s *caService) SignCertificate(ctx context.Context, csr *x509.CertificateRequest, deviceID string) (*models.DeviceCertificate, error) {
	if s.caKey == nil || s.caCert == nil {
		return nil, fmt.Errorf("CA not initialized - call NewCAService first")
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   deviceID,
			Organization: []string{"FreeKiosk Enterprise"},
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(0, 0, s.validityDays),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the certificate using the CA
	derBytes, err := x509.CreateCertificate(rand.Reader, template, s.caCert, csr.PublicKey, s.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode to PEM
	certPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}))

	// Store in database
	cert := &models.DeviceCertificate{
		ID:             uuid.New().String(),
		DeviceID:       deviceID,
		CertificatePem: certPem,
		SerialNumber:   serialNumber.String(),
		IssuedAt:       now,
		ExpiresAt:      template.NotAfter,
	}

	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	slog.Info("✅ Certificate signed", "device_id", deviceID, "serial", serialNumber.String())
	return cert, nil
}

// RevokeCertificate revokes a certificate
func (s *caService) RevokeCertificate(ctx context.Context, serialNumber string, reason string) error {
	cert, err := s.certRepo.GetBySerialNumber(ctx, serialNumber)
	if err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	return s.certRepo.Revoke(ctx, cert.ID, reason)
}

// IsCertificateValid checks if a certificate is valid (not revoked or expired)
func (s *caService) IsCertificateValid(ctx context.Context, serialNumber string) (bool, error) {
	cert, err := s.certRepo.GetBySerialNumber(ctx, serialNumber)
	if err != nil {
		return false, err
	}

	// Check if revoked
	if cert.RevokedAt != nil {
		return false, nil
	}

	// Check if expired
	if cert.ExpiresAt.Before(time.Now()) {
		return false, nil
	}

	return true, nil
}
