package repositories

import (
	"testing"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// TestMDMTabletRepository_CreateDevice tests creating a new MDM device
func TestMDMTabletRepository_CreateDevice(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:             "test-device-001",
		Number:         "FKD-001",
		Name:           "测试设备-01",
		Description:    "这是一台测试设备",
		IMEI:           "860855041234567",
		Phone:          "13800138000",
		Model:          "华为MatePad SE",
		Manufacturer:   "华为",
		OSVersion:      "Android 12",
		SDKVersion:     31,
		AppVersion:     "2.3.1",
		AppVersionCode: 231,
		Carrier:        "中国移动",
		Status:         string(models.MDMTabletStatusActive),
		TenantID:       "test-tenant-001",
		Metadata:       `{"location":"教室A"}`,
	}

	err := repo.CreateDevice(device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Verify device was created
	created, err := repo.GetDeviceByID(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceByID failed: %v", err)
	}

	if created.Number != device.Number {
		t.Errorf("Number mismatch: got %s, want %s", created.Number, device.Number)
	}
	if created.Name != device.Name {
		t.Errorf("Name mismatch: got %s, want %s", created.Name, device.Name)
	}
	if created.IMEI != device.IMEI {
		t.Errorf("IMEI mismatch: got %s, want %s", created.IMEI, device.IMEI)
	}
	if created.Status != string(models.MDMTabletStatusActive) {
		t.Errorf("Status mismatch: got %s, want active", created.Status)
	}
}

// TestMDMTabletRepository_GetDeviceByNumber tests retrieving device by number
func TestMDMTabletRepository_GetDeviceByNumber(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "test-device-002",
		Number:   "FKD-002",
		Name:     "测试设备-02",
		TenantID: "test-tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}

	err := repo.CreateDevice(device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	retrieved, err := repo.GetDeviceByNumber("FKD-002")
	if err != nil {
		t.Fatalf("GetDeviceByNumber failed: %v", err)
	}

	if retrieved.ID != device.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, device.ID)
	}
}

// TestMDMTabletRepository_UpdateDevice tests updating device information
func TestMDMTabletRepository_UpdateDevice(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "test-device-003",
		Number:   "FKD-003",
		Name:     "原始名称",
		TenantID: "test-tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}

	err := repo.CreateDevice(device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Update device
	newName := "更新后的名称"
	newStatus := string(models.MDMTabletStatusInactive)
	device.Name = newName
	device.Status = newStatus
	device.UpdatedAt = time.Now().Unix()

	err = repo.UpdateDevice(device)
	if err != nil {
		t.Fatalf("UpdateDevice failed: %v", err)
	}

	// Verify update
	updated, err := repo.GetDeviceByID(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceByID after update failed: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Name not updated: got %s, want %s", updated.Name, newName)
	}
	if updated.Status != newStatus {
		t.Errorf("Status not updated: got %s, want %s", updated.Status, newStatus)
	}
}

// TestMDMTabletRepository_DeleteDevice tests soft delete of device
func TestMDMTabletRepository_DeleteDevice(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "test-device-004",
		Number:   "FKD-004",
		Name:     "待删除设备",
		TenantID: "test-tenant-001",
		Status:   string(models.MDMTabletStatusActive),
	}

	err := repo.CreateDevice(device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = repo.DeleteDevice(device.ID)
	if err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	// Verify device is marked as retired
	deleted, err := repo.GetDeviceByID(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceByID after delete failed: %v", err)
	}

	if deleted.Status != string(models.MDMTabletStatusRetired) {
		t.Errorf("Device not retired: got %s, want retired", deleted.Status)
	}
}

// TestMDMTabletRepository_SearchDevices tests device search functionality
func TestMDMTabletRepository_SearchDevices(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	// Create test devices
	devices := []*models.MDMTablet{
		{ID: "search-001", Number: "SEARCH-001", Name: "教室A-01", TenantID: "search-tenant", Status: string(models.MDMTabletStatusActive)},
		{ID: "search-002", Number: "SEARCH-002", Name: "教室A-02", TenantID: "search-tenant", Status: string(models.MDMTabletStatusActive)},
		{ID: "search-003", Number: "SEARCH-003", Name: "教室B-01", TenantID: "search-tenant", Status: string(models.MDMTabletStatusInactive)},
		{ID: "search-004", Number: "SEARCH-004", Name: "办公室-01", TenantID: "other-tenant", Status: string(models.MDMTabletStatusActive)},
	}

	for _, d := range devices {
		if err := repo.CreateDevice(d); err != nil {
			t.Fatalf("CreateDevice %s failed: %v", d.ID, err)
		}
	}

	// Test: Search by tenant
	filter := &models.DeviceSearchFilter{
		TenantID: "search-tenant",
		Limit:    50,
	}
	results, total, err := repo.SearchDevices(filter)
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Total mismatch: got %d, want 3", total)
	}
	if len(results) != 3 {
		t.Errorf("Results count mismatch: got %d, want 3", len(results))
	}

	// Test: Search by status
	filter2 := &models.DeviceSearchFilter{
		TenantID: "search-tenant",
		Status:   string(models.MDMTabletStatusActive),
		Limit:    50,
	}
	_, total2, err := repo.SearchDevices(filter2)
	if err != nil {
		t.Fatalf("SearchDevices by status failed: %v", err)
	}
	if total2 != 2 {
		t.Errorf("Total by status mismatch: got %d, want 2", total2)
	}

	// Test: Search by keyword
	filter3 := &models.DeviceSearchFilter{
		TenantID: "search-tenant",
		Search:   "教室A",
		Limit:    50,
	}
	_, total3, err := repo.SearchDevices(filter3)
	if err != nil {
		t.Fatalf("SearchDevices by keyword failed: %v", err)
	}
	if total3 != 2 {
		t.Errorf("Total by keyword mismatch: got %d, want 2", total3)
	}
}

