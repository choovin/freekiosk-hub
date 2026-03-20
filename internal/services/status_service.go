package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// DeviceStatusInfo 设备状态信息
type DeviceStatusInfo struct {
	DeviceID     string    `json:"device_id"`
	Online       bool      `json:"online"`
	LastSeen     time.Time `json:"last_seen"`
	BatteryLevel int       `json:"battery_level"`
	ScreenOn     bool      `json:"screen_on"`
}

// DeviceStatusService 设备状态服务接口
type DeviceStatusService interface {
	// 状态查询
	IsDeviceOnline(ctx context.Context, deviceID string) (bool, error)
	GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatusInfo, error)
	GetAllDeviceStatuses(ctx context.Context, tenantID string) ([]*DeviceStatusInfo, error)

	// 状态更新
	UpdateDeviceStatus(ctx context.Context, deviceID string, status *DeviceStatusInfo) error
	SetDeviceOnline(ctx context.Context, deviceID string, online bool) error

	// 状态订阅
	SubscribeStatusChanges(ctx context.Context) (<-chan *StatusChangeEvent, error)
}

// StatusChangeEvent 状态变更事件
type StatusChangeEvent struct {
	DeviceID  string          `json:"device_id"`
	OldStatus *DeviceStatusInfo `json:"old_status"`
	NewStatus *DeviceStatusInfo `json:"new_status"`
	Timestamp time.Time       `json:"timestamp"`
}

type deviceStatusService struct {
	deviceRepo repositories.DeviceRepository
	statusCache sync.Map // 内存状态缓存

	// 状态变更通道
	statusChannels []chan *StatusChangeEvent
	mu             sync.RWMutex
}

// NewDeviceStatusService 创建设备状态服务
func NewDeviceStatusService(deviceRepo repositories.DeviceRepository) DeviceStatusService {
	return &deviceStatusService{
		deviceRepo:     deviceRepo,
		statusChannels: make([]chan *StatusChangeEvent, 0),
	}
}

// IsDeviceOnline 检查设备是否在线
func (s *deviceStatusService) IsDeviceOnline(ctx context.Context, deviceID string) (bool, error) {
	// 先检查内存缓存
	if status, ok := s.statusCache.Load(deviceID); ok {
		info := status.(*DeviceStatusInfo)
		return info.Online, nil
	}

	// 从数据库获取
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return false, err
	}

	// 检查 last_seen_at
	if device.LastSeenAt == nil {
		return false, nil
	}

	// 如果最后可见时间在 5 分钟内，认为在线
	online := time.Since(*device.LastSeenAt) < 5*time.Minute

	// 更新缓存
	s.statusCache.Store(deviceID, &DeviceStatusInfo{
		DeviceID: deviceID,
		Online:   online,
		LastSeen: *device.LastSeenAt,
	})

	return online, nil
}

// GetDeviceStatus 获取设备状态
func (s *deviceStatusService) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatusInfo, error) {
	// 先检查内存缓存
	if status, ok := s.statusCache.Load(deviceID); ok {
		return status.(*DeviceStatusInfo), nil
	}

	// 从数据库获取
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	info := &DeviceStatusInfo{
		DeviceID: deviceID,
	}

	if device.LastSeenAt != nil {
		info.LastSeen = *device.LastSeenAt
		info.Online = time.Since(*device.LastSeenAt) < 5*time.Minute
	}

	// 从 device_info 提取更多信息
	if device.DeviceInfo != nil {
		if battery, ok := device.DeviceInfo["battery_level"].(float64); ok {
			info.BatteryLevel = int(battery)
		}
		if screenOn, ok := device.DeviceInfo["screen_on"].(bool); ok {
			info.ScreenOn = screenOn
		}
	}

	// 更新缓存
	s.statusCache.Store(deviceID, info)

	return info, nil
}

// GetAllDeviceStatuses 获取租户下所有设备状态
func (s *deviceStatusService) GetAllDeviceStatuses(ctx context.Context, tenantID string) ([]*DeviceStatusInfo, error) {
	devices, _, err := s.deviceRepo.List(ctx, tenantID, "", 1000, 0)
	if err != nil {
		return nil, err
	}

	statuses := make([]*DeviceStatusInfo, 0, len(devices))
	for _, device := range devices {
		info := &DeviceStatusInfo{
			DeviceID: device.ID,
		}

		if device.LastSeenAt != nil {
			info.LastSeen = *device.LastSeenAt
			info.Online = time.Since(*device.LastSeenAt) < 5*time.Minute
		}

		if device.DeviceInfo != nil {
			if battery, ok := device.DeviceInfo["battery_level"].(float64); ok {
				info.BatteryLevel = int(battery)
			}
			if screenOn, ok := device.DeviceInfo["screen_on"].(bool); ok {
				info.ScreenOn = screenOn
			}
		}

		statuses = append(statuses, info)
		s.statusCache.Store(device.ID, info)
	}

	return statuses, nil
}

