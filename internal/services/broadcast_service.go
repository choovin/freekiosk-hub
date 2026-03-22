package services

import (
	"encoding/json"
	"log/slog"

	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// BroadcastService handles broadcast message delivery
type BroadcastService struct {
	repo *repositories.FieldTripRepository
	mqtt *MQTTService
}

// NewBroadcastService creates a new BroadcastService
func NewBroadcastService(repo *repositories.FieldTripRepository, mqtt *MQTTService) *BroadcastService {
	return &BroadcastService{
		repo: repo,
		mqtt: mqtt,
	}
}

// SendToGroup sends a broadcast message to all devices in a group
func (s *BroadcastService) SendToGroup(groupID, message, sound string) error {
	if groupID == "" {
		slog.Warn("SendToGroup called with empty groupID")
		return nil
	}

	// Get devices in the group
	devices, err := s.repo.ListDevicesByGroup(groupID)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		slog.Info("No devices in group for broadcast", "group_id", groupID)
		return nil
	}

	// Prepare broadcast payload
	payload := map[string]interface{}{
		"type":    "broadcast",
		"message": message,
		"sound":   sound,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Publish to MQTT for each device
	delivered := 0
	failed := 0
	for _, device := range devices {
		if s.mqtt != nil {
			topic := "freekiosk/fieldtrip/" + device.ID + "/broadcast"
			if err := s.mqtt.Publish(topic, payloadBytes); err != nil {
				slog.Warn("Failed to publish broadcast to device", "device_id", device.ID, "error", err)
				failed++
			} else {
				delivered++
			}
		}
	}

	slog.Info("Broadcast sent to group", "group_id", groupID, "delivered", delivered, "failed", failed)
	return nil
}

// SendToAll sends a broadcast message to all active devices
func (s *BroadcastService) SendToAll(message, sound string) error {
	// Get all active devices
	devices, err := s.repo.ListDevices()
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		slog.Info("No devices for broadcast")
		return nil
	}

	// Prepare broadcast payload
	payload := map[string]interface{}{
		"type":    "broadcast",
		"message": message,
		"sound":   sound,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Publish to MQTT for each active device
	delivered := 0
	failed := 0
	for _, device := range devices {
		if device.Status != "active" {
			continue
		}
		if s.mqtt != nil {
			topic := "freekiosk/fieldtrip/" + device.ID + "/broadcast"
			if err := s.mqtt.Publish(topic, payloadBytes); err != nil {
				slog.Warn("Failed to publish broadcast to device", "device_id", device.ID, "error", err)
				failed++
			} else {
				delivered++
			}
		}
	}

	slog.Info("Broadcast sent to all devices", "delivered", delivered, "failed", failed)
	return nil
}
