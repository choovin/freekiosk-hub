package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IntegrityService handles Play Integrity API verification
type IntegrityService interface {
	VerifyToken(ctx context.Context, token string, requestHash string) (*IntegrityVerdict, error)
}

// IntegrityVerdict represents the decoded Play Integrity verdict
type IntegrityVerdict struct {
	RequestPackageName          string   `json:"requestPackageName"`
	Nonce                      string   `json:"nonce"`
	TimestampMillis            int64    `json:"timestampMillis"`
	AppIntegrity               AppIntegrity `json:"appIntegrity"`
	DeviceIntegrity            DeviceIntegrity `json:"deviceIntegrity"`
	AccountIntegrity           []string `json:"accountIntegrity"`
}

// AppIntegrity represents app recognition verdict
type AppIntegrity struct {
	AppRecognitionVerdict string `json:"appRecognitionVerdict"`
	PackageName            string `json:"packageName"`
	CertificateSha256Hash  string `json:"certificateSha256Hash"`
}

// DeviceIntegrity represents device recognition verdict
type DeviceIntegrity struct {
	DeviceRecognitionVerdict []string `json:"deviceRecognitionVerdict"`
}

type integrityService struct {
	googlePlayConsoleAPIKey string
	appPackageName          string
	httpClient              *http.Client
}

// NewIntegrityService creates a new Play Integrity service
func NewIntegrityService(apiKey, appPackageName string) IntegrityService {
	return &integrityService{
		googlePlayConsoleAPIKey: apiKey,
		appPackageName:          appPackageName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerifyToken verifies a Play Integrity token with Google's servers
func (s *integrityService) VerifyToken(ctx context.Context, token string, requestHash string) (*IntegrityVerdict, error) {
	// In production, this would call Google Play Integrity API
	// https://developer.android.com/google/play/integrity/verdict
	//
	// For now, we'll decode the JWT token locally
	// In production, you MUST verify with Google's servers using your decryption key

	// Call Google Play Integrity API to decrypt and verify
	url := fmt.Sprintf("https://www.googleapis.com/androidpublisher/v3/applications/%s/integrity/verify", s.appPackageName)

	reqBody := map[string]string{
		"integrity_token": token,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.googlePlayConsoleAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var verdict IntegrityVerdict
	if err := json.NewDecoder(resp.Body).Decode(&verdict); err != nil {
		return nil, fmt.Errorf("failed to decode verdict: %w", err)
	}

	return &verdict, nil
}

// CheckDeviceIntegrity checks if device meets integrity requirements
func CheckDeviceIntegrity(verdict *IntegrityVerdict) (bool, string) {
	// Check device recognition verdict
	for _, v := range verdict.DeviceIntegrity.DeviceRecognitionVerdict {
		switch v {
		case "MEETS_STRONG_INTEGRITY":
			return true, "MEETS_STRONG_INTEGRITY"
		case "MEETS_DEVICE_INTEGRITY":
			return true, "MEETS_DEVICE_INTEGRITY"
		case "MEETS_BASIC_INTEGRITY":
			return true, "MEETS_BASIC_INTEGRITY"
		}
	}

	// Check app integrity
	if verdict.AppIntegrity.AppRecognitionVerdict == "PLAY_RECOGNIZED" {
		return true, "APP_PLAY_RECOGNIZED"
	}
	if verdict.AppIntegrity.AppRecognitionVerdict == "UNRECOGNIZED_VERSION" {
		// App is not the official version - could be a concern
		return false, "APP_UNRECOGNIZED_VERSION"
	}

	return false, "NO_INTEGRITY"
}
