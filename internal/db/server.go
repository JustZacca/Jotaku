package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ServerDB struct {
	conn *sql.DB
}

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	Active       bool      `json:"active"`
}

type ServerNote struct {
	ID             string    `json:"id"`
	UserID         int64     `json:"user_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Tags           string    `json:"tags"`
	ParentFolderID string    `json:"parent_folder_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ServerFolder struct {
	ID             string    `json:"id"`
	UserID         int64     `json:"user_id"`
	Title          string    `json:"title"`
	ParentFolderID string    `json:"parent_folder_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ServerNoteVersion struct {
	ID         string    `json:"id"`
	NoteID     string    `json:"note_id"`
	UserID     int64     `json:"user_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Tags       string    `json:"tags"`
	Hash       string    `json:"hash"`
	VersionNum int       `json:"version_num"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewServerDB(dbPath string) (*ServerDB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &ServerDB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *ServerDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		active BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS notes (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT,
		parent_folder_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		parent_folder_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS note_versions (
		id TEXT PRIMARY KEY,
		note_id TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT,
		hash TEXT,
		version_num INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_notes_user ON notes(user_id);
	CREATE INDEX IF NOT EXISTS idx_notes_updated ON notes(updated_at);
	CREATE INDEX IF NOT EXISTS idx_notes_folder ON notes(parent_folder_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_folders_user ON folders(user_id);
	CREATE INDEX IF NOT EXISTS idx_folders_updated ON folders(updated_at);
	CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_folder_id);
	CREATE INDEX IF NOT EXISTS idx_versions_note ON note_versions(note_id);
	CREATE INDEX IF NOT EXISTS idx_versions_user ON note_versions(user_id);
	`
	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: add parent_folder_id column if not exists
	db.conn.Exec(`ALTER TABLE notes ADD COLUMN parent_folder_id TEXT`)

	return nil
}

func (db *ServerDB) Close() error {
	return db.conn.Close()
}

// User operations

func (db *ServerDB) CreateUser(username, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	result, err := db.conn.Exec(`
		INSERT INTO users (username, password_hash, created_at, active)
		VALUES (?, ?, ?, 1)
	`, username, string(hash), now)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:        id,
		Username:  username,
		CreatedAt: now,
		Active:    true,
	}, nil
}

