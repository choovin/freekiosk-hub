package services

import (
	"testing"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// MockGeofenceRepository 测试用的模拟地理围栏仓库
type MockGeofenceRepository struct {
	geofences   map[string]*models.Geofence
	assignments map[string][]string // geofenceID -> deviceIDs
	events      []*models.GeofenceEvent
}

func NewMockGeofenceRepository() *MockGeofenceRepository {
	return &MockGeofenceRepository{
		geofences:   make(map[string]*models.Geofence),
		assignments: make(map[string][]string),
		events:      []*models.GeofenceEvent{},
	}
}

func (m *MockGeofenceRepository) InitSchema(ctx interface{}) error {
	return nil
}

func (m *MockGeofenceRepository) Create(gf *models.Geofence) error {
	m.geofences[gf.ID] = gf
	return nil
}

func (m *MockGeofenceRepository) GetByID(id string) (*models.Geofence, error) {
	gf, ok := m.geofences[id]
	if !ok {
		return nil, nil
	}
	return gf, nil
}

func (m *MockGeofenceRepository) Update(gf *models.Geofence) error {
	m.geofences[gf.ID] = gf
	return nil
}

func (m *MockGeofenceRepository) Delete(id string) error {
	delete(m.geofences, id)
	return nil
}

func (m *MockGeofenceRepository) List(tenantID string, limit, offset int) ([]*models.Geofence, int64, error) {
	var result []*models.Geofence
	for _, gf := range m.geofences {
		if gf.TenantID == tenantID {
			result = append(result, gf)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockGeofenceRepository) ListActive(tenantID string) ([]*models.Geofence, error) {
	var result []*models.Geofence
	for _, gf := range m.geofences {
		if gf.TenantID == tenantID && gf.IsActive {
			result = append(result, gf)
		}
	}
	return result, nil
}

func (m *MockGeofenceRepository) AssignDevice(geofenceID, deviceID, tenantID, assignedBy string) error {
	m.assignments[geofenceID] = append(m.assignments[geofenceID], deviceID)
	return nil
}

func (m *MockGeofenceRepository) UnassignDevice(geofenceID, deviceID string) error {
	devices := m.assignments[geofenceID]
	for i, d := range devices {
		if d == deviceID {
			m.assignments[geofenceID] = append(devices[:i], devices[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockGeofenceRepository) GetDeviceGeofences(deviceID string) ([]*models.Geofence, error) {
	var result []*models.Geofence
	for _, gf := range m.geofences {
		if gf.IsActive {
			for _, assignedDevice := range m.assignments[gf.ID] {
				if assignedDevice == deviceID {
					result = append(result, gf)
					break
				}
			}
		}
	}
	return result, nil
}

func (m *MockGeofenceRepository) GetAssignedDevices(geofenceID string) ([]string, error) {
	return m.assignments[geofenceID], nil
}

func (m *MockGeofenceRepository) RecordEvent(event *models.GeofenceEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockGeofenceRepository) GetDeviceEvents(deviceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	var result []*models.GeofenceEvent
	for _, e := range m.events {
		if e.DeviceID == deviceID {
			result = append(result, e)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockGeofenceRepository) GetGeofenceEvents(geofenceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	var result []*models.GeofenceEvent
	for _, e := range m.events {
		if e.GeofenceID == geofenceID {
			result = append(result, e)
		}
	}
	return result, int64(len(result)), nil
}

func TestGeofenceService_Create(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:         "测试围栏",
		Description:  "这是一个测试围栏",
		TenantID:     "tenant-1",
		FenceType:    "circle",
		Latitude:     39.9042,
		Longitude:   116.4074,
		Radius:      500,
		IsActive:    true,
		AlertOnEnter: true,
		AlertOnExit:  true,
	}

	err := svc.Create(gf)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if gf.ID == "" {
		t.Error("Geofence ID should be set")
	}

	if gf.CreatedAt == 0 {
		t.Error("CreatedAt should be set")
	}

	retrieved, err := svc.GetByID(gf.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.Name != gf.Name {
		t.Errorf("Expected name %s, got %s", gf.Name, retrieved.Name)
	}
}

func TestGeofenceService_AssignToDevice(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:       "围栏1",
		TenantID:   "tenant-1",
		FenceType:  "circle",
		Latitude:   39.9042,
		Longitude:  116.4074,
		Radius:     500,
		IsActive:   true,
	}
	svc.Create(gf)

	err := svc.AssignToDevice(gf.ID, "device-1", "tenant-1", "admin")
	if err != nil {
		t.Fatalf("AssignToDevice failed: %v", err)
	}

	geofences, err := svc.GetDeviceGeofences("device-1")
	if err != nil {
		t.Fatalf("GetDeviceGeofences failed: %v", err)
	}
	if len(geofences) != 1 {
		t.Errorf("Expected 1 geofence, got %d", len(geofences))
	}
}

func TestGeofenceService_UnassignFromDevice(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:       "围栏1",
		TenantID:   "tenant-1",
		FenceType:  "circle",
		Latitude:   39.9042,
		Longitude:  116.4074,
		Radius:     500,
		IsActive:   true,
	}
	svc.Create(gf)
	svc.AssignToDevice(gf.ID, "device-1", "tenant-1", "admin")

	err := svc.UnassignFromDevice(gf.ID, "device-1")
	if err != nil {
		t.Fatalf("UnassignFromDevice failed: %v", err)
	}

	geofences, _ := svc.GetDeviceGeofences("device-1")
	if len(geofences) != 0 {
		t.Errorf("Expected 0 geofences after unassign, got %d", len(geofences))
	}
}

func TestGeofenceService_ListActive(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	// 创建激活的围栏
	gf1 := &models.Geofence{Name: "围栏1", TenantID: "tenant-1", IsActive: true, FenceType: "circle", Latitude: 39.9, Longitude: 116.4, Radius: 100}
	svc.Create(gf1)

	// 创建未激活的围栏
	gf2 := &models.Geofence{Name: "围栏2", TenantID: "tenant-1", IsActive: false, FenceType: "circle", Latitude: 39.9, Longitude: 116.4, Radius: 100}
	svc.Create(gf2)

	active, err := svc.ListActive("tenant-1")
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("Expected 1 active geofence, got %d", len(active))
	}
	if active[0].Name != "围栏1" {
		t.Errorf("Expected active geofence name '围栏1', got '%s'", active[0].Name)
	}
}

func TestGeofenceService_RecordEvent(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:       "围栏1",
		TenantID:   "tenant-1",
		FenceType:  "circle",
		Latitude:   39.9042,
		Longitude:  116.4074,
		Radius:     500,
		IsActive:   true,
	}
	svc.Create(gf)

	err := svc.RecordEvent(gf.ID, "device-1", "tenant-1", "enter", 39.9042, 116.4074)
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}

	events, total, err := svc.GetDeviceEvents("device-1", 50, 0)
	if err != nil {
		t.Fatalf("GetDeviceEvents failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 event, got %d", total)
	}
	if events[0].EventType != "enter" {
		t.Errorf("Expected event type 'enter', got '%s'", events[0].EventType)
	}
}

func TestGeofenceService_Delete(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:     "待删除围栏",
		TenantID: "tenant-1",
		FenceType: "circle",
		Latitude:  39.9,
		Longitude: 116.4,
		Radius:    100,
	}
	svc.Create(gf)

	err := svc.Delete(gf.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, _ := svc.GetByID(gf.ID)
	if retrieved != nil {
		t.Error("Geofence should be nil after delete")
	}
}

func TestGeofenceService_Update(t *testing.T) {
	repo := NewMockGeofenceRepository()
	svc := NewGeofenceService(repo)

	gf := &models.Geofence{
		Name:     "原始名称",
		TenantID: "tenant-1",
		FenceType: "circle",
		Latitude:  39.9,
		Longitude: 116.4,
		Radius:    100,
	}
	svc.Create(gf)

	gf.Name = "新名称"
	err := svc.Update(gf)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := svc.GetByID(gf.ID)
	if retrieved.Name != "新名称" {
		t.Errorf("Expected updated name '新名称', got '%s'", retrieved.Name)
	}
}
