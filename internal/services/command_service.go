package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/mqtt"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// CommandService 命令服务接口
type CommandService interface {
	// 单设备命令
	SendCommand(ctx context.Context, tenantID, deviceID string, cmd *models.Command) (*models.CommandResult, error)

	// 批量命令
	SendBatchCommand(ctx context.Context, tenantID string, target *models.CommandTarget, cmd *models.Command) (*models.BatchCommandResult, error)

	// 命令历史
	GetCommandHistory(ctx context.Context, tenantID, deviceID string, limit, offset int) ([]*models.CommandRecord, int64, error)
	GetCommandByID(ctx context.Context, commandID string) (*models.CommandRecord, error)

	// 取消命令
	CancelCommand(ctx context.Context, commandID string) error
}

// CommandServiceConfig 命令服务配置
type CommandServiceConfig struct {
	DefaultTimeout time.Duration
	MaxConcurrent  int
	RetryCount     int
	RetryDelay     time.Duration
}

type commandService struct {
	mqttClient   *mqtt.Client
	deviceRepo   repositories.DeviceRepository
	commandRepo  CommandRecordRepository
	statusSvc    DeviceStatusService
	config       CommandServiceConfig

	// 待处理命令管理
	pendingCommands map[string]chan *models.CommandResult
	mu              sync.RWMutex
}

// CommandRecordRepository 命令记录仓储接口
type CommandRecordRepository interface {
	Create(ctx context.Context, record *models.CommandRecord) error
	Update(ctx context.Context, record *models.CommandRecord) error
	GetByID(ctx context.Context, id string) (*models.CommandRecord, error)
	GetByCommandID(ctx context.Context, commandID string) (*models.CommandRecord, error)
	ListByDevice(ctx context.Context, tenantID, deviceID string, limit, offset int) ([]*models.CommandRecord, int64, error)
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*models.CommandRecord, int64, error)
	DeleteOldRecords(ctx context.Context, before time.Time) (int64, error)
}

// NewCommandService 创建命令服务
func NewCommandService(
	mqttClient *mqtt.Client,
	deviceRepo repositories.DeviceRepository,
	commandRepo CommandRecordRepository,
	statusSvc DeviceStatusService,
	config CommandServiceConfig,
) CommandService {
	return &commandService{
		mqttClient:      mqttClient,
		deviceRepo:      deviceRepo,
		commandRepo:     commandRepo,
		statusSvc:       statusSvc,
		config:          config,
		pendingCommands: make(map[string]chan *models.CommandResult),
	}
}

