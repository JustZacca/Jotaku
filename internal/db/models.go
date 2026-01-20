package db

import "time"

type SyncStatus string

const (
	SyncStatusLocal   SyncStatus = "local"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusPending SyncStatus = "pending"
)

type Note struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Tags         []string   `json:"tags"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ServerID     string     `json:"server_id,omitempty"`
	SyncStatus   SyncStatus `json:"sync_status"`
	Deleted      bool       `json:"deleted"`
	Password     string     `json:"-"`
	ParentFolder int64      `json:"parent_folder,omitempty"`
}

type Folder struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Password     string    `json:"-"`
	ParentFolder int64     `json:"parent_folder,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Deleted      bool      `json:"deleted"`
}

type ListItem interface {
	GetID() int64
	GetTitle() string
	GetType() string // "note" o "folder"
	IsDeleted() bool
}

type NoteListItem struct {
	ID         int64      `json:"id"`
	Title      string     `json:"title"`
	UpdatedAt  time.Time  `json:"updated_at"`
	SyncStatus SyncStatus `json:"sync_status"`
	Type       string     `json:"type"` // "note" o "folder"
}

func (n NoteListItem) GetID() int64 {
	return n.ID
}

func (n NoteListItem) GetTitle() string {
	return n.Title
}

func (n NoteListItem) GetType() string {
	return n.Type
}

func (n NoteListItem) IsDeleted() bool {
	return false
}

type NoteVersion struct {
	ID         int64     `json:"id"`
	NoteID     int64     `json:"note_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"created_at"`
	VersionNum int       `json:"version_num"`
}
