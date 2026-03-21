package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// SecurityPolicyRepository 安全策略仓储接口
type SecurityPolicyRepository interface {
	Create(ctx context.Context, policy *models.SecurityPolicy) error
	GetByID(ctx context.Context, id string) (*models.SecurityPolicy, error)
	GetByTenant(ctx context.Context, tenantID string) ([]*models.SecurityPolicy, error)
	Update(ctx context.Context, policy *models.SecurityPolicy) error
	Delete(ctx context.Context, id string) error
	AssignToDevice(ctx context.Context, policyID, deviceID string) error
	RemoveFromDevice(ctx context.Context, deviceID string) error
	GetByDevice(ctx context.Context, deviceID string) (*models.SecurityPolicy, error)
}

type securityPolicyRepository struct {
	db *sqlx.DB
}

// NewSecurityPolicyRepository 创建安全策略仓储
func NewSecurityPolicyRepository(db *sqlx.DB) SecurityPolicyRepository {
	return &securityPolicyRepository{db: db}
}

func (r *securityPolicyRepository) Create(ctx context.Context, policy *models.SecurityPolicy) error {
	settingsJSON, err := json.Marshal(policy.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	whitelistJSON, err := json.Marshal(policy.AppWhitelist)
	if err != nil {
		return fmt.Errorf("failed to marshal whitelist: %w", err)
	}

	query := `
		INSERT INTO security_policies (id, tenant_id, name, description, settings, app_whitelist, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err = r.db.QueryRowxContext(ctx, query,
		policy.ID,
		policy.TenantID,
		policy.Name,
		policy.Description,
		settingsJSON,
		whitelistJSON,
	).Scan(&policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create security policy: %w", err)
	}

	return nil
}

func (r *securityPolicyRepository) GetByID(ctx context.Context, id string) (*models.SecurityPolicy, error) {
	var policy models.SecurityPolicy
	var settingsJSON []byte
	var whitelistJSON []byte

	query := `
		SELECT id, tenant_id, name, description, settings, app_whitelist, created_at, updated_at
		FROM security_policies
		WHERE id = $1
	`

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&policy.ID,
		&policy.TenantID,
		&policy.Name,
		&policy.Description,
		&settingsJSON,
		&whitelistJSON,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("security policy not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get security policy: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &policy.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(whitelistJSON, &policy.AppWhitelist); err != nil {
		return nil, fmt.Errorf("failed to unmarshal whitelist: %w", err)
	}

	return &policy, nil
}

func (r *securityPolicyRepository) GetByTenant(ctx context.Context, tenantID string) ([]*models.SecurityPolicy, error) {
	var policies []*models.SecurityPolicy

	query := `
		SELECT id, tenant_id, name, description, settings, app_whitelist, created_at, updated_at
		FROM security_policies
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query security policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var policy models.SecurityPolicy
		var settingsJSON, whitelistJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.TenantID,
			&policy.Name,
			&policy.Description,
			&settingsJSON,
			&whitelistJSON,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan security policy: %w", err)
		}

		if err := json.Unmarshal(settingsJSON, &policy.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}

		if err := json.Unmarshal(whitelistJSON, &policy.AppWhitelist); err != nil {
			return nil, fmt.Errorf("failed to unmarshal whitelist: %w", err)
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}

func (r *securityPolicyRepository) Update(ctx context.Context, policy *models.SecurityPolicy) error {
	settingsJSON, err := json.Marshal(policy.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	whitelistJSON, err := json.Marshal(policy.AppWhitelist)
	if err != nil {
		return fmt.Errorf("failed to marshal whitelist: %w", err)
	}

	query := `
		UPDATE security_policies
		SET name = $1, description = $2, settings = $3, app_whitelist = $4, updated_at = NOW()
		WHERE id = $5
	`

	result, err := r.db.ExecContext(ctx, query,
		policy.Name,
		policy.Description,
		settingsJSON,
		whitelistJSON,
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update security policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("security policy not found: %s", policy.ID)
	}

	return nil
}

func (r *securityPolicyRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM security_policies WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete security policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("security policy not found: %s", id)
	}

	return nil
}

func (r *securityPolicyRepository) AssignToDevice(ctx context.Context, policyID, deviceID string) error {
	query := `UPDATE devices SET security_policy_id = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, policyID, deviceID)
	if err != nil {
		return fmt.Errorf("failed to assign policy to device: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	return nil
}

func (r *securityPolicyRepository) RemoveFromDevice(ctx context.Context, deviceID string) error {
	query := `UPDATE devices SET security_policy_id = NULL, updated_at = NOW() WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to remove policy from device: %w", err)
	}

	return nil
}

func (r *securityPolicyRepository) GetByDevice(ctx context.Context, deviceID string) (*models.SecurityPolicy, error) {
	var policyID sql.NullString

	query := `SELECT security_policy_id FROM devices WHERE id = $1`
	err := r.db.QueryRowxContext(ctx, query, deviceID).Scan(&policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	if !policyID.Valid || policyID.String == "" {
		// 返回默认策略
		return r.GetByID(ctx, "00000000-0000-0000-0000-000000000001")
	}

	return r.GetByID(ctx, policyID.String)
}

// AppWhitelistRepository 应用白名单仓储接口
type AppWhitelistRepository interface {
	GetByDevice(ctx context.Context, deviceID string) ([]models.AppWhitelistEntry, error)
	GetByPolicy(ctx context.Context, policyID string) ([]models.AppWhitelistEntry, error)
	AddApp(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error
	RemoveApp(ctx context.Context, policyID, packageName string) error
	UpdateApp(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error
}

type appWhitelistRepository struct {
	policyRepo SecurityPolicyRepository
}

func NewAppWhitelistRepository(db *sqlx.DB) AppWhitelistRepository {
	return &appWhitelistRepository{
		policyRepo: NewSecurityPolicyRepository(db),
	}
}

func (r *appWhitelistRepository) GetByDevice(ctx context.Context, deviceID string) ([]models.AppWhitelistEntry, error) {
	policy, err := r.policyRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return policy.AppWhitelist, nil
}

func (r *appWhitelistRepository) GetByPolicy(ctx context.Context, policyID string) ([]models.AppWhitelistEntry, error) {
	policy, err := r.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return nil, err
	}
	return policy.AppWhitelist, nil
}

func (r *appWhitelistRepository) AddApp(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error {
	policy, err := r.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return err
	}

	// 检查是否已存在
	for _, existing := range policy.AppWhitelist {
		if existing.PackageName == entry.PackageName {
			return fmt.Errorf("app already in whitelist: %s", entry.PackageName)
		}
	}

	policy.AppWhitelist = append(policy.AppWhitelist, entry)
	return r.policyRepo.Update(ctx, policy)
}

func (r *appWhitelistRepository) RemoveApp(ctx context.Context, policyID, packageName string) error {
	policy, err := r.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return err
	}

	for i, entry := range policy.AppWhitelist {
		if entry.PackageName == packageName {
			policy.AppWhitelist = append(policy.AppWhitelist[:i], policy.AppWhitelist[i+1:]...)
			return r.policyRepo.Update(ctx, policy)
		}
	}

	return fmt.Errorf("app not found in whitelist: %s", packageName)
}

func (r *appWhitelistRepository) UpdateApp(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error {
	policy, err := r.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return err
	}

	for i, existing := range policy.AppWhitelist {
		if existing.PackageName == entry.PackageName {
			policy.AppWhitelist[i] = entry
			return r.policyRepo.Update(ctx, policy)
		}
	}

	return fmt.Errorf("app not found in whitelist: %s", entry.PackageName)
}
