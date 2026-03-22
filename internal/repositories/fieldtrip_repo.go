package repositories

import (
	"github.com/jmoiron/sqlx"
	"github.com/wared2003/freekiosk-hub/internal/models"
)

type FieldTripRepository struct {
	db     *sqlx.DB
	hubURL string
}

func NewFieldTripRepository(db *sqlx.DB, hubURL string) *FieldTripRepository {
	return &FieldTripRepository{db: db, hubURL: hubURL}
}

func (r *FieldTripRepository) InitSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS fieldtrip_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			group_key TEXT UNIQUE NOT NULL,
			broadcast_sound TEXT DEFAULT 'default',
			update_policy TEXT DEFAULT 'manual',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fieldtrip_devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			group_id TEXT REFERENCES fieldtrip_groups(id),
			api_key_hash TEXT NOT NULL,
			hub_url TEXT NOT NULL,
			last_seen INTEGER,
			last_lat REAL,
			last_lng REAL,
			status TEXT DEFAULT 'active',
			signing_pubkey TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ftd_group ON fieldtrip_devices(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ftd_status ON fieldtrip_devices(status)`,
		`CREATE TABLE IF NOT EXISTS gps_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT REFERENCES fieldtrip_devices(id) ON DELETE CASCADE,
			lat REAL NOT NULL,
			lng REAL NOT NULL,
			accuracy REAL,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gps_device ON gps_logs(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gps_timestamp ON gps_logs(timestamp)`,
		`CREATE TABLE IF NOT EXISTS broadcasts (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES fieldtrip_groups(id),
			message TEXT NOT NULL,
			sound TEXT DEFAULT 'default',
			created_by TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			delivered_count INTEGER DEFAULT 0,
			failed_count INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bc_group ON broadcasts(group_id)`,
		`CREATE TABLE IF NOT EXISTS pending_commands (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			command_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at INTEGER NOT NULL,
			delivered_at INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pc_device ON pending_commands(device_id, status)`,
	}
	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (r *FieldTripRepository) GetHubURL() string {
	return r.hubURL
}

// CreateGroup creates a new field trip group
func (r *FieldTripRepository) CreateGroup(g *models.FieldTripGroup) error {
	_, err := r.db.NamedExec(`INSERT INTO fieldtrip_groups (id, name, group_key, broadcast_sound, update_policy, created_at, updated_at)
		VALUES (:id, :name, :group_key, :broadcast_sound, :update_policy, :created_at, :updated_at)`, g)
	return err
}

// GetGroupByKey retrieves a group by its group_key
func (r *FieldTripRepository) GetGroupByKey(key string) (*models.FieldTripGroup, error) {
	var g models.FieldTripGroup
	err := r.db.Get(&g, "SELECT * FROM fieldtrip_groups WHERE group_key = ?", key)
	return &g, err
}

// GetGroupByID retrieves a group by its ID
func (r *FieldTripRepository) GetGroupByID(id string) (*models.FieldTripGroup, error) {
	var g models.FieldTripGroup
	err := r.db.Get(&g, "SELECT * FROM fieldtrip_groups WHERE id = ?", id)
	return &g, err
}

// ListGroups returns all groups
func (r *FieldTripRepository) ListGroups() ([]models.FieldTripGroup, error) {
	var groups []models.FieldTripGroup
	err := r.db.Select(&groups, "SELECT * FROM fieldtrip_groups ORDER BY name")
	return groups, err
}

// CreateDevice creates a new field trip device
func (r *FieldTripRepository) CreateDevice(d *models.FieldTripDevice) error {
	_, err := r.db.NamedExec(`INSERT INTO fieldtrip_devices (id, name, group_id, api_key_hash, hub_url, status, signing_pubkey, created_at, updated_at)
		VALUES (:id, :name, :group_id, :api_key_hash, :hub_url, :status, :signing_pubkey, :created_at, :updated_at)`, d)
	return err
}

// GetDeviceByID retrieves a device by ID
func (r *FieldTripRepository) GetDeviceByID(id string) (*models.FieldTripDevice, error) {
	var d models.FieldTripDevice
	err := r.db.Get(&d, "SELECT * FROM fieldtrip_devices WHERE id = ?", id)
	return &d, err
}

// ListDevices returns all devices
func (r *FieldTripRepository) ListDevices() ([]models.FieldTripDevice, error) {
	var devices []models.FieldTripDevice
	err := r.db.Select(&devices, "SELECT * FROM fieldtrip_devices ORDER BY name")
	return devices, err
}

// ListDevicesByGroup returns all devices in a group
func (r *FieldTripRepository) ListDevicesByGroup(groupID string) ([]models.FieldTripDevice, error) {
	var devices []models.FieldTripDevice
	err := r.db.Select(&devices, "SELECT * FROM fieldtrip_devices WHERE group_id = ? ORDER BY name", groupID)
	return devices, err
}

// UpdateDeviceLocation updates device GPS location and last_seen
func (r *FieldTripRepository) UpdateDeviceLocation(id string, lat, lng float64, lastSeen int64) error {
	_, err := r.db.Exec(`UPDATE fieldtrip_devices SET last_lat=?, last_lng=?, last_seen=?, status='active', updated_at=? WHERE id=?`,
		lat, lng, lastSeen, lastSeen, id)
	return err
}

// SetDeviceStatus updates device status and last_seen
func (r *FieldTripRepository) SetDeviceStatus(id, status string, lastSeen int64) error {
	_, err := r.db.Exec(`UPDATE fieldtrip_devices SET status=?, last_seen=?, updated_at=? WHERE id=?`,
		status, lastSeen, lastSeen, id)
	return err
}

// UpdateDeviceName updates device name
func (r *FieldTripRepository) UpdateDeviceName(id, name string, updatedAt int64) error {
	_, err := r.db.Exec(`UPDATE fieldtrip_devices SET name=?, updated_at=? WHERE id=?`, name, updatedAt, id)
	return err
}

// DeleteDevice deletes a device
func (r *FieldTripRepository) DeleteDevice(id string) error {
	_, err := r.db.Exec("DELETE FROM fieldtrip_devices WHERE id=?", id)
	return err
}

// DeleteGroup deletes a group
func (r *FieldTripRepository) DeleteGroup(id string) error {
	_, err := r.db.Exec("DELETE FROM fieldtrip_groups WHERE id=?", id)
	return err
}

// InsertGPSLog inserts a GPS log entry
func (r *FieldTripRepository) InsertGPSLog(deviceID string, lat, lng, accuracy float64, ts, createdAt int64) error {
	_, err := r.db.Exec(`INSERT INTO gps_logs (device_id, lat, lng, accuracy, timestamp, created_at) VALUES (?,?,?,?,?,?)`,
		deviceID, lat, lng, accuracy, ts, createdAt)
	return err
}

// GetGPSHistory returns the most recent GPS entries for a device
func (r *FieldTripRepository) GetGPSHistory(deviceID string, limit int) ([]models.GPSReport, error) {
	rows, err := r.db.Queryx(`SELECT lat, lng, accuracy, timestamp FROM gps_logs
		WHERE device_id=? ORDER BY timestamp DESC LIMIT ?`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.GPSReport
	for rows.Next() {
		var r models.GPSReport
		r.DeviceID = deviceID
		if err := rows.Scan(&r.Lat, &r.Lng, &r.Accuracy, &r.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, r)
	}
	return logs, nil
}

// CreateBroadcast creates a broadcast record
func (r *FieldTripRepository) CreateBroadcast(b *models.Broadcast) error {
	_, err := r.db.NamedExec(`INSERT INTO broadcasts (id, group_id, message, sound, created_by, created_at, delivered_count, failed_count)
		VALUES (:id, :group_id, :message, :sound, :created_by, :created_at, :delivered_count, :failed_count)`, b)
	return err
}

// IncrementBroadcastCounts updates delivery counts
func (r *FieldTripRepository) IncrementBroadcastCounts(id string, delivered, failed int) error {
	_, err := r.db.Exec(`UPDATE broadcasts SET delivered_count=delivered_count+?, failed_count=failed_count+? WHERE id=?`,
		delivered, failed, id)
	return err
}

// PushPendingCommand adds a pending command for a device
func (r *FieldTripRepository) PushPendingCommand(cmd *models.PendingCommand) error {
	_, err := r.db.NamedExec(`INSERT INTO pending_commands (id, device_id, command_type, payload, status, created_at)
		VALUES (:id, :device_id, :command_type, :payload, :status, :created_at)`, cmd)
	return err
}

// PopPendingCommands returns and marks pending commands as delivered
func (r *FieldTripRepository) PopPendingCommands(deviceID string) ([]models.PendingCommand, error) {
	rows, err := r.db.Queryx(`SELECT * FROM pending_commands WHERE device_id=? AND status='pending' ORDER BY created_at`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cmds []models.PendingCommand
	for rows.Next() {
		var c models.PendingCommand
		if err := rows.StructScan(&c); err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	// Mark as delivered
	if len(cmds) > 0 {
		for _, c := range cmds {
			r.db.Exec(`UPDATE pending_commands SET status='delivered', delivered_at=? WHERE id=?`, c.CreatedAt, c.ID)
		}
	}
	return cmds, nil
}
