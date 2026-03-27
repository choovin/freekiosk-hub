package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// PushNotificationService 推送通知服务接口
type PushNotificationService interface {
	// 通知管理
	CreateNotification(tenantID, title, content, notificationType string, priority string, actions []models.PushAction) (*models.PushNotification, error)
	GetNotification(id string) (*models.PushNotification, error)
	UpdateNotification(notification *models.PushNotification) error
	DeleteNotification(id string) error
	ListNotifications(tenantID string, limit, offset int) ([]*models.PushNotification, int64, error)
	ListDeviceNotifications(deviceID string, limit, offset int) ([]*models.PushNotification, int64, error)

	// 发送
	SendToDevice(tenantID, deviceID, title, content, notificationType string) (*models.PushNotification, error)
	SendToGroup(tenantID, groupID, title, content, notificationType string) (*models.PushNotification, error)
	SendToAll(tenantID, title, content, notificationType string) (*models.PushNotification, error)
	ScheduleNotification(tenantID, title, content, notificationType string, scheduledAt int64) (*models.PushNotification, error)

	// 回执
	GetReceipts(notificationID string) ([]*models.PushNotificationReceipt, error)
	RecordDelivery(notificationID, deviceID string) error
	RecordRead(notificationID, deviceID string) error
	RecordFailure(notificationID, deviceID, errorMsg string) error
	GetDeviceReceipts(deviceID string, limit, offset int) ([]*models.PushNotificationReceipt, int64, error)
}

// DefaultPushNotificationService 默认推送通知服务实现
type DefaultPushNotificationService struct {
	repo    PushNotificationRepo
	mqttSvc *MQTTService
}

// PushNotificationRepo 推送通知仓库接口
type PushNotificationRepo interface {
	Create(notification *models.PushNotification) error
	GetByID(id string) (*models.PushNotification, error)
	Update(notification *models.PushNotification) error
	Delete(id string) error
	List(tenantID string, limit, offset int) ([]*models.PushNotification, int64, error)
	ListByDevice(deviceID string, limit, offset int) ([]*models.PushNotification, int64, error)
	ListScheduled(tenantID string) ([]*models.PushNotification, error)
	SaveReceipt(receipt *models.PushNotificationReceipt) error
	GetReceipts(notificationID string) ([]*models.PushNotificationReceipt, error)
	UpdateReceipt(receipt *models.PushNotificationReceipt) error
	GetDeviceReceipts(deviceID string, limit, offset int) ([]*models.PushNotificationReceipt, int64, error)
}

// NewPushNotificationService 创建推送通知服务
func NewPushNotificationService(repo PushNotificationRepo, mqttSvc *MQTTService) *DefaultPushNotificationService {
	return &DefaultPushNotificationService{
		repo:    repo,
		mqttSvc: mqttSvc,
	}
}

