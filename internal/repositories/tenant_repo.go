package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// TenantRepository handles tenant database operations
type TenantRepository interface {
	InitTable(ctx context.Context) error
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

// JSONMap is a helper type for unmarshaling JSON from SQLite
type JSONMap map[string]interface{}

// Scan implements the sql.Scanner interface
func (j *JSONMap) Scan(src interface{}) error {
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("cannot scan type %T into JSONMap", src)
	}
	if s == "" || s == "{}" {
		*j = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(s), j)
}

// NullTime is a helper type for scanning SQLite datetime strings into time.Time
type NullTime struct {
	time.Time
}

// Scan implements the sql.Scanner interface
func (nt *NullTime) Scan(src interface{}) error {
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("cannot scan type %T into NullTime", src)
	}
	if s == "" {
		nt.Time = time.Time{}
		return nil
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	nt.Time = t
	return nil
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sqlx.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) InitTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		plan TEXT NOT NULL DEFAULT 'starter',
		status TEXT NOT NULL DEFAULT 'active',
		settings TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	query := `
		INSERT INTO tenants (id, name, slug, plan, status, settings)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = r.db.ExecContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Plan, tenant.Status, string(settingsJSON),
	)
	if err != nil {
		return err
	}
	// Fetch the created tenant to get timestamps
	created, err := r.GetByID(ctx, tenant.ID)
	if err != nil {
		return err
	}
	tenant.CreatedAt = created.CreatedAt
	tenant.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant struct {
		models.Tenant
		SettingsRaw string    `db:"settings"`
		CreatedAt   NullTime  `db:"created_at"`
		UpdatedAt   NullTime  `db:"updated_at"`
	}
	query := `SELECT id, name, slug, plan, status, settings, created_at, updated_at FROM tenants WHERE id = $1`
	err := r.db.GetContext(ctx, &tenant, query, id)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	if err := json.Unmarshal([]byte(tenant.SettingsRaw), &tenant.Settings); err != nil {
		tenant.Settings = make(map[string]interface{})
	}
	tenant.Tenant.CreatedAt = tenant.CreatedAt.Time
	tenant.Tenant.UpdatedAt = tenant.UpdatedAt.Time
	return &tenant.Tenant, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant struct {
		models.Tenant
		SettingsRaw string    `db:"settings"`
		CreatedAt   NullTime  `db:"created_at"`
		UpdatedAt   NullTime  `db:"updated_at"`
	}
	query := `SELECT id, name, slug, plan, status, settings, created_at, updated_at FROM tenants WHERE slug = $1`
	err := r.db.GetContext(ctx, &tenant, query, slug)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	if err := json.Unmarshal([]byte(tenant.SettingsRaw), &tenant.Settings); err != nil {
		tenant.Settings = make(map[string]interface{})
	}
	tenant.Tenant.CreatedAt = tenant.CreatedAt.Time
	tenant.Tenant.UpdatedAt = tenant.UpdatedAt.Time
	return &tenant.Tenant, nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	query := `
		UPDATE tenants
		SET name = $1, plan = $2, status = $3, settings = $4
		WHERE id = $5
	`
	_, err = r.db.ExecContext(ctx, query,
		tenant.Name, tenant.Plan, tenant.Status, string(settingsJSON), tenant.ID,
	)
	if err != nil {
		return err
	}
	// Fetch the updated tenant to get timestamps
	updated, err := r.GetByID(ctx, tenant.ID)
	if err != nil {
		return err
	}
	tenant.UpdatedAt = updated.UpdatedAt
	return nil
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
	var tenants []*struct {
		models.Tenant
		SettingsRaw string   `db:"settings"`
		CreatedAt   NullTime `db:"created_at"`
		UpdatedAt   NullTime `db:"updated_at"`
	}
	query := `SELECT id, name, slug, plan, status, settings, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*models.Tenant, len(tenants))
	for i, t := range tenants {
		if err := json.Unmarshal([]byte(t.SettingsRaw), &t.Settings); err != nil {
			t.Settings = make(map[string]interface{})
		}
		t.Tenant.CreatedAt = t.CreatedAt.Time
		t.Tenant.UpdatedAt = t.UpdatedAt.Time
		result[i] = &t.Tenant
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenants`
	err = r.db.GetContext(ctx, &total, countQuery)
	return result, total, err
}
