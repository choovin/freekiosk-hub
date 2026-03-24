package repositories

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) (*SQLiteMDMTabletRepository, *sql.DB) {
	// Create temp database file
	tmpFile, err := os.CreateTemp("", "test-mdm-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite", tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create schema
	if err := createTestSchema(db); err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to create schema: %v", err)
	}

	repo := NewSQLiteMDMTabletRepository(db)

	return repo, db
}

// createTestSchema creates the required tables for testing
func createTestSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS mdm_devices (
		id TEXT PRIMARY KEY,
		number TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		imei TEXT,
		phone TEXT,
		model TEXT,
		manufacturer TEXT,
		os_version TEXT,
		sdk_version INTEGER,
		app_version TEXT,
		app_version_code INTEGER,
		carrier TEXT,
		last_lat REAL,
		last_lng REAL,
		last_location_time INTEGER,
		last_seen INTEGER,
		status TEXT DEFAULT 'active',
		configuration_id TEXT,
		group_id TEXT,
		tenant_id NOT NULL,
		metadata TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS device_groups (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT,
		description TEXT,
		tenant_id NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS device_tags (
		id TEXT PRIMARY KEY,
		device_id NOT NULL,
		tag NOT NULL,
		value TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS device_events (
		id TEXT PRIMARY KEY,
		device_id NOT NULL,
		event_type NOT NULL,
		event_data TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_mdm_devices_tenant ON mdm_devices(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_mdm_devices_status ON mdm_devices(status);
	CREATE INDEX IF NOT EXISTS idx_mdm_devices_number ON mdm_devices(number);
	CREATE INDEX IF NOT EXISTS idx_device_groups_tenant ON device_groups(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_device_events_device ON device_events(device_id);
	CREATE INDEX IF NOT EXISTS idx_device_events_created ON device_events(created_at);
	`

	_, err := db.Exec(schema)
	return err
}

// cleanupTestDB closes the database and removes the temp file
func cleanupTestDB(db *sql.DB, t *testing.T) {
	db.Close()
}

// generateUUID generates a simple UUID for testing
func generateUUID() string {
	return fmt.Sprintf("test-uuid-%d", time.Now().UnixNano())
}
