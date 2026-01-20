package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Health check

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// Auth handlers

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !s.db.ValidatePassword(user, req.Password) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.Active {
		jsonError(w, "user is disabled", http.StatusForbidden)
		return
	}

	token, expiresAt, err := s.jwt.Generate(user.ID, user.Username)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		UserID:    user.ID,
		Username:  user.Username,
	}, http.StatusOK)
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	existing, _ := s.db.GetUserByUsername(req.Username)
	if existing != nil {
		jsonError(w, "username already exists", http.StatusConflict)
		return
	}

	user, err := s.db.CreateUser(req.Username, req.Password)
	if err != nil {
		jsonError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	token, expiresAt, err := s.jwt.Generate(user.ID, user.Username)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		UserID:    user.ID,
		Username:  user.Username,
	}, http.StatusCreated)
}

// Notes handlers

type NoteResponse struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Tags           string `json:"tags"`
	ParentFolderID string `json:"parent_folder_id,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type NoteListResponse struct {
	Notes []NoteResponse `json:"notes"`
}

func (s *Server) listNotesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	notes, err := s.db.ListNotesByUser(user.ID)
	if err != nil {
		jsonError(w, "failed to list notes", http.StatusInternalServerError)
		return
	}

	response := NoteListResponse{Notes: make([]NoteResponse, len(notes))}
	for i, n := range notes {
		response.Notes[i] = NoteResponse{
			ID:             n.ID,
			Title:          n.Title,
			Content:        n.Content,
			Tags:           n.Tags,
			ParentFolderID: n.ParentFolderID,
			CreatedAt:      n.CreatedAt.Unix(),
			UpdatedAt:      n.UpdatedAt.Unix(),
		}
	}

	jsonResponse(w, response, http.StatusOK)
}

func (s *Server) getNoteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	noteID := chi.URLParam(r, "id")

	note, err := s.db.GetNote(noteID, user.ID)
	if err != nil {
		jsonError(w, "failed to get note", http.StatusInternalServerError)
		return
	}
	if note == nil {
		jsonError(w, "note not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, NoteResponse{
		ID:             note.ID,
		Title:          note.Title,
		Content:        note.Content,
		Tags:           note.Tags,
		ParentFolderID: note.ParentFolderID,
		CreatedAt:      note.CreatedAt.Unix(),
		UpdatedAt:      note.UpdatedAt.Unix(),
	}, http.StatusOK)
}

type UpsertNoteRequest struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Tags           string `json:"tags"`
	ParentFolderID string `json:"parent_folder_id,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

func (s *Server) upsertNoteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	var req UpsertNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		jsonError(w, "title required", http.StatusBadRequest)
		return
	}

	createdAt := time.Now()
	if req.CreatedAt > 0 {
		createdAt = time.Unix(req.CreatedAt, 0)
	}

	updatedAt := time.Now()
	if req.UpdatedAt > 0 {
		updatedAt = time.Unix(req.UpdatedAt, 0)
	}

	note, err := s.db.UpsertNote(user.ID, req.ID, req.Title, req.Content, req.Tags, req.ParentFolderID, createdAt, updatedAt)
	if err != nil {
		jsonError(w, "failed to save note", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, NoteResponse{
		ID:             note.ID,
		Title:          note.Title,
		Content:        note.Content,
		Tags:           note.Tags,
		ParentFolderID: note.ParentFolderID,
		CreatedAt:      note.CreatedAt.Unix(),
		UpdatedAt:      note.UpdatedAt.Unix(),
	}, http.StatusOK)
}

func (s *Server) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	noteID := chi.URLParam(r, "id")

	if err := s.db.DeleteNote(noteID, user.ID); err != nil {
		jsonError(w, "failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) syncNotesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		sinceUnix := int64(0)
		if _, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since, _ = time.Parse(time.RFC3339, sinceStr)
		} else {
			if n, err := time.ParseDuration(sinceStr); err == nil {
				since = time.Now().Add(-n)
			} else {
				// Try parsing as unix timestamp
				if ts, err := json.Number(sinceStr).Int64(); err == nil {
					sinceUnix = ts
					since = time.Unix(sinceUnix, 0)
				}
			}
		}
	}

	notes, err := s.db.GetNotesSince(user.ID, since)
	if err != nil {
		jsonError(w, "failed to get notes", http.StatusInternalServerError)
		return
	}

	response := NoteListResponse{Notes: make([]NoteResponse, len(notes))}
	for i, n := range notes {
		response.Notes[i] = NoteResponse{
			ID:             n.ID,
			Title:          n.Title,
			Content:        n.Content,
			Tags:           n.Tags,
			ParentFolderID: n.ParentFolderID,
			CreatedAt:      n.CreatedAt.Unix(),
			UpdatedAt:      n.UpdatedAt.Unix(),
		}
	}

	jsonResponse(w, response, http.StatusOK)
}