// CreateNotification 创建通知
func (s *DefaultPushNotificationService) CreateNotification(tenantID, title, content, notificationType string, priority string, actions []models.PushAction) (*models.PushNotification, error) {
	if priority == "" {
		priority = "normal"
	}
	if notificationType == "" {
		notificationType = "info"
	}

	var actionsJSON string
	if len(actions) > 0 {
		actionsBytes, err := json.Marshal(actions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal actions: %w", err)
		}
		actionsJSON = string(actionsBytes)
	}

	now := time.Now().Unix()
	notification := &models.PushNotification{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Title:     title,
		Content:   content,
		Priority:  priority,
		Type:      notificationType,
		Actions:   actionsJSON,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetNotification 获取通知
func (s *DefaultPushNotificationService) GetNotification(id string) (*models.PushNotification, error) {
	return s.repo.GetByID(id)
}

// UpdateNotification 更新通知
func (s *DefaultPushNotificationService) UpdateNotification(notification *models.PushNotification) error {
	return s.repo.Update(notification)
}

// DeleteNotification 删除通知
func (s *DefaultPushNotificationService) DeleteNotification(id string) error {
	return s.repo.Delete(id)
}

// ListNotifications 获取租户的通知列表
func (s *DefaultPushNotificationService) ListNotifications(tenantID string, limit, offset int) ([]*models.PushNotification, int64, error) {
	return s.repo.List(tenantID, limit, offset)
}

// ListDeviceNotifications 获取设备的通知列表
func (s *DefaultPushNotificationService) ListDeviceNotifications(deviceID string, limit, offset int) ([]*models.PushNotification, int64, error) {
	return s.repo.ListByDevice(deviceID, limit, offset)
}

// SendToDevice 发送通知到设备
func (s *DefaultPushNotificationService) SendToDevice(tenantID, deviceID, title, content, notificationType string) (*models.PushNotification, error) {
	notification, err := s.CreateNotification(tenantID, title, content, notificationType, "normal", nil)
	if err != nil {
		return nil, err
	}

	notification.DeviceID = deviceID
	notification.Status = "sent"
	notification.SentAt = time.Now().Unix()

	if err := s.repo.Update(notification); err != nil {
		return nil, err
	}

	// 通过MQTT发送
	if s.mqttSvc != nil && s.mqttSvc.IsConnected() {
		topic := fmt.Sprintf("kiosk/%s/%s/notification", tenantID, deviceID)
		payloadBytes, _ := json.Marshal(map[string]interface{}{
			"id":      notification.ID,
			"title":   title,
			"content": content,
			"type":    notificationType,
			"sent_at": notification.SentAt,
		})
		if err := s.mqttSvc.Publish(topic, payloadBytes); err != nil {
			// 即使MQTT发送失败，也标记为已发送
			notification.Status = "sent"
		}
	}

	return notification, nil
}

// SendToGroup 发送通知到设备组
func (s *DefaultPushNotificationService) SendToGroup(tenantID, groupID, title, content, notificationType string) (*models.PushNotification, error) {
	notification, err := s.CreateNotification(tenantID, title, content, notificationType, "normal", nil)
	if err != nil {
		return nil, err
	}

	notification.GroupID = groupID
	notification.Status = "sent"
	notification.SentAt = time.Now().Unix()

	if err := s.repo.Update(notification); err != nil {
		return nil, err
	}

	// 组通知通常通过MQTT广播到组内所有设备
	// 具体实现取决于设备分组的管理方式

	return notification, nil
}

// SendToAll 发送通知到所有设备
func (s *DefaultPushNotificationService) SendToAll(tenantID, title, content, notificationType string) (*models.PushNotification, error) {
	notification, err := s.CreateNotification(tenantID, title, content, notificationType, "high", nil)
	if err != nil {
		return nil, err
	}

	notification.Status = "sent"
	notification.SentAt = time.Now().Unix()

	if err := s.repo.Update(notification); err != nil {
		return nil, err
	}

	// 通过MQTT广播
	if s.mqttSvc != nil && s.mqttSvc.IsConnected() {
		topic := fmt.Sprintf("kiosk/%s/broadcast/notification", tenantID)
		payloadBytes, _ := json.Marshal(map[string]interface{}{
			"id":      notification.ID,
			"title":   title,
			"content": content,
			"type":    notificationType,
			"sent_at": notification.SentAt,
		})
		s.mqttSvc.Publish(topic, payloadBytes)
	}

	return notification, nil
}

// ScheduleNotification 计划发送通知
func (s *DefaultPushNotificationService) ScheduleNotification(tenantID, title, content, notificationType string, scheduledAt int64) (*models.PushNotification, error) {
	notification, err := s.CreateNotification(tenantID, title, content, notificationType, "normal", nil)
	if err != nil {
		return nil, err
	}

	notification.ScheduledAt = scheduledAt
	notification.Status = "pending"

	if err := s.repo.Update(notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetReceipts 获取通知的所有回执
func (s *DefaultPushNotificationService) GetReceipts(notificationID string) ([]*models.PushNotificationReceipt, error) {
	return s.repo.GetReceipts(notificationID)
}

// RecordDelivery 记录送达
func (s *DefaultPushNotificationService) RecordDelivery(notificationID, deviceID string) error {
	receipt := &models.PushNotificationReceipt{
		ID:             uuid.New().String(),
		NotificationID: notificationID,
		DeviceID:       deviceID,
		Status:         "delivered",
		DeliveredAt:    time.Now().Unix(),
		CreatedAt:      time.Now().Unix(),
	}
	return s.repo.SaveReceipt(receipt)
}

// RecordRead 记录已读
func (s *DefaultPushNotificationService) RecordRead(notificationID, deviceID string) error {
	receipts, err := s.repo.GetReceipts(notificationID)
	if err != nil {
		return err
	}

	for _, receipt := range receipts {
		if receipt.DeviceID == deviceID {
			receipt.Status = "read"
			receipt.ReadAt = time.Now().Unix()
			return s.repo.UpdateReceipt(receipt)
		}
	}

	// 如果没有找到回执，创建一个新的
	receipt := &models.PushNotificationReceipt{
		ID:             uuid.New().String(),
		NotificationID: notificationID,
		DeviceID:       deviceID,
		Status:         "read",
		ReadAt:         time.Now().Unix(),
		CreatedAt:      time.Now().Unix(),
	}
	return s.repo.SaveReceipt(receipt)
}

// RecordFailure 记录发送失败
func (s *DefaultPushNotificationService) RecordFailure(notificationID, deviceID, errorMsg string) error {
	receipt := &models.PushNotificationReceipt{
		ID:             uuid.New().String(),
		NotificationID:  notificationID,
		DeviceID:       deviceID,
		Status:         "failed",
		ErrorMessage:   errorMsg,
		CreatedAt:      time.Now().Unix(),
	}
	return s.repo.SaveReceipt(receipt)
}

// GetDeviceReceipts 获取设备的回执列表
func (s *DefaultPushNotificationService) GetDeviceReceipts(deviceID string, limit, offset int) ([]*models.PushNotificationReceipt, int64, error) {
	return s.repo.GetDeviceReceipts(deviceID, limit, offset)
}
