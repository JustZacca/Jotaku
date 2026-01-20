package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

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

type NoteResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type NoteListResponse struct {
	Notes []NoteResponse `json:"notes"`
}

type UpsertNoteRequest struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) IsConfigured() bool {
	return c.baseURL != ""
}

func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

func (c *Client) Login(username, password string) (*LoginResponse, error) {
	req := LoginRequest{Username: username, Password: password}

	var resp LoginResponse
	if err := c.post("/api/auth/login", req, &resp); err != nil {
		return nil, err
	}

	c.token = resp.Token
	return &resp, nil
}

func (c *Client) Register(username, password string) (*LoginResponse, error) {
	req := LoginRequest{Username: username, Password: password}

	var resp LoginResponse
	if err := c.post("/api/auth/register", req, &resp); err != nil {
		return nil, err
	}

	c.token = resp.Token
	return &resp, nil
}

func (c *Client) ListNotes() ([]NoteResponse, error) {
	var resp NoteListResponse
	if err := c.get("/api/notes", &resp); err != nil {
		return nil, err
	}
	return resp.Notes, nil
}

func (c *Client) GetNote(id string) (*NoteResponse, error) {
	var resp NoteResponse
	if err := c.get("/api/notes/"+id, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpsertNote(note UpsertNoteRequest) (*NoteResponse, error) {
	var resp NoteResponse
	if err := c.post("/api/notes", note, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeleteNote(id string) error {
	return c.delete("/api/notes/" + id)
}

func (c *Client) SyncNotes(since int64) ([]NoteResponse, error) {
	url := "/api/notes/sync"
	if since > 0 {
		url = fmt.Sprintf("/api/notes/sync?since=%d", since)
	}

	var resp NoteListResponse
	if err := c.get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Notes, nil
}

func (c *Client) Ping() error {
	return c.get("/health", nil)
}

// HTTP helpers

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req, result)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, nil)
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
