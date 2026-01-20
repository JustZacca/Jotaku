package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT,
		password TEXT,
		parent_folder_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		server_id TEXT,
		sync_status TEXT DEFAULT 'local',
		deleted INTEGER DEFAULT 0,
		FOREIGN KEY(parent_folder_id) REFERENCES folders(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		password TEXT,
		parent_folder_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted INTEGER DEFAULT 0,
		FOREIGN KEY(parent_folder_id) REFERENCES folders(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS note_versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		note_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT,
		hash TEXT,
		version_num INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(note_id) REFERENCES notes(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_notes_title ON notes(title);
	CREATE INDEX IF NOT EXISTS idx_notes_updated ON notes(updated_at);
	CREATE INDEX IF NOT EXISTS idx_notes_server_id ON notes(server_id);
	CREATE INDEX IF NOT EXISTS idx_notes_sync ON notes(sync_status);
	CREATE INDEX IF NOT EXISTS idx_notes_parent ON notes(parent_folder_id);
	CREATE INDEX IF NOT EXISTS idx_folders_title ON folders(title);
	CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_folder_id);
	CREATE INDEX IF NOT EXISTS idx_versions_note ON note_versions(note_id);
	CREATE INDEX IF NOT EXISTS idx_versions_num ON note_versions(version_num);
	`
	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: add new columns if they don't exist
	// Ignore errors as columns may already exist
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN password TEXT`)
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN parent_folder_id INTEGER`)
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN server_id TEXT`)
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN sync_status TEXT DEFAULT 'local'`)
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN deleted INTEGER DEFAULT 0`)
	db.conn.Exec(`ALTER TABLE note_versions ADD COLUMN hash TEXT`)

	// Ensure indexes exist
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_notes_server_id ON notes(server_id)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_notes_sync ON notes(sync_status)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_notes_parent ON notes(parent_folder_id)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_folder_id)`)

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