// SendCommand 发送命令到单个设备
func (s *commandService) SendCommand(ctx context.Context, tenantID, deviceID string, cmd *models.Command) (*models.CommandResult, error) {
	// 1. 验证设备在线状态
	online, err := s.statusSvc.IsDeviceOnline(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check device status: %w", err)
	}
	if !online {
		return nil, fmt.Errorf("device %s is offline", deviceID)
	}

	// 2. 生成命令 ID
	if cmd.ID == "" {
		cmd.ID = uuid.New().String()
	}
	cmd.Timestamp = time.Now()

	// 设置默认超时
	timeout := s.config.DefaultTimeout
	if cmd.Timeout > 0 {
		timeout = time.Duration(cmd.Timeout) * time.Second
	}

	// 3. 创建命令记录
	record := &models.CommandRecord{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		DeviceID:    deviceID,
		CommandType: cmd.Type,
		CommandID:   cmd.ID,
		Payload:     cmd.Params,
		Status:      string(models.CommandStatusPending),
		CreatedAt:   time.Now(),
	}

	if err := s.commandRepo.Create(ctx, record); err != nil {
		slog.Error("Failed to create command record", "error", err)
	}

	// 4. 创建响应通道
	respChan := make(chan *models.CommandResult, 1)
	s.mu.Lock()
	s.pendingCommands[cmd.ID] = respChan
	s.mu.Unlock()

	// 清理
	defer func() {
		s.mu.Lock()
		delete(s.pendingCommands, cmd.ID)
		s.mu.Unlock()
		close(respChan)
	}()

	// 5. 发布命令到 MQTT
	topicBuilder := mqtt.NewTopicBuilder(tenantID, deviceID)
	payload, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := s.mqttClient.Publish(ctx, topicBuilder.CommandTopic(), payload); err != nil {
		// 更新命令状态为失败
		record.Status = string(models.CommandStatusFailed)
		record.ErrorMessage = err.Error()
		s.commandRepo.Update(ctx, record)

		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	slog.Info("📤 Command sent",
		"commandId", cmd.ID,
		"deviceId", deviceID,
		"type", cmd.Type,
	)

	// 6. 等待响应
	select {
	case result := <-respChan:
		// 更新命令记录
		record.Status = string(models.CommandStatusSuccess)
		if !result.Success {
			record.Status = string(models.CommandStatusFailed)
		}
		now := time.Now()
		record.CompletedAt = &now
		record.Duration = now.UnixMilli() - cmd.Timestamp.UnixMilli()
		if result.Result != nil {
			record.Result = result.Result
		}
		if result.Error != "" {
			record.ErrorMessage = result.Error
		}
		s.commandRepo.Update(ctx, record)

		return result, nil

	case <-time.After(timeout):
		// 超时
		record.Status = string(models.CommandStatusTimeout)
		now := time.Now()
		record.CompletedAt = &now
		record.ErrorMessage = "command timeout"
		s.commandRepo.Update(ctx, record)

		return nil, fmt.Errorf("command timeout after %v", timeout)

	case <-ctx.Done():
		// 上下文取消
		record.Status = string(models.CommandStatusCanceled)
		now := time.Now()
		record.CompletedAt = &now
		record.ErrorMessage = ctx.Err().Error()
		s.commandRepo.Update(ctx, record)

		return nil, ctx.Err()
	}
}

// SendBatchCommand 发送批量命令
func (s *commandService) SendBatchCommand(ctx context.Context, tenantID string, target *models.CommandTarget, cmd *models.Command) (*models.BatchCommandResult, error) {
	// 1. 获取目标设备列表
	deviceIDs := target.DeviceIDs

	if len(target.GroupIDs) > 0 {
		// 从分组获取设备
		// TODO: 实现分组设备查询
	}

	if target.All {
		// 获取租户下所有设备
		// TODO: 实现全部设备查询
	}

	if len(deviceIDs) == 0 {
		return nil, fmt.Errorf("no target devices specified")
	}

	// 2. 创建批量结果
	batchID := uuid.New().String()
	result := &models.BatchCommandResult{
		BatchID:     batchID,
		TotalCount:  len(deviceIDs),
		Results:     make([]models.DeviceCommandResult, 0, len(deviceIDs)),
		CreatedAt:   time.Now(),
	}

	// 3. 并发发送命令
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, deviceID := range deviceIDs {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()

			// 复制命令
			deviceCmd := &models.Command{
				ID:        uuid.New().String(),
				Type:      cmd.Type,
				Timestamp: time.Now(),
				Params:    cmd.Params,
				Timeout:   cmd.Timeout,
			}

			cmdResult, err := s.SendCommand(ctx, tenantID, did, deviceCmd)

			mu.Lock()
			defer mu.Unlock()

			deviceResult := models.DeviceCommandResult{
				DeviceID: did,
			}

			if err != nil {
				deviceResult.Success = false
				deviceResult.Error = err.Error()
				result.FailedCount++
			} else {
				deviceResult.Success = cmdResult.Success
				deviceResult.Result = cmdResult
				if cmdResult.Success {
					result.SuccessCount++
				} else {
					result.FailedCount++
				}
			}

			result.Results = append(result.Results, deviceResult)
		}(deviceID)
	}

	wg.Wait()

	slog.Info("📤 Batch command completed",
		"batchId", batchID,
		"total", result.TotalCount,
		"success", result.SuccessCount,
		"failed", result.FailedCount,
	)

	return result, nil
}

// GetCommandHistory 获取命令历史
func (s *commandService) GetCommandHistory(ctx context.Context, tenantID, deviceID string, limit, offset int) ([]*models.CommandRecord, int64, error) {
	return s.commandRepo.ListByDevice(ctx, tenantID, deviceID, limit, offset)
}

// GetCommandByID 根据 ID 获取命令
func (s *commandService) GetCommandByID(ctx context.Context, commandID string) (*models.CommandRecord, error) {
	return s.commandRepo.GetByCommandID(ctx, commandID)
}

// CancelCommand 取消命令
func (s *commandService) CancelCommand(ctx context.Context, commandID string) error {
	record, err := s.commandRepo.GetByCommandID(ctx, commandID)
	if err != nil {
		return err
	}

	if record.Status != string(models.CommandStatusPending) {
		return fmt.Errorf("cannot cancel command in status: %s", record.Status)
	}

	record.Status = string(models.CommandStatusCanceled)
	now := time.Now()
	record.CompletedAt = &now
	record.ErrorMessage = "canceled by user"

	return s.commandRepo.Update(ctx, record)
}

// HandleCommandResponse 处理命令响应
func (s *commandService) HandleCommandResponse(commandID string, result *models.CommandResult) {
	s.mu.RLock()
	ch, ok := s.pendingCommands[commandID]
	s.mu.RUnlock()

	if ok {
		select {
		case ch <- result:
		default:
			slog.Warn("Command response channel full", "commandId", commandID)
		}
	}
}