func (db *ServerDB) GetUserByUsername(username string) (*User, error) {
	var u User
	err := db.conn.QueryRow(`
		SELECT id, username, password_hash, created_at, active
		FROM users WHERE username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.Active)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (db *ServerDB) GetUserByID(id int64) (*User, error) {
	var u User
	err := db.conn.QueryRow(`
		SELECT id, username, password_hash, created_at, active
		FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.Active)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (db *ServerDB) ValidatePassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// Note operations

func (db *ServerDB) ListNotesByUser(userID int64) ([]ServerNote, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, title, content, tags, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM notes
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []ServerNote
	for rows.Next() {
		var n ServerNote
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.ParentFolderID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (db *ServerDB) GetNote(id string, userID int64) (*ServerNote, error) {
	var n ServerNote
	err := db.conn.QueryRow(`
		SELECT id, user_id, title, content, tags, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM notes WHERE id = ? AND user_id = ?
	`, id, userID).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.ParentFolderID, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}
	return &n, nil
}

func (db *ServerDB) UpsertNote(userID int64, id, title, content, tags, parentFolderID string, createdAt, updatedAt time.Time) (*ServerNote, error) {
	if id == "" {
		id = uuid.New().String()
	}

	var folderID interface{} = nil
	if parentFolderID != "" {
		folderID = parentFolderID
	}

	_, err := db.conn.Exec(`
		INSERT INTO notes (id, user_id, title, content, tags, parent_folder_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			tags = excluded.tags,
			parent_folder_id = excluded.parent_folder_id,
			updated_at = excluded.updated_at
		WHERE user_id = ?
	`, id, userID, title, content, tags, folderID, createdAt, updatedAt, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert note: %w", err)
	}

	return &ServerNote{
		ID:             id,
		UserID:         userID,
		Title:          title,
		Content:        content,
		Tags:           tags,
		ParentFolderID: parentFolderID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func (db *ServerDB) DeleteNote(id string, userID int64) error {
	_, err := db.conn.Exec(`DELETE FROM notes WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}
	return nil
}

func (db *ServerDB) GetNotesSince(userID int64, since time.Time) ([]ServerNote, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, title, content, tags, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM notes
		WHERE user_id = ? AND updated_at > ?
		ORDER BY updated_at DESC
	`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []ServerNote
	for rows.Next() {
		var n ServerNote
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.ParentFolderID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Folder operations

func (db *ServerDB) ListFoldersByUser(userID int64) ([]ServerFolder, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, title, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM folders
		WHERE user_id = ?
		ORDER BY title ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []ServerFolder
	for rows.Next() {
		var f ServerFolder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Title, &f.ParentFolderID, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (db *ServerDB) GetFolder(id string, userID int64) (*ServerFolder, error) {
	var f ServerFolder
	err := db.conn.QueryRow(`
		SELECT id, user_id, title, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM folders WHERE id = ? AND user_id = ?
	`, id, userID).Scan(&f.ID, &f.UserID, &f.Title, &f.ParentFolderID, &f.CreatedAt, &f.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	return &f, nil
}

func (db *ServerDB) UpsertFolder(userID int64, id, title, parentFolderID string, createdAt, updatedAt time.Time) (*ServerFolder, error) {
	if id == "" {
		id = uuid.New().String()
	}

	var parentID interface{} = nil
	if parentFolderID != "" {
		parentID = parentFolderID
	}

	_, err := db.conn.Exec(`
		INSERT INTO folders (id, user_id, title, parent_folder_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			parent_folder_id = excluded.parent_folder_id,
			updated_at = excluded.updated_at
		WHERE user_id = ?
	`, id, userID, title, parentID, createdAt, updatedAt, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert folder: %w", err)
	}

	return &ServerFolder{
		ID:             id,
		UserID:         userID,
		Title:          title,
		ParentFolderID: parentFolderID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func (db *ServerDB) DeleteFolder(id string, userID int64) error {
	_, err := db.conn.Exec(`DELETE FROM folders WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}
	return nil
}

func (db *ServerDB) GetFoldersSince(userID int64, since time.Time) ([]ServerFolder, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, title, COALESCE(parent_folder_id, ''), created_at, updated_at
		FROM folders
		WHERE user_id = ? AND updated_at > ?
		ORDER BY updated_at DESC
	`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []ServerFolder
	for rows.Next() {
		var f ServerFolder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Title, &f.ParentFolderID, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// Note version operations

func (db *ServerDB) ListVersionsByNote(noteID string, userID int64) ([]ServerNoteVersion, error) {
	rows, err := db.conn.Query(`
		SELECT id, note_id, user_id, title, content, tags, COALESCE(hash, ''), version_num, created_at
		FROM note_versions
		WHERE note_id = ? AND user_id = ?
		ORDER BY version_num DESC
	`, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []ServerNoteVersion
	for rows.Next() {
		var v ServerNoteVersion
		if err := rows.Scan(&v.ID, &v.NoteID, &v.UserID, &v.Title, &v.Content, &v.Tags, &v.Hash, &v.VersionNum, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *ServerDB) UpsertVersion(userID int64, id, noteID, title, content, tags, hash string, versionNum int, createdAt time.Time) (*ServerNoteVersion, error) {
	if id == "" {
		id = uuid.New().String()
	}

	_, err := db.conn.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, title, content, tags, hash, version_num, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			tags = excluded.tags,
			hash = excluded.hash
	`, id, noteID, userID, title, content, tags, hash, versionNum, createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert version: %w", err)
	}

	return &ServerNoteVersion{
		ID:         id,
		NoteID:     noteID,
		UserID:     userID,
		Title:      title,
		Content:    content,
		Tags:       tags,
		Hash:       hash,
		VersionNum: versionNum,
		CreatedAt:  createdAt,
	}, nil
}

func (db *ServerDB) GetVersionsSince(userID int64, since time.Time) ([]ServerNoteVersion, error) {
	rows, err := db.conn.Query(`
		SELECT id, note_id, user_id, title, content, tags, COALESCE(hash, ''), version_num, created_at
		FROM note_versions
		WHERE user_id = ? AND created_at > ?
		ORDER BY created_at DESC
	`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []ServerNoteVersion
	for rows.Next() {
		var v ServerNoteVersion
		if err := rows.Scan(&v.ID, &v.NoteID, &v.UserID, &v.Title, &v.Content, &v.Tags, &v.Hash, &v.VersionNum, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}
