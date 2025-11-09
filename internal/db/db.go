package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection and runs migrations
func NewDB(dbPath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{sqlDB}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("[INFO] Database initialized successfully")
	return db, nil
}

// migrate runs all database migrations
func (db *DB) migrate() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			google_id TEXT UNIQUE NOT NULL,
			email TEXT NOT NULL,
			encrypted_access_token TEXT,
			encrypted_refresh_token TEXT,
			token_expiry DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Import jobs table
		`CREATE TABLE IF NOT EXISTS import_jobs (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			total_files INTEGER DEFAULT 0,
			completed_files INTEGER DEFAULT 0,
			failed_files INTEGER DEFAULT 0,
			total_bytes INTEGER DEFAULT 0,
			transferred_bytes INTEGER DEFAULT 0,
			started_at DATETIME,
			completed_at DATETIME,
			options TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Import files table
		`CREATE TABLE IF NOT EXISTS import_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			drive_file_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_size INTEGER DEFAULT 0,
			status TEXT NOT NULL,
			progress INTEGER DEFAULT 0,
			sha256_hash TEXT,
			ipfs_cid TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES import_jobs(id) ON DELETE CASCADE
		)`,

		// Sync configurations table (Phase 3)
		`CREATE TABLE IF NOT EXISTS sync_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			drive_folder_id TEXT NOT NULL,
			enabled BOOLEAN DEFAULT TRUE,
			sync_interval INTEGER DEFAULT 60,
			last_sync_at DATETIME,
			next_sync_at DATETIME,
			options TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_import_jobs_user_id ON import_jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_import_files_job_id ON import_files(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_import_files_status ON import_files(status)`,
		`CREATE INDEX IF NOT EXISTS idx_import_files_drive_file_id ON import_files(drive_file_id)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	log.Printf("[INFO] Applied %d database migrations", len(migrations))
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
