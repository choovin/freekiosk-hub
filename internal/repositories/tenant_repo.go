package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// TenantRepository handles tenant database operations
type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*models.Tenant, int64, error)
}

type tenantRepository struct {
	db *sqlx.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sqlx.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, slug, plan, status, settings)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Plan, tenant.Status, tenant.Settings,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
}

func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	query := `SELECT * FROM tenants WHERE id = $1`
	err := r.db.GetContext(ctx, &tenant, query, id)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	query := `SELECT * FROM tenants WHERE slug = $1`
	err := r.db.GetContext(ctx, &tenant, query, slug)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $1, plan = $2, status = $3, settings = $4
		WHERE id = $5
		RETURNING updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		tenant.Name, tenant.Plan, tenant.Status, tenant.Settings, tenant.ID,
	).Scan(&tenant.UpdatedAt)
}

func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tenants WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *tenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, int64, error) {
	var tenants []*models.Tenant
	query := `SELECT * FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenants`
	err = r.db.GetContext(ctx, &total, countQuery)
	return tenants, total, err
}
