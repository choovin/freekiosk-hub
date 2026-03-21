package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// PolicyService 安全策略服务接口
type PolicyService interface {
	// 策略管理
	CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*models.SecurityPolicy, error)
	GetPolicy(ctx context.Context, policyID string) (*models.SecurityPolicy, error)
	ListPolicies(ctx context.Context, tenantID string) ([]*models.SecurityPolicy, error)
	UpdatePolicy(ctx context.Context, policy *models.SecurityPolicy) error
	DeletePolicy(ctx context.Context, policyID string) error

	// 策略分配
	AssignPolicy(ctx context.Context, policyID, deviceID string) error
	RemovePolicy(ctx context.Context, deviceID string) error
	GetDevicePolicy(ctx context.Context, deviceID string) (*models.SecurityPolicy, error)

	// 应用白名单管理
	AddAppToWhitelist(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error
	RemoveAppFromWhitelist(ctx context.Context, policyID, packageName string) error
	UpdateAppWhitelist(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error
	GetDeviceWhitelist(ctx context.Context, deviceID string) ([]models.AppWhitelistEntry, error)
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Settings    *models.SecurityPolicySettings `json:"settings,omitempty"`
	AppWhitelist []models.AppWhitelistEntry `json:"app_whitelist,omitempty"`
}

// PolicyServiceConfig 策略服务配置
type PolicyServiceConfig struct {
	DefaultPolicyID string
}

type policyService struct {
	policyRepo repositories.SecurityPolicyRepository
	appRepo    repositories.AppWhitelistRepository
	config     PolicyServiceConfig
}

// NewPolicyService 创建策略服务
func NewPolicyService(
	policyRepo repositories.SecurityPolicyRepository,
	appRepo repositories.AppWhitelistRepository,
	config PolicyServiceConfig,
) PolicyService {
	return &policyService{
		policyRepo: policyRepo,
		appRepo:    appRepo,
		config:     config,
	}
}

// CreatePolicy 创建安全策略
func (s *policyService) CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*models.SecurityPolicy, error) {
	policy := &models.SecurityPolicy{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
	}

	// 如果没有提供设置，使用默认设置
	if req.Settings != nil {
		policy.Settings = *req.Settings
	} else {
		defaultPolicy := models.GetDefaultSecurityPolicy(tenantID)
		policy.Settings = defaultPolicy.Settings
	}

	// 如果没有提供白名单，使用空白名单
	if req.AppWhitelist != nil {
		policy.AppWhitelist = req.AppWhitelist
	} else {
		policy.AppWhitelist = []models.AppWhitelistEntry{}
	}

	if err := s.policyRepo.Create(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	slog.Info("Created security policy",
		"policyId", policy.ID,
		"tenantId", tenantID,
		"name", policy.Name,
	)

	return policy, nil
}

// GetPolicy 获取策略
func (s *policyService) GetPolicy(ctx context.Context, policyID string) (*models.SecurityPolicy, error) {
	policy, err := s.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return policy, nil
}

// ListPolicies 列出租户的所有策略
func (s *policyService) ListPolicies(ctx context.Context, tenantID string) ([]*models.SecurityPolicy, error) {
	policies, err := s.policyRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, nil
}

// UpdatePolicy 更新策略
func (s *policyService) UpdatePolicy(ctx context.Context, policy *models.SecurityPolicy) error {
	if err := s.policyRepo.Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	slog.Info("Updated security policy",
		"policyId", policy.ID,
		"name", policy.Name,
	)

	return nil
}

// DeletePolicy 删除策略
func (s *policyService) DeletePolicy(ctx context.Context, policyID string) error {
	// 检查是否是默认策略
	if policyID == s.config.DefaultPolicyID {
		return fmt.Errorf("cannot delete default policy")
	}

	if err := s.policyRepo.Delete(ctx, policyID); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	slog.Info("Deleted security policy",
		"policyId", policyID,
	)

	return nil
}

// AssignPolicy 分配策略给设备
func (s *policyService) AssignPolicy(ctx context.Context, policyID, deviceID string) error {
	if err := s.policyRepo.AssignToDevice(ctx, policyID, deviceID); err != nil {
		return fmt.Errorf("failed to assign policy: %w", err)
	}

	slog.Info("Assigned policy to device",
		"policyId", policyID,
		"deviceId", deviceID,
	)

	return nil
}

// RemovePolicy 移除设备的策略
func (s *policyService) RemovePolicy(ctx context.Context, deviceID string) error {
	if err := s.policyRepo.RemoveFromDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}

	slog.Info("Removed policy from device",
		"deviceId", deviceID,
	)

	return nil
}

// GetDevicePolicy 获取设备的策略
func (s *policyService) GetDevicePolicy(ctx context.Context, deviceID string) (*models.SecurityPolicy, error) {
	policy, err := s.policyRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device policy: %w", err)
	}
	return policy, nil
}

// AddAppToWhitelist 添加应用到白名单
func (s *policyService) AddAppToWhitelist(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error {
	if err := s.appRepo.AddApp(ctx, policyID, entry); err != nil {
		return fmt.Errorf("failed to add app to whitelist: %w", err)
	}

	slog.Info("Added app to whitelist",
		"policyId", policyID,
		"packageName", entry.PackageName,
	)

	return nil
}

// RemoveAppFromWhitelist 从白名单移除应用
func (s *policyService) RemoveAppFromWhitelist(ctx context.Context, policyID, packageName string) error {
	if err := s.appRepo.RemoveApp(ctx, policyID, packageName); err != nil {
		return fmt.Errorf("failed to remove app from whitelist: %w", err)
	}

	slog.Info("Removed app from whitelist",
		"policyId", policyID,
		"packageName", packageName,
	)

	return nil
}

// UpdateAppWhitelist 更新白名单中的应用
func (s *policyService) UpdateAppWhitelist(ctx context.Context, policyID string, entry models.AppWhitelistEntry) error {
	if err := s.appRepo.UpdateApp(ctx, policyID, entry); err != nil {
		return fmt.Errorf("failed to update app whitelist: %w", err)
	}

	slog.Info("Updated app in whitelist",
		"policyId", policyID,
		"packageName", entry.PackageName,
	)

	return nil
}

// GetDeviceWhitelist 获取设备的有效白名单
func (s *policyService) GetDeviceWhitelist(ctx context.Context, deviceID string) ([]models.AppWhitelistEntry, error) {
	whitelist, err := s.appRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device whitelist: %w", err)
	}
	return whitelist, nil
}
