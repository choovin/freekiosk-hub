package services

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// GeofenceService 地理围栏服务接口
type GeofenceService interface {
	// 围栏CRUD
	Create(gf *models.Geofence) error
	GetByID(id string) (*models.Geofence, error)
	Update(gf *models.Geofence) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.Geofence, int64, error)
	ListActive(tenantID string) ([]*models.Geofence, error)

	// 设备分配
	AssignToDevice(geofenceID, deviceID, tenantID, assignedBy string) error
	UnassignFromDevice(geofenceID, deviceID string) error
	GetDeviceGeofences(deviceID string) ([]*models.Geofence, error)
	GetAssignedDevices(geofenceID string) ([]string, error)

	// 事件处理
	RecordEvent(geofenceID, deviceID, tenantID, eventType string, lat, lng float64) error
	GetDeviceEvents(deviceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error)
	GetGeofenceEvents(geofenceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error)

	// 位置检查
	CheckDeviceLocation(deviceID string, lat, lng float64) ([]*models.GeofenceEvent, error)
}

// geofenceServiceImpl 地理围栏服务实现
type geofenceServiceImpl struct {
	repo repositories.GeofenceRepository
}

// NewGeofenceService 创建地理围栏服务
func NewGeofenceService(repo repositories.GeofenceRepository) GeofenceService {
	return &geofenceServiceImpl{repo: repo}
}

// Create 创建地理围栏
func (s *geofenceServiceImpl) Create(gf *models.Geofence) error {
	gf.ID = fmt.Sprintf("gf-%s", uuid.New().String()[:8])
	now := time.Now().Unix()
	gf.CreatedAt = now
	gf.UpdatedAt = now
	if gf.FenceType == "" {
		gf.FenceType = "circle"
	}
	if gf.Radius == 0 {
		gf.Radius = 100 // 默认100米
	}
	return s.repo.Create(gf)
}

// GetByID 获取地理围栏
func (s *geofenceServiceImpl) GetByID(id string) (*models.Geofence, error) {
	return s.repo.GetByID(id)
}

// Update 更新地理围栏
func (s *geofenceServiceImpl) Update(gf *models.Geofence) error {
	gf.UpdatedAt = time.Now().Unix()
	return s.repo.Update(gf)
}

// Delete 删除地理围栏
func (s *geofenceServiceImpl) Delete(id string) error {
	return s.repo.Delete(id)
}

// List 获取地理围栏列表
func (s *geofenceServiceImpl) List(tenantID string, limit, offset int) ([]*models.Geofence, int64, error) {
	return s.repo.List(tenantID, limit, offset)
}

// ListActive 获取激活的地理围栏列表
func (s *geofenceServiceImpl) ListActive(tenantID string) ([]*models.Geofence, error) {
	return s.repo.ListActive(tenantID)
}

// AssignToDevice 分配围栏到设备
func (s *geofenceServiceImpl) AssignToDevice(geofenceID, deviceID, tenantID, assignedBy string) error {
	// 验证围栏存在
	gf, err := s.repo.GetByID(geofenceID)
	if err != nil {
		return fmt.Errorf("geofence not found: %w", err)
	}
	if gf == nil {
		return fmt.Errorf("geofence not found")
	}
	if gf.TenantID != tenantID {
		return fmt.Errorf("geofence does not belong to this tenant")
	}
	return s.repo.AssignDevice(geofenceID, deviceID, tenantID, assignedBy)
}

// UnassignFromDevice 取消分配
func (s *geofenceServiceImpl) UnassignFromDevice(geofenceID, deviceID string) error {
	return s.repo.UnassignDevice(geofenceID, deviceID)
}

// GetDeviceGeofences 获取设备的围栏
func (s *geofenceServiceImpl) GetDeviceGeofences(deviceID string) ([]*models.Geofence, error) {
	return s.repo.GetDeviceGeofences(deviceID)
}

// GetAssignedDevices 获取围栏关联的设备
func (s *geofenceServiceImpl) GetAssignedDevices(geofenceID string) ([]string, error) {
	return s.repo.GetAssignedDevices(geofenceID)
}

// RecordEvent 记录围栏事件
func (s *geofenceServiceImpl) RecordEvent(geofenceID, deviceID, tenantID, eventType string, lat, lng float64) error {
	gf, err := s.repo.GetByID(geofenceID)
	if err != nil || gf == nil {
		return fmt.Errorf("geofence not found")
	}

	event := &models.GeofenceEvent{
		ID:          fmt.Sprintf("gfe-%s", uuid.New().String()[:8]),
		GeofenceID: geofenceID,
		DeviceID:   deviceID,
		TenantID:   tenantID,
		EventType:  eventType,
		Latitude:   lat,
		Longitude:  lng,
		GeofenceName: gf.Name,
		Timestamp: time.Now().Unix(),
	}

	return s.repo.RecordEvent(event)
}

// GetDeviceEvents 获取设备围栏事件
func (s *geofenceServiceImpl) GetDeviceEvents(deviceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	return s.repo.GetDeviceEvents(deviceID, limit, offset)
}

// GetGeofenceEvents 获取围栏事件
func (s *geofenceServiceImpl) GetGeofenceEvents(geofenceID string, limit, offset int) ([]*models.GeofenceEvent, int64, error) {
	return s.repo.GetGeofenceEvents(geofenceID, limit, offset)
}

// CheckDeviceLocation 检查设备位置是否在围栏内
// 返回触发的事件列表
func (s *geofenceServiceImpl) CheckDeviceLocation(deviceID string, lat, lng float64) ([]*models.GeofenceEvent, error) {
	geofences, err := s.repo.GetDeviceGeofences(deviceID)
	if err != nil {
		return nil, err
	}

	var events []*models.GeofenceEvent
	now := time.Now().Unix()

	for _, gf := range geofences {
		isInside := isPointInCircle(lat, lng, gf.Latitude, gf.Longitude, gf.Radius)

		// TODO: 需要维护设备上一次的位置状态来判断是进入还是离开
		// 这里简化处理，假设设备位置变化时会调用此方法

		if isInside {
			if gf.AlertOnEnter {
				event := &models.GeofenceEvent{
					ID:           fmt.Sprintf("gfe-%s", uuid.New().String()[:8]),
					GeofenceID:  gf.ID,
					DeviceID:    deviceID,
					TenantID:    gf.TenantID,
					EventType:   "enter",
					Latitude:    lat,
					Longitude:   lng,
					GeofenceName: gf.Name,
					Timestamp:   now,
				}
				events = append(events, event)
				s.repo.RecordEvent(event)
			}
		}
	}

	return events, nil
}

// isPointInCircle 判断点是否在圆形围栏内
func isPointInCircle(lat, lng, centerLat, centerLng, radius float64) bool {
	// 使用Haversine公式计算两点之间的距离
	const earthRadius = 6371000 // 地球半径，单位：米

	lat1Rad := lat * math.Pi / 180
	lat2Rad := centerLat * math.Pi / 180
	deltaLat := (centerLat - lat) * math.Pi / 180
	deltaLng := (centerLng - lng) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadius * c

	return distance <= radius
}