// TestMDMTabletRepository_ListDevices tests device listing with pagination
func TestMDMTabletRepository_ListDevices(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	// Create 5 devices
	for i := 1; i <= 5; i++ {
		device := &models.MDMTablet{
			ID:       "list-device-" + string(rune('0'+i)),
			Number:   "LIST-" + string(rune('0'+i)),
			Name:     "列表设备" + string(rune('0'+i)),
			TenantID: "list-tenant",
			Status:   string(models.MDMTabletStatusActive),
		}
		if err := repo.CreateDevice(device); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
	}

	// Test: List first page
	devices, total, err := repo.ListDevices("list-tenant", 3, 0)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Total mismatch: got %d, want 5", total)
	}
	if len(devices) != 3 {
		t.Errorf("Page size mismatch: got %d, want 3", len(devices))
	}

	// Test: List second page
	devices2, _, err := repo.ListDevices("list-tenant", 3, 3)
	if err != nil {
		t.Fatalf("ListDevices page 2 failed: %v", err)
	}
	if len(devices2) != 2 {
		t.Errorf("Page 2 size mismatch: got %d, want 2", len(devices2))
	}
}

// TestMDMTabletRepository_CreateGroup tests device group creation
func TestMDMTabletRepository_CreateGroup(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	group := &models.MDMTabletGroup{
		ID:          "group-001",
		Name:        "初三班级",
		Description: "初三学年的所有设备",
		TenantID:    "group-tenant",
	}

	err := repo.CreateGroup(group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groups, err := repo.ListGroups("group-tenant")
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("Groups count mismatch: got %d, want 1", len(groups))
	}
	if groups[0].Name != group.Name {
		t.Errorf("Group name mismatch: got %s, want %s", groups[0].Name, group.Name)
	}
}

// TestMDMTabletRepository_UpdateLocation tests GPS location update
func TestMDMTabletRepository_UpdateLocation(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "location-device",
		Number:   "LOC-001",
		Name:     "定位测试设备",
		TenantID: "location-tenant",
		Status:   string(models.MDMTabletStatusActive),
	}
	if err := repo.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	lat := 39.9042
	lng := 116.4074
	timestamp := time.Now().Unix()

	err := repo.UpdateLocation(device.ID, lat, lng, timestamp)
	if err != nil {
		t.Fatalf("UpdateLocation failed: %v", err)
	}

	location, err := repo.GetDeviceLocation(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceLocation failed: %v", err)
	}

	if location.Lat != lat {
		t.Errorf("Lat mismatch: got %f, want %f", location.Lat, lat)
	}
	if location.Lng != lng {
		t.Errorf("Lng mismatch: got %f, want %f", location.Lng, lng)
	}
}

