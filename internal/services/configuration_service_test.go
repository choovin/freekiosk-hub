package services

import (
	"testing"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// MockConfigurationRepository 测试用的模拟配置仓库
type MockConfigurationRepository struct {
	configs   map[string]*models.ConfigurationProfile
	assignments map[string]string // deviceID -> configID
}

func NewMockConfigurationRepository() *MockConfigurationRepository {
	return &MockConfigurationRepository{
		configs:     make(map[string]*models.ConfigurationProfile),
		assignments: make(map[string]string),
	}
}

func (m *MockConfigurationRepository) InitSchema(ctx interface{}) error {
	return nil
}

func (m *MockConfigurationRepository) Create(cfg *models.ConfigurationProfile) error {
	m.configs[cfg.ID] = cfg
	return nil
}

func (m *MockConfigurationRepository) GetByID(id string) (*models.ConfigurationProfile, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, nil
	}
	return cfg, nil
}

func (m *MockConfigurationRepository) Update(cfg *models.ConfigurationProfile) error {
	m.configs[cfg.ID] = cfg
	return nil
}

func (m *MockConfigurationRepository) Delete(id string) error {
	delete(m.configs, id)
	return nil
}

func (m *MockConfigurationRepository) List(tenantID string, limit, offset int) ([]*models.ConfigurationProfile, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var result []*models.ConfigurationProfile
	for _, cfg := range m.configs {
		if cfg.TenantID == tenantID {
			result = append(result, cfg)
		}
	}
	total := int64(len(result))
	// Apply pagination
	if offset >= len(result) {
		return []*models.ConfigurationProfile{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *MockConfigurationRepository) AssignToDevice(deviceID, configID, tenantID, assignedBy string) error {
	m.assignments[deviceID] = configID
	return nil
}

func (m *MockConfigurationRepository) UnassignFromDevice(deviceID, configID string) error {
	delete(m.assignments, deviceID)
	return nil
}

func (m *MockConfigurationRepository) GetDeviceConfiguration(deviceID string) (*models.ConfigurationProfile, error) {
	configID, ok := m.assignments[deviceID]
	if !ok {
		return nil, nil
	}
	return m.GetByID(configID)
}

func (m *MockConfigurationRepository) GetDeviceConfigurations(deviceID string) ([]*models.ConfigurationProfile, error) {
	configID, ok := m.assignments[deviceID]
	if !ok {
		return nil, nil
	}
	cfg, _ := m.GetByID(configID)
	if cfg == nil {
		return nil, nil
	}
	return []*models.ConfigurationProfile{cfg}, nil
}

func TestConfigurationService_Create(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	cfg := &models.ConfigurationProfile{
		Name:                    "测试配置",
		Description:             "这是一个测试配置",
		TenantID:                "tenant-1",
		PasswordMinLength:       6,
		PasswordRequireNumber:   true,
		PasswordRequireSpecial:   true,
		PasswordExpireDays:      90,
		AllowInstallUnknownApps:  false,
		AllowedHoursStart:       "08:00",
		AllowedHoursEnd:         "18:00",
		DeviceTimeout:          30,
		EnableGPS:              true,
		EnableCamera:           false,
		EnableUSB:              true,
	}

	err := svc.Create(cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if cfg.ID == "" {
		t.Error("Configuration ID should be set")
	}

	if cfg.CreatedAt == 0 {
		t.Error("CreatedAt should be set")
	}

	if cfg.UpdatedAt == 0 {
		t.Error("UpdatedAt should be set")
	}

	// 验证仓库中已创建
	retrieved, err := svc.GetByID(cfg.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.Name != cfg.Name {
		t.Errorf("Expected name %s, got %s", cfg.Name, retrieved.Name)
	}
}

func TestConfigurationService_GetByID(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	// 测试不存在的配置
	_, err := svc.GetByID("non-existent")
	if err != nil {
		t.Errorf("GetByID should not return error for non-existent: %v", err)
	}
}

func TestConfigurationService_AssignToDevice(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	cfg := &models.ConfigurationProfile{
		Name:     "设备配置",
		TenantID: "tenant-1",
	}
	svc.Create(cfg)

	err := svc.AssignToDevice("device-1", cfg.ID, "tenant-1", "admin")
	if err != nil {
		t.Fatalf("AssignToDevice failed: %v", err)
	}

	// 获取设备配置
	retrieved, err := svc.GetDeviceConfiguration("device-1")
	if err != nil {
		t.Fatalf("GetDeviceConfiguration failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Device configuration should not be nil")
	}
	if retrieved.ID != cfg.ID {
		t.Errorf("Expected config ID %s, got %s", cfg.ID, retrieved.ID)
	}
}

func TestConfigurationService_UnassignFromDevice(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	cfg := &models.ConfigurationProfile{
		Name:     "设备配置",
		TenantID: "tenant-1",
	}
	svc.Create(cfg)
	svc.AssignToDevice("device-1", cfg.ID, "tenant-1", "admin")

	err := svc.UnassignFromDevice("device-1", cfg.ID)
	if err != nil {
		t.Fatalf("UnassignFromDevice failed: %v", err)
	}

	retrieved, err := svc.GetDeviceConfiguration("device-1")
	if err != nil {
		t.Fatalf("GetDeviceConfiguration failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Device configuration should be nil after unassign")
	}
}

func TestConfigurationService_List(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	// 创建多个配置
	for i := 0; i < 3; i++ {
		cfg := &models.ConfigurationProfile{
			Name:     "配置" + string(rune('A'+i)),
			TenantID: "tenant-1",
		}
		svc.Create(cfg)
	}

	configs, total, err := svc.List("tenant-1", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if len(configs) != 3 {
		t.Errorf("Expected 3 configs, got %d", len(configs))
	}

	// 测试分页
	configs, total, err = svc.List("tenant-1", 2, 0)
	if err != nil {
		t.Fatalf("List with pagination failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if len(configs) != 2 {
		t.Errorf("Expected 2 configs with limit 2, got %d", len(configs))
	}
}

func TestConfigurationService_Update(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	cfg := &models.ConfigurationProfile{
		Name:     "原始名称",
		TenantID: "tenant-1",
	}
	svc.Create(cfg)

	cfg.Name = "更新后的名称"
	err := svc.Update(cfg)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := svc.GetByID(cfg.ID)
	if retrieved.Name != "更新后的名称" {
		t.Errorf("Expected updated name, got %s", retrieved.Name)
	}
}

func TestConfigurationService_Delete(t *testing.T) {
	repo := NewMockConfigurationRepository()
	svc := NewConfigurationService(repo)

	cfg := &models.ConfigurationProfile{
		Name:     "待删除配置",
		TenantID: "tenant-1",
	}
	svc.Create(cfg)

	err := svc.Delete(cfg.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, _ := svc.GetByID(cfg.ID)
	if retrieved != nil {
		t.Error("Configuration should be nil after delete")
	}
}
