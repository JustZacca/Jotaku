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
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_notes_user ON notes(user_id);
	CREATE INDEX IF NOT EXISTS idx_notes_updated ON notes(updated_at);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`
	_, err := db.conn.Exec(schema)
	return err
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
		SELECT id, user_id, title, content, tags, created_at, updated_at
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
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (db *ServerDB) GetNote(id string, userID int64) (*ServerNote, error) {
	var n ServerNote
	err := db.conn.QueryRow(`
		SELECT id, user_id, title, content, tags, created_at, updated_at
		FROM notes WHERE id = ? AND user_id = ?
	`, id, userID).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}
	return &n, nil
}

func (db *ServerDB) UpsertNote(userID int64, id, title, content, tags string, createdAt, updatedAt time.Time) (*ServerNote, error) {
	if id == "" {
		id = uuid.New().String()
	}

	_, err := db.conn.Exec(`
		INSERT INTO notes (id, user_id, title, content, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			tags = excluded.tags,
			updated_at = excluded.updated_at
		WHERE user_id = ?
	`, id, userID, title, content, tags, createdAt, updatedAt, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert note: %w", err)
	}

	return &ServerNote{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Content:   content,
		Tags:      tags,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
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
		SELECT id, user_id, title, content, tags, created_at, updated_at
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
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
