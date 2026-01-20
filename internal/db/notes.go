package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (db *DB) ListNotes() ([]NoteListItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, updated_at, COALESCE(sync_status, 'local')
		FROM notes
		WHERE (deleted = 0 OR deleted IS NULL) AND parent_folder_id IS NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []NoteListItem
	for rows.Next() {
		var n NoteListItem
		var syncStatus string
		if err := rows.Scan(&n.ID, &n.Title, &n.UpdatedAt, &syncStatus); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		n.SyncStatus = SyncStatus(syncStatus)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (db *DB) GetNote(id int64) (*Note, error) {
	var n Note
	var tagsJSON sql.NullString
	var serverID sql.NullString
	var syncStatus sql.NullString
	var deleted sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, title, content, tags, created_at, updated_at,
		       server_id, COALESCE(sync_status, 'local'), COALESCE(deleted, 0)
		FROM notes WHERE id = ?
	`, id).Scan(&n.ID, &n.Title, &n.Content, &tagsJSON, &n.CreatedAt, &n.UpdatedAt,
		&serverID, &syncStatus, &deleted)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &n.Tags); err != nil {
			n.Tags = []string{}
		}
	}

	if serverID.Valid {
		n.ServerID = serverID.String
	}
	if syncStatus.Valid {
		n.SyncStatus = SyncStatus(syncStatus.String)
	} else {
		n.SyncStatus = SyncStatusLocal
	}
	n.Deleted = deleted.Valid && deleted.Int64 == 1

	return &n, nil
}

func (db *DB) CreateNote(title, content string, tags []string) (*Note, error) {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	result, err := db.conn.Exec(`
		INSERT INTO notes (title, content, tags, created_at, updated_at, sync_status, deleted)
		VALUES (?, ?, ?, ?, ?, 'pending', 0)
	`, title, content, string(tagsJSON), now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Note{
		ID:         id,
		Title:      title,
		Content:    content,
		Tags:       tags,
		CreatedAt:  now,
		UpdatedAt:  now,
		SyncStatus: SyncStatusPending,
	}, nil
}

func (db *DB) CreateNoteInFolder(title, content string, tags []string, folderID int64) (*Note, error) {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	var parentID interface{} = nil
	if folderID > 0 {
		parentID = folderID
	}

	result, err := db.conn.Exec(`
		INSERT INTO notes (title, content, tags, parent_folder_id, created_at, updated_at, sync_status, deleted)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', 0)
	`, title, content, string(tagsJSON), parentID, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Note{
		ID:           id,
		Title:        title,
		Content:      content,
		Tags:         tags,
		CreatedAt:    now,
		UpdatedAt:    now,
		SyncStatus:   SyncStatusPending,
		ParentFolder: folderID,
	}, nil
}

func (db *DB) UpdateNote(id int64, title, content string, tags []string) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	_, err = db.conn.Exec(`
		UPDATE notes
		SET title = ?, content = ?, tags = ?, updated_at = ?, sync_status = 'pending'
		WHERE id = ?
	`, title, content, string(tagsJSON), time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

func (db *DB) DeleteNote(id int64) error {
	// Soft delete - mark as deleted and pending sync
	_, err := db.conn.Exec(`
		UPDATE notes SET deleted = 1, sync_status = 'pending', updated_at = ?
		WHERE id = ?
	`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}
	return nil
}

func (db *DB) SearchNotes(query string, tags []string) ([]NoteListItem, error) {
	var args []interface{}
	var conditions []string

	baseQuery := `SELECT id, title, updated_at, COALESCE(sync_status, 'local') FROM notes WHERE (deleted = 0 OR deleted IS NULL)`

	if query != "" {
		conditions = append(conditions, `(title LIKE ? OR content LIKE ?)`)
		searchTerm := "%" + query + "%"
		args = append(args, searchTerm, searchTerm)
	}

	for _, tag := range tags {
		conditions = append(conditions, `tags LIKE ?`)
		args = append(args, "%\""+tag+"\"%")
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY updated_at DESC"

	rows, err := db.conn.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}
	defer rows.Close()

	var notes []NoteListItem
	for rows.Next() {
		var n NoteListItem
		var syncStatus string
		if err := rows.Scan(&n.ID, &n.Title, &n.UpdatedAt, &syncStatus); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		n.SyncStatus = SyncStatus(syncStatus)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Sync-related functions

func (db *DB) GetPendingNotes() ([]Note, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, content, tags, created_at, updated_at, server_id, sync_status, COALESCE(deleted, 0)
		FROM notes
		WHERE sync_status = 'pending'
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		var tagsJSON sql.NullString
		var serverID sql.NullString
		var syncStatus string
		var deleted int

		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &tagsJSON, &n.CreatedAt, &n.UpdatedAt,
			&serverID, &syncStatus, &deleted); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}

		if tagsJSON.Valid && tagsJSON.String != "" {
			json.Unmarshal([]byte(tagsJSON.String), &n.Tags)
		}
		if serverID.Valid {
			n.ServerID = serverID.String
		}
		n.SyncStatus = SyncStatus(syncStatus)
		n.Deleted = deleted == 1

		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (db *DB) SetNoteSynced(id int64, serverID string) error {
	_, err := db.conn.Exec(`
		UPDATE notes SET server_id = ?, sync_status = 'synced' WHERE id = ?
	`, serverID, id)
	return err
}

func (db *DB) GetNoteByServerID(serverID string) (*Note, error) {
	var n Note
	var tagsJSON sql.NullString
	var srvID sql.NullString
	var syncStatus sql.NullString
	var deleted sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, title, content, tags, created_at, updated_at, server_id, sync_status, COALESCE(deleted, 0)
		FROM notes WHERE server_id = ?
	`, serverID).Scan(&n.ID, &n.Title, &n.Content, &tagsJSON, &n.CreatedAt, &n.UpdatedAt,
		&srvID, &syncStatus, &deleted)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tagsJSON.Valid && tagsJSON.String != "" {
		json.Unmarshal([]byte(tagsJSON.String), &n.Tags)
	}
	if srvID.Valid {
		n.ServerID = srvID.String
	}
	if syncStatus.Valid {
		n.SyncStatus = SyncStatus(syncStatus.String)
	}
	n.Deleted = deleted.Valid && deleted.Int64 == 1

	return &n, nil
}

func (db *DB) UpsertFromServer(serverID, title, content, tags string, createdAt, updatedAt time.Time) error {
	existing, _ := db.GetNoteByServerID(serverID)

	if existing != nil {
		// Update only if server version is newer
		if updatedAt.After(existing.UpdatedAt) {
			_, err := db.conn.Exec(`
				UPDATE notes SET title = ?, content = ?, tags = ?, updated_at = ?, sync_status = 'synced'
				WHERE server_id = ?
			`, title, content, tags, updatedAt, serverID)
			return err
		}
		return nil
	}

	// Insert new note from server
	_, err := db.conn.Exec(`
		INSERT INTO notes (title, content, tags, created_at, updated_at, server_id, sync_status, deleted)
		VALUES (?, ?, ?, ?, ?, ?, 'synced', 0)
	`, title, content, tags, createdAt, updatedAt, serverID)
	return err
}

func (db *DB) PermanentlyDeleteSynced(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM notes WHERE id = ? AND deleted = 1`, id)
	return err
}

// Version control functions

func (db *DB) SaveNoteVersion(noteID int64, title, content string, tags []string) error {
	// Calculate hash of content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])[:12] // First 12 chars for short hash

	// Check if the last version has the same hash (no actual changes)
	var lastHash sql.NullString
	err := db.conn.QueryRow(`
		SELECT hash FROM note_versions 
		WHERE note_id = ? 
		ORDER BY version_num DESC 
		LIMIT 1
	`, noteID).Scan(&lastHash)

	// If last version has same hash, don't create a new version (prevents duplicates during typing)
	if err == nil && lastHash.Valid && lastHash.String == hashStr {
		return nil
	}

	// Get current version number for this note
	var maxVersion int
	err = db.conn.QueryRow(`SELECT COALESCE(MAX(version_num), 0) FROM note_versions WHERE note_id = ?`, noteID).Scan(&maxVersion)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	tagsJSON, _ := json.Marshal(tags)

	_, err = db.conn.Exec(`
		INSERT INTO note_versions (note_id, title, content, tags, hash, version_num, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, noteID, title, content, string(tagsJSON), hashStr, maxVersion+1)

	return err
}

func (db *DB) GetNoteVersions(noteID int64) ([]NoteVersion, error) {
	rows, err := db.conn.Query(`
		SELECT id, note_id, title, content, tags, hash, version_num, created_at
		FROM note_versions
		WHERE note_id = ?
		ORDER BY version_num DESC
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []NoteVersion
	for rows.Next() {
		var v NoteVersion
		var tagsJSON string
		var hash sql.NullString
		err := rows.Scan(&v.ID, &v.NoteID, &v.Title, &v.Content, &tagsJSON, &hash, &v.VersionNum, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		if hash.Valid {
			v.Hash = hash.String
		}
		json.Unmarshal([]byte(tagsJSON), &v.Tags)
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *DB) GetNoteVersion(versionID int64) (*NoteVersion, error) {
	var v NoteVersion
	var tagsJSON string
	var hash sql.NullString
	err := db.conn.QueryRow(`
		SELECT id, note_id, title, content, tags, hash, version_num, created_at
		FROM note_versions
		WHERE id = ?
	`, versionID).Scan(&v.ID, &v.NoteID, &v.Title, &v.Content, &tagsJSON, &hash, &v.VersionNum, &v.CreatedAt)

	if err != nil {
		return nil, err
	}
	if hash.Valid {
		v.Hash = hash.String
	}
	json.Unmarshal([]byte(tagsJSON), &v.Tags)
	return &v, nil
}

func (db *DB) RestoreNoteVersion(noteID int64, versionID int64) error {
	version, err := db.GetNoteVersion(versionID)
	if err != nil {
		return err
	}

	tagsJSON, _ := json.Marshal(version.Tags)
	_, err = db.conn.Exec(`
		UPDATE notes
		SET title = ?, content = ?, tags = ?, updated_at = CURRENT_TIMESTAMP, sync_status = 'pending'
		WHERE id = ?
	`, version.Title, version.Content, string(tagsJSON), noteID)

	return err
}

// Folder operations
func (db *DB) CreateFolder(title string, parentID int64) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO folders (title, parent_folder_id, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, title, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to create folder: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) GetFolder(id int64) (*Folder, error) {
	var f Folder
	var parentID sql.NullInt64
	var password sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, title, password, parent_folder_id, created_at, updated_at, COALESCE(deleted, 0)
		FROM folders WHERE id = ?
	`, id).Scan(&f.ID, &f.Title, &password, &parentID, &f.CreatedAt, &f.UpdatedAt, &f.Deleted)

	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	if password.Valid {
		f.Password = password.String
	}
	if parentID.Valid {
		f.ParentFolder = parentID.Int64
	}

	return &f, nil
}

func (db *DB) ListFolders(parentID int64) ([]Folder, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, password, parent_folder_id, created_at, updated_at, COALESCE(deleted, 0)
		FROM folders
		WHERE (parent_folder_id = ? OR (parent_folder_id IS NULL AND ? = 0))
		AND (deleted = 0 OR deleted IS NULL)
		ORDER BY title ASC
	`, parentID, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		var parentID sql.NullInt64
		var password sql.NullString
		if err := rows.Scan(&f.ID, &f.Title, &password, &parentID, &f.CreatedAt, &f.UpdatedAt, &f.Deleted); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		if password.Valid {
			f.Password = password.String
		}
		if parentID.Valid {
			f.ParentFolder = parentID.Int64
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (db *DB) ListNotesInFolder(folderID int64) ([]NoteListItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, updated_at, COALESCE(sync_status, 'local'), 'note' as type
		FROM notes
		WHERE parent_folder_id = ? AND (deleted = 0 OR deleted IS NULL)
		ORDER BY updated_at DESC
	`, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes in folder: %w", err)
	}
	defer rows.Close()

	var notes []NoteListItem
	for rows.Next() {
		var n NoteListItem
		var syncStatus string
		if err := rows.Scan(&n.ID, &n.Title, &n.UpdatedAt, &syncStatus, &n.Type); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		n.SyncStatus = SyncStatus(syncStatus)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (db *DB) CountNotesInFolder(folderID int64) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM notes
		WHERE parent_folder_id = ? AND (deleted = 0 OR deleted IS NULL)
	`, folderID).Scan(&count)
	return count, err
}

func (db *DB) SetNotePassword(noteID int64, password string) error {
	_, err := db.conn.Exec(`
		UPDATE notes SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, password, noteID)
	return err
}

func (db *DB) SetFolderPassword(folderID int64, password string) error {
	_, err := db.conn.Exec(`
		UPDATE folders SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, password, folderID)
	return err
}

func (db *DB) DeleteFolder(id int64) error {
	_, err := db.conn.Exec(`UPDATE folders SET deleted = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}
