package dtos

import (
	"time"

	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// TabletDisplay 组合平板数据用于显示
type TabletDisplay struct {
	repositories.Tablet
	LastReport *repositories.TabletReport
	Groups     []repositories.Group
}

// DeviceStatusDisplay 设备状态显示数据
type DeviceStatusDisplay struct {
	DeviceID    string                 `json:"device_id"`
	Name        string                 `json:"name,omitempty"`
	Online      bool                   `json:"online"`
	LastSeen    time.Time              `json:"last_seen,omitempty"`
	Status      map[string]interface{} `json:"status,omitempty"`
	Battery     int                    `json:"battery,omitempty"`
	ScreenOn    bool                   `json:"screen_on,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
}
