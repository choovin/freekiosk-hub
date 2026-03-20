package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// CertificateRepository handles certificate database operations
type CertificateRepository interface {
	Create(ctx context.Context, cert *models.DeviceCertificate) error
	GetByID(ctx context.Context, id string) (*models.DeviceCertificate, error)
	GetByDeviceID(ctx context.Context, deviceID string) (*models.DeviceCertificate, error)
	GetBySerialNumber(ctx context.Context, serialNumber string) (*models.DeviceCertificate, error)
	Revoke(ctx context.Context, id string, reason string) error
	Delete(ctx context.Context, id string) error
	DeleteByDeviceID(ctx context.Context, deviceID string) error
	GetExpiringSoon(ctx context.Context, within time.Duration) ([]*models.DeviceCertificate, error)
}

type certificateRepository struct {
	db *sqlx.DB
}

// NewCertificateRepository creates a new certificate repository
func NewCertificateRepository(db *sqlx.DB) CertificateRepository {
	return &certificateRepository{db: db}
}

func (r *certificateRepository) Create(ctx context.Context, cert *models.DeviceCertificate) error {
	query := `
		INSERT INTO device_certificates (id, device_id, certificate_pem, serial_number, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return r.db.QueryRowxContext(ctx, query,
		cert.ID, cert.DeviceID, cert.CertificatePem, cert.SerialNumber, cert.IssuedAt, cert.ExpiresAt,
	).Scan(&cert.ID)
}

func (r *certificateRepository) GetByID(ctx context.Context, id string) (*models.DeviceCertificate, error) {
	var cert models.DeviceCertificate
	query := `SELECT * FROM device_certificates WHERE id = $1`
	err := r.db.GetContext(ctx, &cert, query, id)
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	return &cert, nil
}

func (r *certificateRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.DeviceCertificate, error) {
	var cert models.DeviceCertificate
	query := `SELECT * FROM device_certificates WHERE device_id = $1 AND revoked_at IS NULL ORDER BY issued_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &cert, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	return &cert, nil
}

func (r *certificateRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (*models.DeviceCertificate, error) {
	var cert models.DeviceCertificate
	query := `SELECT * FROM device_certificates WHERE serial_number = $1`
	err := r.db.GetContext(ctx, &cert, query, serialNumber)
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	return &cert, nil
}

func (r *certificateRepository) Revoke(ctx context.Context, id string, reason string) error {
	query := `UPDATE device_certificates SET revoked_at = $1, revoked_reason = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, time.Now(), reason, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *certificateRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM device_certificates WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *certificateRepository) DeleteByDeviceID(ctx context.Context, deviceID string) error {
	query := `DELETE FROM device_certificates WHERE device_id = $1`
	_, err := r.db.ExecContext(ctx, query, deviceID)
	return err
}

func (r *certificateRepository) GetExpiringSoon(ctx context.Context, within time.Duration) ([]*models.DeviceCertificate, error) {
	var certs []*models.DeviceCertificate
	query := `
		SELECT * FROM device_certificates
		WHERE expires_at <= $1 AND revoked_at IS NULL
		ORDER BY expires_at ASC
	`
	err := r.db.SelectContext(ctx, &certs, query, time.Now().Add(within))
	return certs, err
}

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	GetByDeviceID(ctx context.Context, deviceID string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForDevice(ctx context.Context, deviceID string) error
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type refreshTokenRepository struct {
	db *sqlx.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *sqlx.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, device_id, token_hash, issued_at, expires_at, previous_token_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return r.db.QueryRowxContext(ctx, query,
		token.ID, token.DeviceID, token.TokenHash, token.IssuedAt, token.ExpiresAt, token.PreviousTokenHash,
	).Scan(&token.ID)
}

func (r *refreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `SELECT * FROM refresh_tokens WHERE token_hash = $1 AND revoked_at IS NULL`
	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	return &token, nil
}

func (r *refreshTokenRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `SELECT * FROM refresh_tokens WHERE device_id = $1 AND revoked_at IS NULL ORDER BY issued_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &token, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllForDevice(ctx context.Context, deviceID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE device_id = $2 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), deviceID)
	return err
}

func (r *refreshTokenRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1 OR revoked_at IS NOT NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
