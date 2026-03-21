package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// TenantService 租户服务接口
type TenantService interface {
	// 租户 CRUD
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*models.Tenant, error)
	GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, tenantID string, req *UpdateTenantRequest) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, tenantID string) error
	ListTenants(ctx context.Context, limit, offset int) ([]*models.Tenant, int64, error)

	// 配额管理
	GetQuota(ctx context.Context, tenantID string) (*models.TenantQuota, error)
	UpdateQuota(ctx context.Context, tenantID string, quota *models.TenantQuota) error
	CheckQuota(ctx context.Context, tenantID string, resource string) (bool, error)

	// 设备配额
	CheckDeviceQuota(ctx context.Context, tenantID string) (bool, error)
	IncrementDeviceCount(ctx context.Context, tenantID string) error
	DecrementDeviceCount(ctx context.Context, tenantID string) error
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Plan   string `json:"plan"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name   string `json:"name,omitempty"`
	Plan   string `json:"plan,omitempty"`
	Status string `json:"status,omitempty"`
}

type tenantService struct {
	tenantRepo repositories.TenantRepository
	deviceRepo repositories.DeviceRepository
}

// NewTenantService 创建租户服务
func NewTenantService(
	tenantRepo repositories.TenantRepository,
	deviceRepo repositories.DeviceRepository,
) TenantService {
	return &tenantService{
		tenantRepo: tenantRepo,
		deviceRepo: deviceRepo,
	}
}

func (s *tenantService) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*models.Tenant, error) {
	// 验证 slug 格式
	if req.Slug == "" {
		return nil, fmt.Errorf("slug is required")
	}

	// 检查 slug 是否已存在
	existing, err := s.tenantRepo.GetBySlug(ctx, req.Slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("tenant with slug %s already exists", req.Slug)
	}

	// 确定计划
	plan := models.TenantPlan(req.Plan)
	if plan == "" {
		plan = models.PlanStarter
	}
	if !isValidPlan(plan) {
		return nil, fmt.Errorf("invalid plan: %s", plan)
	}

	tenant := &models.Tenant{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Slug:     req.Slug,
		Plan:     string(plan),
		Status:   string(models.TenantStatusActive),
		Settings: make(map[string]interface{}),
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	slog.Info("Tenant created", "tenant_id", tenant.ID, "name", tenant.Name, "plan", tenant.Plan)
	return tenant, nil
}

func (s *tenantService) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	// Try by ID first
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err == nil {
		return tenant, nil
	}
	// Try by slug
	tenant, err = s.tenantRepo.GetBySlug(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	return tenant, nil
}

func (s *tenantService) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	tenant, err := s.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	return tenant, nil
}

// getTenantByIDOrSlug tries to get tenant by ID first, then by slug
func (s *tenantService) getTenantByIDOrSlug(ctx context.Context, tenantID string) (*models.Tenant, error) {
	// Try by ID first
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err == nil {
		return tenant, nil
	}
	// Try by slug
	tenant, err = s.tenantRepo.GetBySlug(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	return tenant, nil
}

func (s *tenantService) UpdateTenant(ctx context.Context, tenantID string, req *UpdateTenantRequest) (*models.Tenant, error) {
	tenant, err := s.getTenantByIDOrSlug(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Plan != "" {
		if !isValidPlan(models.TenantPlan(req.Plan)) {
			return nil, fmt.Errorf("invalid plan: %s", req.Plan)
		}
		tenant.Plan = req.Plan
	}
	if req.Status != "" {
		tenant.Status = req.Status
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	slog.Info("Tenant updated", "tenant_id", tenant.ID, "name", tenant.Name)
	return tenant, nil
}

func (s *tenantService) DeleteTenant(ctx context.Context, tenantID string) error {
	tenant, err := s.getTenantByIDOrSlug(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	// 软删除：设置为 deleted 状态
	tenant.Status = string(models.TenantStatusDeleted)
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	slog.Info("Tenant deleted (soft)", "tenant_id", tenantID)
	return nil
}

func (s *tenantService) ListTenants(ctx context.Context, limit, offset int) ([]*models.Tenant, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.tenantRepo.List(ctx, limit, offset)
}

func (s *tenantService) GetQuota(ctx context.Context, tenantID string) (*models.TenantQuota, error) {
	tenant, err := s.getTenantByIDOrSlug(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	quota := models.GetDefaultQuota(models.TenantPlan(tenant.Plan))

	// 如果有自定义配额，从设置中读取
	if tenant.Settings != nil {
		if maxDevices, ok := tenant.Settings["max_devices"].(float64); ok {
			quota.MaxDevices = int(maxDevices)
		}
		if maxUsers, ok := tenant.Settings["max_users"].(float64); ok {
			quota.MaxUsers = int(maxUsers)
		}
		if retentionDays, ok := tenant.Settings["retention_days"].(float64); ok {
			quota.RetentionDays = int(retentionDays)
		}
	}

	return &quota, nil
}

func (s *tenantService) UpdateQuota(ctx context.Context, tenantID string, quota *models.TenantQuota) error {
	tenant, err := s.getTenantByIDOrSlug(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}
	tenant.Settings["max_devices"] = quota.MaxDevices
	tenant.Settings["max_users"] = quota.MaxUsers
	tenant.Settings["max_groups"] = quota.MaxGroups
	tenant.Settings["retention_days"] = quota.RetentionDays
	tenant.Settings["api_rate_limit"] = quota.APIRateLimit
	tenant.Settings["storage_gb"] = quota.StorageGB

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	slog.Info("Tenant quota updated", "tenant_id", tenantID)
	return nil
}

func (s *tenantService) CheckQuota(ctx context.Context, tenantID string, resource string) (bool, error) {
	quota, err := s.GetQuota(ctx, tenantID)
	if err != nil {
		return false, err
	}

	switch resource {
	case "devices":
		count, err := s.deviceRepo.CountByTenant(ctx, tenantID)
		if err != nil {
			return false, fmt.Errorf("failed to count devices: %w", err)
		}
		return count < int64(quota.MaxDevices), nil
	default:
		return true, nil
	}
}

func (s *tenantService) CheckDeviceQuota(ctx context.Context, tenantID string) (bool, error) {
	return s.CheckQuota(ctx, tenantID, "devices")
}

func (s *tenantService) IncrementDeviceCount(ctx context.Context, tenantID string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}

	current := 0
	if count, ok := tenant.Settings["device_count"].(float64); ok {
		current = int(count)
	}
	tenant.Settings["device_count"] = current + 1
	tenant.Settings["last_device_increment"] = time.Now().Unix()

	return s.tenantRepo.Update(ctx, tenant)
}

func (s *tenantService) DecrementDeviceCount(ctx context.Context, tenantID string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	if tenant.Settings == nil {
		return nil
	}

	current := 0
	if count, ok := tenant.Settings["device_count"].(float64); ok {
		current = int(count)
	}
	if current > 0 {
		tenant.Settings["device_count"] = current - 1
	}

	return s.tenantRepo.Update(ctx, tenant)
}

// isValidPlan 检查计划是否有效
func isValidPlan(plan models.TenantPlan) bool {
	switch plan {
	case models.PlanStarter, models.PlanProfessional, models.PlanEnterprise:
		return true
	default:
		return false
	}
}
