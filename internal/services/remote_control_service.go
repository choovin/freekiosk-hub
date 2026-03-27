package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

// RemoteControlService 远程控制服务接口
type RemoteControlService interface {
	// 会话管理
	CreateSession(deviceID, tenantID, initiatorID, sessionType string, iceServers []map[string]interface{}) (*models.RemoteSession, error)
	GetSession(sessionID string) (*models.RemoteSession, error)
	UpdateSession(sessionID, status string) error
	DeleteSession(sessionID string) error
	ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error)
	GetActiveSession(deviceID string) (*models.RemoteSession, error)

	// 事件记录
	RecordEvent(sessionID, deviceID, tenantID, eventType, message string) error
	GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error)

	// 屏幕截图
	SaveScreenCapture(sessionID, deviceID, tenantID, filePath string, fileSize int64) (*models.ScreenCapture, error)
	GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error)

	// 命令管理
	SendCommand(sessionID, deviceID, tenantID, commandType string, params map[string]interface{}) (*models.RemoteCommand, error)
	UpdateCommandStatus(cmdID, status, response string) error
	GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error)
}

// DefaultRemoteControlService 默认远程控制服务实现
type DefaultRemoteControlService struct {
	repo RemoteControlRepo
}

// RemoteControlRepo 远程控制仓库接口（供服务内部使用）
type RemoteControlRepo interface {
	CreateSession(session *models.RemoteSession) error
	GetSession(id string) (*models.RemoteSession, error)
	UpdateSession(session *models.RemoteSession) error
	DeleteSession(id string) error
	ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error)
	GetActiveSession(deviceID string) (*models.RemoteSession, error)
	RecordEvent(event *models.RemoteSessionEvent) error
	GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error)
	SaveScreenCapture(capture *models.ScreenCapture) error
	GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error)
	SaveCommand(cmd *models.RemoteCommand) error
	UpdateCommandStatus(id, status, response string) error
	GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error)
}

// NewRemoteControlService 创建远程控制服务
func NewRemoteControlService(repo RemoteControlRepo) *DefaultRemoteControlService {
	return &DefaultRemoteControlService{repo: repo}
}

// CreateSession 创建远程会话
func (s *DefaultRemoteControlService) CreateSession(deviceID, tenantID, initiatorID, sessionType string, iceServers []map[string]interface{}) (*models.RemoteSession, error) {
	iceServersJSON, err := json.Marshal(iceServers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ICE servers: %w", err)
	}

	now := time.Now().Unix()
	session := &models.RemoteSession{
		ID:          uuid.New().String(),
		DeviceID:    deviceID,
		TenantID:    tenantID,
		InitiatorID: initiatorID,
		Status:      "pending",
		SessionType: sessionType,
		ICEServers:  string(iceServersJSON),
		StartedAt:   0,
		EndedAt:     nil,
		ExpiresAt:   now + 3600, // 1小时后过期
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession 获取会话
func (s *DefaultRemoteControlService) GetSession(sessionID string) (*models.RemoteSession, error) {
	return s.repo.GetSession(sessionID)
}

// UpdateSession 更新会话状态
func (s *DefaultRemoteControlService) UpdateSession(sessionID, status string) error {
	session, err := s.repo.GetSession(sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	now := time.Now().Unix()
	session.Status = status
	session.UpdatedAt = now

	if status == "active" && session.StartedAt == 0 {
		session.StartedAt = now
	} else if status == "ended" || status == "expired" {
		session.EndedAt = &now
	}

	return s.repo.UpdateSession(session)
}

// DeleteSession 删除会话
func (s *DefaultRemoteControlService) DeleteSession(sessionID string) error {
	return s.repo.DeleteSession(sessionID)
}

// ListDeviceSessions 获取设备的会话列表
func (s *DefaultRemoteControlService) ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error) {
	return s.repo.ListDeviceSessions(deviceID, limit, offset)
}

// GetActiveSession 获取设备的活跃会话
func (s *DefaultRemoteControlService) GetActiveSession(deviceID string) (*models.RemoteSession, error) {
	return s.repo.GetActiveSession(deviceID)
}

// RecordEvent 记录会话事件
func (s *DefaultRemoteControlService) RecordEvent(sessionID, deviceID, tenantID, eventType, message string) error {
	event := &models.RemoteSessionEvent{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		DeviceID:  deviceID,
		TenantID:  tenantID,
		EventType: eventType,
		Message:   message,
		Timestamp: time.Now().Unix(),
		CreatedAt: time.Now().Unix(),
	}
	return s.repo.RecordEvent(event)
}

// GetSessionEvents 获取会话的所有事件
func (s *DefaultRemoteControlService) GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error) {
	return s.repo.GetSessionEvents(sessionID)
}

// SaveScreenCapture 保存屏幕截图
func (s *DefaultRemoteControlService) SaveScreenCapture(sessionID, deviceID, tenantID, filePath string, fileSize int64) (*models.ScreenCapture, error) {
	capture := &models.ScreenCapture{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		DeviceID:  deviceID,
		TenantID:  tenantID,
		FilePath:  filePath,
		FileSize:  fileSize,
		MimeType:  "image/png",
		CapturedAt: time.Now().Unix(),
		CreatedAt: time.Now().Unix(),
	}
	if err := s.repo.SaveScreenCapture(capture); err != nil {
		return nil, err
	}
	return capture, nil
}

// GetSessionScreenCaptures 获取会话的屏幕截图
func (s *DefaultRemoteControlService) GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error) {
	return s.repo.GetSessionScreenCaptures(sessionID)
}

// SendCommand 发送远程命令
func (s *DefaultRemoteControlService) SendCommand(sessionID, deviceID, tenantID, commandType string, params map[string]interface{}) (*models.RemoteCommand, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	cmd := &models.RemoteCommand{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		DeviceID:    deviceID,
		TenantID:    tenantID,
		CommandType: commandType,
		Params:      string(paramsJSON),
		Status:      "pending",
		Response:    "",
		Timestamp:   time.Now().Unix(),
		CreatedAt:   time.Now().Unix(),
	}

	if err := s.repo.SaveCommand(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// UpdateCommandStatus 更新命令状态
func (s *DefaultRemoteControlService) UpdateCommandStatus(cmdID, status, response string) error {
	return s.repo.UpdateCommandStatus(cmdID, status, response)
}

// GetSessionCommands 获取会话的所有命令
func (s *DefaultRemoteControlService) GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error) {
	return s.repo.GetSessionCommands(sessionID)
}
