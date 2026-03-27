package services

import (
	"testing"

	"github.com/wared2003/freekiosk-hub/internal/models"
)

// MockRemoteControlRepository 测试用的模拟远程控制仓库
type MockRemoteControlRepository struct {
	sessions  map[string]*models.RemoteSession
	events    []*models.RemoteSessionEvent
	captures  []*models.ScreenCapture
	commands  []*models.RemoteCommand
}

func NewMockRemoteControlRepository() *MockRemoteControlRepository {
	return &MockRemoteControlRepository{
		sessions: make(map[string]*models.RemoteSession),
		events:   []*models.RemoteSessionEvent{},
		captures: []*models.ScreenCapture{},
		commands: []*models.RemoteCommand{},
	}
}

func (m *MockRemoteControlRepository) InitSchema(ctx interface{}) error {
	return nil
}

func (m *MockRemoteControlRepository) CreateSession(session *models.RemoteSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockRemoteControlRepository) GetSession(id string) (*models.RemoteSession, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, nil
	}
	return session, nil
}

func (m *MockRemoteControlRepository) UpdateSession(session *models.RemoteSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockRemoteControlRepository) DeleteSession(id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *MockRemoteControlRepository) ListDeviceSessions(deviceID string, limit, offset int) ([]*models.RemoteSession, int64, error) {
	var result []*models.RemoteSession
	for _, session := range m.sessions {
		if session.DeviceID == deviceID {
			result = append(result, session)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockRemoteControlRepository) GetActiveSession(deviceID string) (*models.RemoteSession, error) {
	for _, session := range m.sessions {
		if session.DeviceID == deviceID && session.Status == "active" {
			return session, nil
		}
	}
	return nil, nil
}

func (m *MockRemoteControlRepository) RecordEvent(event *models.RemoteSessionEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockRemoteControlRepository) GetSessionEvents(sessionID string) ([]*models.RemoteSessionEvent, error) {
	var result []*models.RemoteSessionEvent
	for _, e := range m.events {
		if e.SessionID == sessionID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockRemoteControlRepository) SaveScreenCapture(capture *models.ScreenCapture) error {
	m.captures = append(m.captures, capture)
	return nil
}

func (m *MockRemoteControlRepository) GetSessionScreenCaptures(sessionID string) ([]*models.ScreenCapture, error) {
	var result []*models.ScreenCapture
	for _, c := range m.captures {
		if c.SessionID == sessionID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MockRemoteControlRepository) SaveCommand(cmd *models.RemoteCommand) error {
	m.commands = append(m.commands, cmd)
	return nil
}

func (m *MockRemoteControlRepository) UpdateCommandStatus(id, status, response string) error {
	for _, cmd := range m.commands {
		if cmd.ID == id {
			cmd.Status = status
			cmd.Response = response
			break
		}
	}
	return nil
}

func (m *MockRemoteControlRepository) GetSessionCommands(sessionID string) ([]*models.RemoteCommand, error) {
	var result []*models.RemoteCommand
	for _, cmd := range m.commands {
		if cmd.SessionID == sessionID {
			result = append(result, cmd)
		}
	}
	return result, nil
}

func TestRemoteControlService_CreateSession(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	session, err := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", []map[string]interface{}{
		{"urls": []string{"stun:stun.example.com"}},
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should be set")
	}

	if session.DeviceID != "device-1" {
		t.Errorf("Expected device_id device-1, got %s", session.DeviceID)
	}

	if session.Status != "pending" {
		t.Errorf("Expected status pending, got %s", session.Status)
	}

	if session.SessionType != "view" {
		t.Errorf("Expected session_type view, got %s", session.SessionType)
	}
}

func TestRemoteControlService_GetSession(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session first
	created, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "control", nil)

	// Get the session
	retrieved, err := svc.GetSession(created.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved session should not be nil")
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestRemoteControlService_UpdateSession(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)

	// Update to active
	err := svc.UpdateSession(session.ID, "active")
	if err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	// Get and verify
	updated, _ := svc.GetSession(session.ID)
	if updated.Status != "active" {
		t.Errorf("Expected status active, got %s", updated.Status)
	}

	if updated.StartedAt == 0 {
		t.Error("StartedAt should be set when status becomes active")
	}
}

func TestRemoteControlService_DeleteSession(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)

	// Delete it
	err := svc.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Try to get it
	retrieved, _ := svc.GetSession(session.ID)
	if retrieved != nil {
		t.Error("Session should be nil after delete")
	}
}

func TestRemoteControlService_RecordEvent(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)

	// Record an event
	err := svc.RecordEvent(session.ID, "device-1", "tenant-1", "start", "Remote session started")
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}

	// Get events
	events, err := svc.GetSessionEvents(session.ID)
	if err != nil {
		t.Fatalf("GetSessionEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].EventType != "start" {
		t.Errorf("Expected event type start, got %s", events[0].EventType)
	}
}

func TestRemoteControlService_SaveScreenCapture(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)

	// Save a screen capture
	capture, err := svc.SaveScreenCapture(session.ID, "device-1", "tenant-1", "/captures/screen1.png", 102400)
	if err != nil {
		t.Fatalf("SaveScreenCapture failed: %v", err)
	}

	if capture.ID == "" {
		t.Error("Capture ID should be set")
	}

	if capture.FilePath != "/captures/screen1.png" {
		t.Errorf("Expected file path /captures/screen1.png, got %s", capture.FilePath)
	}

	// Get captures
	captures, err := svc.GetSessionScreenCaptures(session.ID)
	if err != nil {
		t.Fatalf("GetSessionScreenCaptures failed: %v", err)
	}

	if len(captures) != 1 {
		t.Errorf("Expected 1 capture, got %d", len(captures))
	}
}

