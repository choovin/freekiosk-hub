package services

import (
	"testing"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// MockMDMTabletRepository is a mock implementation for testing
type MockMDMTabletRepository struct {
	devices map[string]*models.MDMTablet
	groups  map[string]*models.MDMTabletGroup
	tags    map[string][]*models.MDMTabletTag
	events  map[string][]*models.MDMTabletEvent
}

func NewMockMDMTabletRepository() *MockMDMTabletRepository {
	return &MockMDMTabletRepository{
		devices: make(map[string]*models.MDMTablet),
		groups:  make(map[string]*models.MDMTabletGroup),
		tags:    make(map[string][]*models.MDMTabletTag),
		events:  make(map[string][]*models.MDMTabletEvent),
	}
}

func (m *MockMDMTabletRepository) InitSchema(ctx interface{}) error {
	return nil
}

func (m *MockMDMTabletRepository) CreateDevice(device *models.MDMTablet) error {
	now := time.Now().Unix()
	device.CreatedAt = now
	device.UpdatedAt = now
	m.devices[device.ID] = device
	return nil
}

func (m *MockMDMTabletRepository) GetDeviceByID(id string) (*models.MDMTablet, error) {
	if d, ok := m.devices[id]; ok {
		return d, nil
	}
	return nil, repositories.ErrDeviceNotFound
}

func (m *MockMDMTabletRepository) GetDeviceByNumber(number string) (*models.MDMTablet, error) {
	for _, d := range m.devices {
		if d.Number == number {
			return d, nil
		}
	}
	return nil, repositories.ErrDeviceNotFound
}

func (m *MockMDMTabletRepository) UpdateDevice(device *models.MDMTablet) error {
	device.UpdatedAt = time.Now().Unix()
	m.devices[device.ID] = device
	return nil
}

func (m *MockMDMTabletRepository) DeleteDevice(id string) error {
	if d, ok := m.devices[id]; ok {
		d.Status = string(models.MDMTabletStatusRetired)
		return nil
	}
	return repositories.ErrDeviceNotFound
}

func (m *MockMDMTabletRepository) ListDevices(tenantID string, limit, offset int) ([]*models.MDMTablet, int64, error) {
	var result []*models.MDMTablet
	for _, d := range m.devices {
		if d.TenantID == tenantID && d.Status != string(models.MDMTabletStatusRetired) {
			result = append(result, d)
		}
	}
	total := int64(len(result))
	if offset >= len(result) {
		return []*models.MDMTablet{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *MockMDMTabletRepository) SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMTablet, int64, error) {
	var result []*models.MDMTablet
	for _, d := range m.devices {
		if d.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && d.Status != filter.Status {
			continue
		}
		if filter.Search != "" {
			if d.Name != filter.Search && d.Number != filter.Search && d.IMEI != filter.Search {
				continue
			}
		}
		result = append(result, d)
	}
	return result, int64(len(result)), nil
}

func (m *MockMDMTabletRepository) CreateGroup(group *models.MDMTabletGroup) error {
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	m.groups[group.ID] = group
	return nil
}

func (m *MockMDMTabletRepository) UpdateGroup(group *models.MDMTabletGroup) error {
	group.UpdatedAt = time.Now().Unix()
	m.groups[group.ID] = group
	return nil
}

func (m *MockMDMTabletRepository) DeleteGroup(id string) error {
	delete(m.groups, id)
	return nil
}

func (m *MockMDMTabletRepository) ListGroups(tenantID string) ([]*models.MDMTabletGroup, error) {
	var result []*models.MDMTabletGroup
	for _, g := range m.groups {
		if g.TenantID == tenantID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *MockMDMTabletRepository) AddTag(tag *models.MDMTabletTag) error {
	tag.CreatedAt = time.Now().Unix()
	m.tags[tag.DeviceID] = append(m.tags[tag.DeviceID], tag)
	return nil
}

func (m *MockMDMTabletRepository) RemoveTag(deviceID, tagName string) error {
	if tags, ok := m.tags[deviceID]; ok {
		for i, t := range tags {
			if t.Tag == tagName {
				m.tags[deviceID] = append(tags[:i], tags[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *MockMDMTabletRepository) GetDeviceTags(deviceID string) ([]*models.MDMTabletTag, error) {
	return m.tags[deviceID], nil
}

func (m *MockMDMTabletRepository) UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error {
	if d, ok := m.devices[deviceID]; ok {
		d.LastLat = &lat
		d.LastLng = &lng
		d.LastLocationTime = &timestamp
		return nil
	}
	return repositories.ErrDeviceNotFound
}

func (m *MockMDMTabletRepository) GetDeviceLocation(deviceID string) (*models.GPSData, error) {
	if d, ok := m.devices[deviceID]; ok {
		if d.LastLat != nil && d.LastLng != nil {
			return &models.GPSData{
				Lat: *d.LastLat,
				Lng: *d.LastLng,
			}, nil
		}
	}
	return nil, repositories.ErrDeviceNotFound
}

func (m *MockMDMTabletRepository) RecordEvent(event *models.MDMTabletEvent) error {
	event.CreatedAt = time.Now().Unix()
	m.events[event.DeviceID] = append(m.events[event.DeviceID], event)
	return nil
}

func (m *MockMDMTabletRepository) GetDeviceEvents(deviceID string, limit int) ([]*models.MDMTabletEvent, error) {
	events := m.events[deviceID]
	if len(events) > limit {
		return events[:limit], nil
	}
	return events, nil
}

func (m *MockMDMTabletRepository) BulkUpdateStatus(deviceIDs []string, status string) error {
	for _, id := range deviceIDs {
		if d, ok := m.devices[id]; ok {
			d.Status = status
		}
	}
	return nil
}

func (m *MockMDMTabletRepository) BulkAssignGroup(deviceIDs []string, groupID string) error {
	for _, id := range deviceIDs {
		if d, ok := m.devices[id]; ok {
			d.GroupID = &groupID
		}
	}
	return nil
}

// TestMDMTabletService_CreateDevice tests creating a device
func TestMDMTabletService_CreateDevice(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "test-device-001",
		Number:   "MDM-001",
		Name:     "测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}

	err := svc.CreateDevice(device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Verify device was created
	retrieved, err := svc.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if retrieved.Name != device.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, device.Name)
	}
}

// TestMDMTabletService_GetDevice tests retrieving a device
func TestMDMTabletService_GetDevice(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "test-device-002",
		Number:   "MDM-002",
		Name:     "测试设备2",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	retrieved, err := svc.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if retrieved.ID != device.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, device.ID)
	}
}

// TestMDMTabletService_GetDevice_NotFound tests error when device not found
func TestMDMTabletService_GetDevice_NotFound(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	_, err := svc.GetDevice("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent device, got nil")
	}
}

// TestMDMTabletService_ListDevices tests device listing with pagination
func TestMDMTabletService_ListDevices(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	// Create 5 devices
	for i := 1; i <= 5; i++ {
		device := &models.MDMTablet{
			ID:       "list-device-" + string(rune('0'+i)),
			Number:   "LIST-" + string(rune('0'+i)),
			Name:     "列表设备" + string(rune('0'+i)),
			TenantID: "list-tenant",
			Status:   string(models.MDMTabletStatusActive),
		}
		svc.CreateDevice(device)
	}

	devices, total, err := svc.ListDevices("list-tenant", 3, 0)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Total mismatch: got %d, want 5", total)
	}
	if len(devices) != 3 {
		t.Errorf("Page size mismatch: got %d, want 3", len(devices))
	}
}

// TestMDMTabletService_SearchDevices tests device search
func TestMDMTabletService_SearchDevices(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	devices := []*models.MDMTablet{
		{ID: "search-001", Number: "SEARCH-001", Name: "教室A-01", TenantID: "search-tenant", Status: string(models.MDMTabletStatusActive)},
		{ID: "search-002", Number: "SEARCH-002", Name: "教室A-02", TenantID: "search-tenant", Status: string(models.MDMTabletStatusActive)},
		{ID: "search-003", Number: "SEARCH-003", Name: "教室B-01", TenantID: "search-tenant", Status: string(models.MDMTabletStatusInactive)},
	}
	for _, d := range devices {
		repo.CreateDevice(d)
	}

	filter := &models.DeviceSearchFilter{
		TenantID: "search-tenant",
		Status:   string(models.MDMTabletStatusActive),
	}
	_, total, err := svc.SearchDevices(filter)
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Total mismatch: got %d, want 2", total)
	}
}

// TestMDMTabletService_UpdateDeviceStatus tests updating device status
func TestMDMTabletService_UpdateDeviceStatus(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "status-device",
		Number:   "STATUS-001",
		Name:     "状态测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	err := svc.UpdateDeviceStatus(device.ID, models.MDMTabletStatusInactive)
	if err != nil {
		t.Fatalf("UpdateDeviceStatus failed: %v", err)
	}

	updated, _ := svc.GetDevice(device.ID)
	if updated.Status != string(models.MDMTabletStatusInactive) {
		t.Errorf("Status not updated: got %s, want inactive", updated.Status)
	}
}

// TestMDMTabletService_BulkUpdateStatus tests bulk status update
func TestMDMTabletService_BulkUpdateStatus(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	deviceIDs := []string{}
	for i := 1; i <= 3; i++ {
		device := &models.MDMTablet{
			ID:       "bulk-status-" + string(rune('0'+i)),
			Number:   "BULK-" + string(rune('0'+i)),
			Name:     "批量测试设备",
			TenantID: "tenant-001",
			Status:   string(models.MDMTabletStatusActive),
		}
		repo.CreateDevice(device)
		deviceIDs = append(deviceIDs, device.ID)
	}

	err := svc.BulkUpdateStatus(deviceIDs, models.MDMTabletStatusInactive)
	if err != nil {
		t.Fatalf("BulkUpdateStatus failed: %v", err)
	}

	for _, id := range deviceIDs {
		device, _ := svc.GetDevice(id)
		if device.Status != string(models.MDMTabletStatusInactive) {
			t.Errorf("Device %s status not updated", id)
		}
	}
}

// TestMDMTabletService_CreateGroup tests group creation
func TestMDMTabletService_CreateGroup(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	group := &models.MDMTabletGroup{
		ID:          "group-001",
		Name:        "初三班级",
		Description: "初三学年的所有设备",
		TenantID:    "tenant-001",
	}

	err := svc.CreateGroup(group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groups, err := svc.ListGroups("tenant-001")
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("Groups count mismatch: got %d, want 1", len(groups))
	}
}

// TestMDMTabletService_AssignDeviceToGroup tests assigning device to group
func TestMDMTabletService_AssignDeviceToGroup(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	group := &models.MDMTabletGroup{
		ID:       "group-001",
		Name:     "测试分组",
		TenantID: "tenant-001",
	}
	svc.CreateGroup(group)

	device := &models.MDMTablet{
		ID:       "device-to-group",
		Number:   "GRP-001",
		Name:     "分组测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	svc.CreateDevice(device)

	err := svc.AssignDeviceToGroup(device.ID, group.ID)
	if err != nil {
		t.Fatalf("AssignDeviceToGroup failed: %v", err)
	}

	updated, _ := svc.GetDevice(device.ID)
	if updated.GroupID == nil || *updated.GroupID != group.ID {
		t.Errorf("Device not assigned to group")
	}
}

// TestMDMTabletService_UpdateLocation tests location update
func TestMDMTabletService_UpdateLocation(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "location-device",
		Number:   "LOC-001",
		Name:     "定位测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	lat := 39.9042
	lng := 116.4074
	timestamp := time.Now().Unix()

	err := svc.UpdateLocation(device.ID, lat, lng, timestamp)
	if err != nil {
		t.Fatalf("UpdateLocation failed: %v", err)
	}

	location, err := svc.GetDeviceLocation(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceLocation failed: %v", err)
	}
	if location.Lat != lat {
		t.Errorf("Lat mismatch: got %f, want %f", location.Lat, lat)
	}
}

// TestMDMTabletService_AddTag tests adding device tag
func TestMDMTabletService_AddTag(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "tag-device",
		Number:   "TAG-001",
		Name:     "标签测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	tag := &models.MDMTabletTag{
		ID:       "tag-001",
		DeviceID: device.ID,
		Tag:      "location",
		Value:    "教室A",
	}
	err := svc.AddTag(tag)
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	tags, err := svc.GetDeviceTags(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceTags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("Tags count mismatch: got %d, want 1", len(tags))
	}
}

// TestMDMTabletService_RecordEvent tests recording device event
func TestMDMTabletService_RecordEvent(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "event-device",
		Number:   "EVT-001",
		Name:     "事件测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	event := &models.MDMTabletEvent{
		ID:        "event-001",
		DeviceID:  device.ID,
		EventType: "location_update",
		EventData: `{"lat":39.9042,"lng":116.4074}`,
	}
	err := svc.RecordEvent(event)
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}

	events, err := svc.GetDeviceEvents(device.ID, 10)
	if err != nil {
		t.Fatalf("GetDeviceEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Events count mismatch: got %d, want 1", len(events))
	}
}

// TestMDMTabletService_SoftDelete tests soft delete of device
func TestMDMTabletService_SoftDelete(t *testing.T) {
	repo := NewMockMDMTabletRepository()
	svc := NewMDMTabletService(repo)

	device := &models.MDMTablet{
		ID:       "delete-device",
		Number:   "DEL-001",
		Name:     "删除测试设备",
		TenantID: "tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}
	repo.CreateDevice(device)

	err := svc.DeleteDevice(device.ID)
	if err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	deleted, _ := svc.GetDevice(device.ID)
	if deleted.Status != string(models.MDMTabletStatusRetired) {
		t.Errorf("Device not retired: got %s, want retired", deleted.Status)
	}
}