// Folder handlers

type FolderResponse struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ParentFolderID string `json:"parent_folder_id,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type FolderListResponse struct {
	Folders []FolderResponse `json:"folders"`
}

type UpsertFolderRequest struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ParentFolderID string `json:"parent_folder_id,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

func (s *Server) listFoldersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	folders, err := s.db.ListFoldersByUser(user.ID)
	if err != nil {
		jsonError(w, "failed to list folders", http.StatusInternalServerError)
		return
	}

	response := FolderListResponse{Folders: make([]FolderResponse, len(folders))}
	for i, f := range folders {
		response.Folders[i] = FolderResponse{
			ID:             f.ID,
			Title:          f.Title,
			ParentFolderID: f.ParentFolderID,
			CreatedAt:      f.CreatedAt.Unix(),
			UpdatedAt:      f.UpdatedAt.Unix(),
		}
	}

	jsonResponse(w, response, http.StatusOK)
}

func (s *Server) getFolderHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	folderID := chi.URLParam(r, "id")

	folder, err := s.db.GetFolder(folderID, user.ID)
	if err != nil {
		jsonError(w, "failed to get folder", http.StatusInternalServerError)
		return
	}
	if folder == nil {
		jsonError(w, "folder not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, FolderResponse{
		ID:             folder.ID,
		Title:          folder.Title,
		ParentFolderID: folder.ParentFolderID,
		CreatedAt:      folder.CreatedAt.Unix(),
		UpdatedAt:      folder.UpdatedAt.Unix(),
	}, http.StatusOK)
}

func (s *Server) upsertFolderHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	var req UpsertFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		jsonError(w, "title required", http.StatusBadRequest)
		return
	}

	createdAt := time.Now()
	if req.CreatedAt > 0 {
		createdAt = time.Unix(req.CreatedAt, 0)
	}

	updatedAt := time.Now()
	if req.UpdatedAt > 0 {
		updatedAt = time.Unix(req.UpdatedAt, 0)
	}

	folder, err := s.db.UpsertFolder(user.ID, req.ID, req.Title, req.ParentFolderID, createdAt, updatedAt)
	if err != nil {
		jsonError(w, "failed to save folder", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, FolderResponse{
		ID:             folder.ID,
		Title:          folder.Title,
		ParentFolderID: folder.ParentFolderID,
		CreatedAt:      folder.CreatedAt.Unix(),
		UpdatedAt:      folder.UpdatedAt.Unix(),
	}, http.StatusOK)
}

func (s *Server) deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	folderID := chi.URLParam(r, "id")

	if err := s.db.DeleteFolder(folderID, user.ID); err != nil {
		jsonError(w, "failed to delete folder", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) syncFoldersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		if _, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since, _ = time.Parse(time.RFC3339, sinceStr)
		} else {
			if n, err := time.ParseDuration(sinceStr); err == nil {
				since = time.Now().Add(-n)
			} else {
				if ts, err := json.Number(sinceStr).Int64(); err == nil {
					since = time.Unix(ts, 0)
				}
			}
		}
	}

	folders, err := s.db.GetFoldersSince(user.ID, since)
	if err != nil {
		jsonError(w, "failed to get folders", http.StatusInternalServerError)
		return
	}

	response := FolderListResponse{Folders: make([]FolderResponse, len(folders))}
	for i, f := range folders {
		response.Folders[i] = FolderResponse{
			ID:             f.ID,
			Title:          f.Title,
			ParentFolderID: f.ParentFolderID,
			CreatedAt:      f.CreatedAt.Unix(),
			UpdatedAt:      f.UpdatedAt.Unix(),
		}
	}

	jsonResponse(w, response, http.StatusOK)
}