func TestRemoteControlService_SendCommand(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "control", nil)

	// Send a command
	params := map[string]interface{}{
		"action": "tap",
		"x":     100,
		"y":     200,
	}
	cmd, err := svc.SendCommand(session.ID, "device-1", "tenant-1", "input", params)
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	if cmd.ID == "" {
		t.Error("Command ID should be set")
	}

	if cmd.Status != "pending" {
		t.Errorf("Expected status pending, got %s", cmd.Status)
	}

	// Update command status
	err = svc.UpdateCommandStatus(cmd.ID, "executed", "Command executed successfully")
	if err != nil {
		t.Fatalf("UpdateCommandStatus failed: %v", err)
	}

	// Get commands
	commands, err := svc.GetSessionCommands(session.ID)
	if err != nil {
		t.Fatalf("GetSessionCommands failed: %v", err)
	}

	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	if commands[0].Status != "executed" {
		t.Errorf("Expected status executed, got %s", commands[0].Status)
	}
}

func TestRemoteControlService_GetActiveSession(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create a session
	session, _ := svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)

	// Initially no active session
	active, _ := svc.GetActiveSession("device-1")
	if active != nil {
		t.Error("Should have no active session initially")
	}

	// Activate the session
	svc.UpdateSession(session.ID, "active")

	// Now should have active session
	active, _ = svc.GetActiveSession("device-1")
	if active == nil {
		t.Fatal("Should have active session after activation")
	}

	if active.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, active.ID)
	}
}

func TestRemoteControlService_ListDeviceSessions(t *testing.T) {
	repo := NewMockRemoteControlRepository()
	svc := NewRemoteControlService(repo)

	// Create multiple sessions for the same device
	svc.CreateSession("device-1", "tenant-1", "admin-1", "view", nil)
	svc.CreateSession("device-1", "tenant-1", "admin-1", "control", nil)
	svc.CreateSession("device-2", "tenant-1", "admin-1", "view", nil)

	// List sessions for device-1
	sessions, total, err := svc.ListDeviceSessions("device-1", 10, 0)
	if err != nil {
		t.Fatalf("ListDeviceSessions failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 sessions for device-1, got %d", total)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions returned, got %d", len(sessions))
	}
}
