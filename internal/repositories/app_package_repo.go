package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// AppPackageRepository 应用包仓库接口
type AppPackageRepository interface {
	InitSchema(ctx interface{}) error
	Create(pkg *models.AppPackage) error
	GetByID(id string) (*models.AppPackage, error)
	GetByPackageName(tenantID, packageName string) (*models.AppPackage, error)
	Update(pkg *models.AppPackage) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.AppPackage, int64, error)
	Search(tenantID, keyword string, limit, offset int) ([]*models.AppPackage, int64, error)
	IncrementInstallCount(id string) error
}

// SQLiteAppPackageRepository SQLite实现
type SQLiteAppPackageRepository struct {
	db *sqlx.DB
}

// NewSQLiteAppPackageRepository 创建应用包仓库
func NewSQLiteAppPackageRepository(db interface{}) *SQLiteAppPackageRepository {
	var sqlxDB *sqlx.DB
	switch v := db.(type) {
	case *sqlx.DB:
		sqlxDB = v
	case *sql.DB:
		sqlxDB = sqlx.NewDb(v, "sqlite")
	default:
		panic(fmt.Sprintf("unsupported db type: %T", db))
	}
	return &SQLiteAppPackageRepository{db: sqlxDB}
}

// InitSchema 初始化表结构
func (r *SQLiteAppPackageRepository) InitSchema(ctx interface{}) error {
	schema := `
		CREATE TABLE IF NOT EXISTS app_packages (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			package_name TEXT NOT NULL,
			version TEXT NOT NULL,
			version_code INTEGER NOT NULL,
			file_size INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			sha256 TEXT NOT NULL,
			description TEXT,
			tenant_id TEXT NOT NULL,
			uploaded_by TEXT,
			upload_url TEXT NOT NULL,
			install_count INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(package_name, version)
		);

		CREATE INDEX IF NOT EXISTS idx_app_packages_tenant ON app_packages(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_app_packages_package ON app_packages(package_name);

		CREATE TABLE IF NOT EXISTS app_installs (
			id TEXT PRIMARY KEY,
			package_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			error_msg TEXT,
			installed_at INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY (package_id) REFERENCES app_packages(id) ON DELETE CASCADE,
			FOREIGN KEY (device_id) REFERENCES mdm_devices(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_app_installs_package ON app_installs(package_id);
		CREATE INDEX IF NOT EXISTS idx_app_installs_device ON app_installs(device_id);
		CREATE INDEX IF NOT EXISTS idx_app_installs_tenant ON app_installs(tenant_id);
	`
	_, err := r.db.Exec(schema)
	return err
}

// Create 创应用包记录
func (r *SQLiteAppPackageRepository) Create(pkg *models.AppPackage) error {
	query := `
		INSERT INTO app_packages (
			id, name, package_name, version, version_code, file_size, file_path,
			sha256, description, tenant_id, uploaded_by, upload_url, install_count,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := r.db.Exec(query,
		pkg.ID, pkg.Name, pkg.PackageName, pkg.Version, pkg.VersionCode,
		pkg.FileSize, pkg.FilePath, pkg.SHA256, pkg.Description, pkg.TenantID,
		pkg.UploadedBy, pkg.UploadURL, pkg.InstallCount, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create app package: %w", err)
	}
	pkg.CreatedAt = now
	pkg.UpdatedAt = now
	return nil
}

// GetByID 根据ID获取
func (r *SQLiteAppPackageRepository) GetByID(id string) (*models.AppPackage, error) {
	var pkg models.AppPackage
	query := `SELECT * FROM app_packages WHERE id = ?`
	err := r.db.Get(&pkg, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("app package not found")
		}
		return nil, fmt.Errorf("failed to get app package: %w", err)
	}
	return &pkg, nil
}

// GetByPackageName 根据包名获取最新版本
func (r *SQLiteAppPackageRepository) GetByPackageName(tenantID, packageName string) (*models.AppPackage, error) {
	var pkg models.AppPackage
	query := `SELECT * FROM app_packages WHERE tenant_id = ? AND package_name = ? ORDER BY version_code DESC LIMIT 1`
	err := r.db.Get(&pkg, query, tenantID, packageName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("app package not found")
		}
		return nil, fmt.Errorf("failed to get app package: %w", err)
	}
	return &pkg, nil
}

// Update 更新应用包
func (r *SQLiteAppPackageRepository) Update(pkg *models.AppPackage) error {
	query := `
		UPDATE app_packages SET
			name = ?, version = ?, version_code = ?, file_size = ?, file_path = ?,
			sha256 = ?, description = ?, upload_url = ?, install_count = ?, updated_at = ?
		WHERE id = ?
	`
	pkg.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(query,
		pkg.Name, pkg.Version, pkg.VersionCode, pkg.FileSize, pkg.FilePath,
		pkg.SHA256, pkg.Description, pkg.UploadURL, pkg.InstallCount, pkg.UpdatedAt, pkg.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update app package: %w", err)
	}
	return nil
}

// Delete 删除应用包
func (r *SQLiteAppPackageRepository) Delete(id string) error {
	query := `DELETE FROM app_packages WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete app package: %w", err)
	}
	return nil
}

// List 获取应用包列表
func (r *SQLiteAppPackageRepository) List(tenantID string, limit, offset int) ([]*models.AppPackage, int64, error) {
	var packages []*models.AppPackage
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := `SELECT * FROM app_packages WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&packages, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list app packages: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM app_packages WHERE tenant_id = ?`
	err = r.db.Get(&total, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count app packages: %w", err)
	}

	return packages, total, nil
}

// Search 搜索应用包
func (r *SQLiteAppPackageRepository) Search(tenantID, keyword string, limit, offset int) ([]*models.AppPackage, int64, error) {
	var packages []*models.AppPackage
	if limit <= 0 {
		limit = 20
	}
	searchPattern := "%" + keyword + "%"
	query := `SELECT * FROM app_packages
		WHERE tenant_id = ? AND (name LIKE ? OR package_name LIKE ? OR description LIKE ?)
		ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.Select(&packages, query, tenantID, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search app packages: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM app_packages
		WHERE tenant_id = ? AND (name LIKE ? OR package_name LIKE ? OR description LIKE ?)`
	err = r.db.Get(&total, countQuery, tenantID, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count app packages: %w", err)
	}

	return packages, total, nil
}

// IncrementInstallCount 增加安装计数
func (r *SQLiteAppPackageRepository) IncrementInstallCount(id string) error {
	query := `UPDATE app_packages SET install_count = install_count + 1, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now().Unix(), id)
	return err
}
