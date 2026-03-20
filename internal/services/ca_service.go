package services

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
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
	// In production, load CA cert and key from files
	// For development, we'll generate them
	// This is a placeholder - actual implementation would load from files

	return &caService{
		caCert:       nil, // Would be loaded from caCertPath
		caKey:        nil, // Would be loaded from caKeyPath
		certRepo:     certRepo,
		validityDays: validityDays,
	}, nil
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

	// Sign the certificate
	// In production, we'd use s.caCert and s.caKey
	// For development, we'll create a self-signed cert
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, csr.PublicKey, s.caKey)
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