// UpdateDeviceStatus 更新设备状态
func (s *deviceStatusService) UpdateDeviceStatus(ctx context.Context, deviceID string, status *DeviceStatusInfo) error {
	// 获取旧状态
	oldStatus, _ := s.statusCache.Load(deviceID)

	// 更新缓存
	s.statusCache.Store(deviceID, status)

	// 发送状态变更通知
	if oldStatus != nil {
		old := oldStatus.(*DeviceStatusInfo)
		if old.Online != status.Online {
			s.notifyStatusChange(&StatusChangeEvent{
				DeviceID:  deviceID,
				OldStatus: old,
				NewStatus: status,
				Timestamp: time.Now(),
			})

			// 记录日志
			if status.Online {
				slog.Info("📱 Device online", "deviceId", deviceID)
			} else {
				slog.Info("📱 Device offline", "deviceId", deviceID)
			}
		}
	}

	return nil
}

// SetDeviceOnline 设置设备在线状态
func (s *deviceStatusService) SetDeviceOnline(ctx context.Context, deviceID string, online bool) error {
	status, err := s.GetDeviceStatus(ctx, deviceID)
	if err != nil {
		status = &DeviceStatusInfo{
			DeviceID: deviceID,
		}
	}

	status.Online = online
	status.LastSeen = time.Now()

	return s.UpdateDeviceStatus(ctx, deviceID, status)
}

// SubscribeStatusChanges 订阅状态变更
func (s *deviceStatusService) SubscribeStatusChanges(ctx context.Context) (<-chan *StatusChangeEvent, error) {
	ch := make(chan *StatusChangeEvent, 100)

	s.mu.Lock()
	s.statusChannels = append(s.statusChannels, ch)
	s.mu.Unlock()

	// 当上下文取消时，移除订阅
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		for i, c := range s.statusChannels {
			if c == ch {
				s.statusChannels = append(s.statusChannels[:i], s.statusChannels[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

// notifyStatusChange 通知状态变更
func (s *deviceStatusService) notifyStatusChange(event *StatusChangeEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.statusChannels {
		select {
		case ch <- event:
		default:
			// 通道已满，跳过
		}
	}
}

// HandleStatusMessage 处理 MQTT 状态消息
func (s *deviceStatusService) HandleStatusMessage(deviceID string, payload []byte) error {
	var statusMsg struct {
		DeviceID       string `json:"deviceId"`
		TenantID       string `json:"tenantId"`
		BatteryLevel   int    `json:"battery_level,omitempty"`
		BatteryCharging bool   `json:"battery_charging,omitempty"`
		ScreenOn       bool   `json:"screen_on,omitempty"`
		ScreenBrightness int   `json:"screen_brightness,omitempty"`
		Volume         int    `json:"volume,omitempty"`
		WifiSSID       string `json:"wifi_ssid,omitempty"`
		WifiSignalStrength int `json:"wifi_signal_strength,omitempty"`
		CPUUsage       float64 `json:"cpu_usage,omitempty"`
		MemoryUsage    float64 `json:"memory_usage,omitempty"`
		StorageUsage   float64 `json:"storage_usage,omitempty"`
		Temperature    float64 `json:"temperature,omitempty"`
		Uptime         int64   `json:"uptime_seconds,omitempty"`
		CurrentApp     string `json:"current_app,omitempty"`
		CurrentURL     string `json:"current_url,omitempty"`
	}

	if err := json.Unmarshal(payload, &statusMsg); err != nil {
		return fmt.Errorf("failed to parse status message: %w", err)
	}

	// 更新状态
	info := &DeviceStatusInfo{
		DeviceID:     deviceID,
		Online:       true,
		LastSeen:     time.Now(),
		BatteryLevel: statusMsg.BatteryLevel,
		ScreenOn:     statusMsg.ScreenOn,
	}

	return s.UpdateDeviceStatus(context.Background(), deviceID, info)
}