// TestMDMTabletRepository_RecordEvent tests device event recording
func TestMDMTabletRepository_RecordEvent(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "event-device",
		Number:   "EVT-001",
		Name:     "事件测试设备",
		TenantID: "event-tenant",
		Status:   string(models.MDMTabletStatusActive),
	}
	if err := repo.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	event := &models.MDMTabletEvent{
		ID:        "event-001",
		DeviceID:  device.ID,
		EventType: "location_update",
		EventData: `{"lat":39.9042,"lng":116.4074}`,
	}
	err := repo.RecordEvent(event)
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}

	events, err := repo.GetDeviceEvents(device.ID, 10)
	if err != nil {
		t.Fatalf("GetDeviceEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Events count mismatch: got %d, want 1", len(events))
	}
	if events[0].EventType != "location_update" {
		t.Errorf("EventType mismatch: got %s, want location_update", events[0].EventType)
	}
}

// TestMDMTabletRepository_BulkUpdateStatus tests bulk status update
func TestMDMTabletRepository_BulkUpdateStatus(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	// Create 3 devices
	deviceIDs := []string{}
	for i := 1; i <= 3; i++ {
		device := &models.MDMTablet{
			ID:       "bulk-status-" + string(rune('0'+i)),
			Number:   "BULK-" + string(rune('0'+i)),
			Name:     "批量测试设备" + string(rune('0'+i)),
			TenantID: "bulk-tenant",
			Status:   string(models.MDMTabletStatusActive),
		}
		if err := repo.CreateDevice(device); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
		deviceIDs = append(deviceIDs, device.ID)
	}

	// Bulk update to inactive
	err := repo.BulkUpdateStatus(deviceIDs, string(models.MDMTabletStatusInactive))
	if err != nil {
		t.Fatalf("BulkUpdateStatus failed: %v", err)
	}

	// Verify all devices are inactive
	for _, id := range deviceIDs {
		device, err := repo.GetDeviceByID(id)
		if err != nil {
			t.Fatalf("GetDeviceByID failed for %s: %v", id, err)
		}
		if device.Status != string(models.MDMTabletStatusInactive) {
			t.Errorf("Device %s status not updated: got %s, want inactive", id, device.Status)
		}
	}
}

// TestMDMTabletRepository_AddTag tests adding device tag
func TestMDMTabletRepository_AddTag(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	device := &models.MDMTablet{
		ID:       "tag-device",
		Number:   "TAG-001",
		Name:     "标签测试设备",
		TenantID: "tag-tenant",
		Status:   string(models.MDMTabletStatusActive),
	}
	if err := repo.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	tag := &models.MDMTabletTag{
		ID:       "tag-001",
		DeviceID: device.ID,
		Tag:      "location",
		Value:    "教室A",
	}
	err := repo.AddTag(tag)
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	tags, err := repo.GetDeviceTags(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceTags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("Tags count mismatch: got %d, want 1", len(tags))
	}
	if tags[0].Tag != "location" {
		t.Errorf("Tag mismatch: got %s, want location", tags[0].Tag)
	}

	// Remove tag
	err = repo.RemoveTag(device.ID, "location")
	if err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}

	tagsAfter, err := repo.GetDeviceTags(device.ID)
	if err != nil {
		t.Fatalf("GetDeviceTags after remove failed: %v", err)
	}
	if len(tagsAfter) != 0 {
		t.Errorf("Tags after remove mismatch: got %d, want 0", len(tagsAfter))
	}
}

// TestMDMTabletRepository_DeviceNotFound tests behavior when device not found
func TestMDMTabletRepository_DeviceNotFound(t *testing.T) {
	repo, db := setupTestDB(t)
	defer db.Close()

	_, err := repo.GetDeviceByID("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent device, got nil")
	}

	_, err = repo.GetDeviceByNumber("non-existent-number")
	if err == nil {
		t.Error("Expected error for non-existent number, got nil")
	}
}
