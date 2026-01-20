package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nzaccagnino/go-notes/internal/api"
	"github.com/nzaccagnino/go-notes/internal/config"
	"github.com/nzaccagnino/go-notes/internal/crypto"
	"github.com/nzaccagnino/go-notes/internal/db"
	"github.com/nzaccagnino/go-notes/internal/i18n"
)

// formatBytes converts bytes to human-readable format (B, KB, MB, GB)
func formatBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

type Mode int

const (
	ModeNormal Mode = iota
	ModeEditing
	ModeSearch
	ModeNewNote
	ModeConfirmDelete
	ModeHelp
	ModeHistory
	ModeEditTags
	ModeSetPassword
	ModeNewChoice
)

type Panel int

const (
	PanelList Panel = iota
	PanelContent
	PanelMetadata
)

type Model struct {
	db        *db.DB
	encryptor *crypto.Encryptor
	config    *config.Config
	apiClient *api.Client

	notes           []db.NoteListItem
	currentNote     *db.Note
	currentReadOnly bool
	cursor          int
	listOffset      int

	mode        Mode
	activePanel Panel

	textarea  textarea.Model
	textinput textinput.Model

	searchQuery string
	searchTags  []string

	width  int
	height int

	keys KeyMap

	lastSave time.Time
	dirty    bool

	// Sync state
	online        bool
	syncing       bool
	syncStatus    string
	uploadBytes   int64
	downloadBytes int64
	tickCount     int // Counter for periodic sync (every 5 minutes = 100 ticks of 3 seconds)

	// History state
	noteVersions  []db.NoteVersion
	versionCursor int

	// Folder/Password state
	currentFolder      int64      // 0 = root
	currentFolderData  *db.Folder // Metadata della cartella selezionata
	folders            []db.Folder
	currentItemType    string // "note" o "folder"
	passwordInput      textinput.Model
	passwordTarget     int64  // ID della nota/cartella per cui settare password
	passwordTargetType string // "note" o "folder"
	newChoice          int    // 0 = note, 1 = folder (per ModeNewChoice)

	// Delete state
	deleteTargetID    int64  // ID dell'elemento da eliminare
	deleteTargetType  string // "note" o "folder"
	deleteTargetTitle string // Titolo dell'elemento da eliminare

	err error
}

type tickMsg time.Time
type notesLoadedMsg []db.NoteListItem
type noteLoadedMsg struct {
	note     *db.Note
	readOnly bool
}
type errMsg error
type syncStartedMsg struct{}
type syncResultMsg struct {
	success       bool
	message       string
	uploadBytes   int64
	downloadBytes int64
}
type versionsLoadedMsg []db.NoteVersion
type onlineCheckMsg bool
type folderLoadedMsg *db.Folder

func NewModel(database *db.DB, enc *crypto.Encryptor, cfg *config.Config) Model {
	t := i18n.T()

	ti := textinput.New()
	ti.Placeholder = t.TitlePlaceholder
	ti.CharLimit = 256

	ta := textarea.New()
	ta.Placeholder = t.NotePlaceholder
	ta.ShowLineNumbers = false

	pi := textinput.New()
	pi.Placeholder = "Password..."
	pi.EchoMode = textinput.EchoPassword
	pi.CharLimit = 256

	var client *api.Client
	if cfg.Server.URL != "" && cfg.Server.Enabled {
		client = api.NewClient(cfg.Server.URL)
		if cfg.Server.Token != "" {
			client.SetToken(cfg.Server.Token)
		}
	}

	m := Model{
		db:            database,
		encryptor:     enc,
		config:        cfg,
		apiClient:     client,
		keys:          NewKeyMap(),
		textinput:     ti,
		textarea:      ta,
		passwordInput: pi,
		activePanel:   PanelList,
		currentFolder: 0,
	}

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.loadNotes(),
		m.tickCmd(),
	}

	if m.apiClient != nil {
		cmds = append(cmds, m.checkOnline())
		// Sync on startup if authenticated
		if m.apiClient.IsAuthenticated() {
			cmds = append(cmds, m.doSync())
		}
	}

	return tea.Batch(cmds...)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		var notes []db.NoteListItem
		var err error

		// Load notes only for current folder level
		if m.currentFolder == 0 {
			// Root level: load only notes without parent_folder_id
			notes, err = m.db.ListNotes()
			if err != nil {
				return errMsg(err)
			}
		} else {
			// Inside folder: load notes for this folder
			notes, err = m.db.ListNotesInFolder(m.currentFolder)
			if err != nil {
				return errMsg(err)
			}
		}

		// Load folders for current folder
		folders, err := m.db.ListFolders(m.currentFolder)
		if err != nil {
			return errMsg(err)
		}

		// Add folders to notes list with @ prefix
		for _, f := range folders {
			folderItem := db.NoteListItem{
				ID:    f.ID,
				Title: "@" + f.Title, // @ prefix indicates folder
				Type:  "folder",
			}
			notes = append(notes, folderItem)
		}

		return notesLoadedMsg(notes)
	}
}

func (m Model) loadNote(id int64) tea.Cmd {
	return func() tea.Msg {
		note, err := m.db.GetNote(id)
		if err != nil {
			return errMsg(err)
		}
		readOnly := false
		if note != nil && m.encryptor != nil && note.Content != "" {
			decrypted, err := m.encryptor.Decrypt(note.Content)
			if err != nil {
				readOnly = true
				note.Content = "[" + i18n.T().EncryptedDifferentKey + "]"
			} else {
				note.Content = decrypted
			}
		}
		return noteLoadedMsg{note: note, readOnly: readOnly}
	}
}

func (m Model) loadNoteVersions(id int64) tea.Cmd {
	return func() tea.Msg {
		versions, err := m.db.GetNoteVersions(id)
		if err != nil {
			return errMsg(err)
		}
		return versionsLoadedMsg(versions)
	}
}

func (m Model) loadFolder(id int64) tea.Cmd {
	return func() tea.Msg {
		folder, err := m.db.GetFolder(id)
		if err != nil {
			return errMsg(err)
		}
		return folderLoadedMsg(folder)
	}
}

func (m Model) checkOnline() tea.Cmd {
	return func() tea.Msg {
		if m.apiClient == nil {
			return onlineCheckMsg(false)
		}
		err := m.apiClient.Ping()
		return onlineCheckMsg(err == nil)
	}
}

func (m Model) doSync() tea.Cmd {
	return func() tea.Msg {
		if m.apiClient == nil || !m.apiClient.IsAuthenticated() {
			return syncResultMsg{success: false, message: i18n.T().Offline}
		}

		result, err := api.Sync(m.db, m.apiClient, m.config.Server.LastSync)
		if err != nil {
			return syncResultMsg{success: false, message: err.Error()}
		}

		msg := fmt.Sprintf("↑%d ↓%d", result.Uploaded, result.Downloaded)
		if len(result.Errors) > 0 {
			return syncResultMsg{success: false, message: msg + " (errori)", uploadBytes: int64(result.Uploaded), downloadBytes: int64(result.Downloaded)}
		}

		return syncResultMsg{success: true, message: msg, uploadBytes: int64(result.Uploaded), downloadBytes: int64(result.Downloaded)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(m.contentWidth() - 4)
		m.textarea.SetHeight(m.contentHeight() - 2)

	case tickMsg:
		if m.dirty && m.mode == ModeEditing {
			cmds = append(cmds, m.saveCurrentNote())
		}
		// Check connection status periodically
		if m.apiClient != nil {
			cmds = append(cmds, m.checkOnline())
			// Sync every 5 minutes (100 ticks of 3 seconds)
			m.tickCount++
			if m.tickCount >= 100 && m.apiClient.IsAuthenticated() && !m.syncing {
				m.tickCount = 0
				cmds = append(cmds, m.doSync())
			}
		}
		cmds = append(cmds, m.tickCmd())

	case notesLoadedMsg:
		m.notes = msg
		// Reset cursor if out of bounds
		if m.cursor >= len(m.notes) {
			m.cursor = 0
			m.listOffset = 0
		}
		// Load first note only if it's not a folder
		if len(m.notes) > 0 && m.currentNote == nil {
			selected := m.currentSelectedItem()
			if selected != nil && selected.Type != "folder" {
				cmds = append(cmds, m.loadNote(selected.ID))
			}
		}

	case noteLoadedMsg:
		m.currentNote = msg.note
		m.currentReadOnly = msg.readOnly
		m.currentFolderData = nil // Clear folder data when loading note
		if msg.note != nil {
			m.textarea.SetValue(msg.note.Content)
		}

	case folderLoadedMsg:
		m.currentFolderData = msg
		m.currentNote = nil // Clear note when viewing folder

	case errMsg:
		m.err = msg

	case syncStartedMsg:
		m.syncing = true
		cmds = append(cmds, m.doSync())

	case onlineCheckMsg:
		m.online = bool(msg)

	case versionsLoadedMsg:
		m.noteVersions = msg

	case syncResultMsg:
		m.syncing = false
		m.syncStatus = msg.message
		m.uploadBytes = msg.uploadBytes
		m.downloadBytes = msg.downloadBytes
		if msg.success {
			m.config.Server.LastSync = time.Now().Unix()
			m.config.Save(config.DefaultConfigPath())
			cmds = append(cmds, m.loadNotes())
		}

	case tea.KeyMsg:
		if m.mode == ModeEditing {
			return m.handleEditingKeys(msg)
		}
		if m.mode == ModeSearch {
			return m.handleSearchKeys(msg)
		}
		if m.mode == ModeNewNote {
			return m.handleNewNoteKeys(msg)
		}
		if m.mode == ModeConfirmDelete {
			return m.handleConfirmDeleteKeys(msg)
		}
		if m.mode == ModeEditTags {
			return m.handleEditTagsKeys(msg)
		}
		if m.mode == ModeSetPassword {
			return m.handleSetPasswordKeys(msg)
		}
		if m.mode == ModeHelp {
			if key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Help) {
				m.mode = ModeNormal
			}
			return m, nil
		}
		if m.mode == ModeHistory {
			return m.handleHistoryKeys(msg)
		}
		return m.handleNormalKeys(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) currentSelectedItem() *db.NoteListItem {
	if len(m.notes) > m.cursor && m.cursor >= 0 {
		return &m.notes[m.cursor]
	}
	return nil
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	t := i18n.T()

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.mode = ModeHelp

	case key.Matches(msg, m.keys.Up):
		if m.activePanel == PanelList && m.cursor > 0 {
			m.cursor--
			if m.cursor < m.listOffset {
				m.listOffset = m.cursor
			}
			// Load note or folder metadata when cursor moves
			selected := m.currentSelectedItem()
			if selected != nil {
				if selected.Type == "folder" {
					return m, m.loadFolder(selected.ID)
				} else {
					return m, m.loadNote(selected.ID)
				}
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.activePanel == PanelList && m.cursor < len(m.notes)-1 {
			m.cursor++
			listHeight := m.contentHeight() - 2
			if m.cursor >= m.listOffset+listHeight {
				m.listOffset = m.cursor - listHeight + 1
			}
			// Load note or folder metadata when cursor moves
			selected := m.currentSelectedItem()
			if selected != nil {
				if selected.Type == "folder" {
					return m, m.loadFolder(selected.ID)
				} else {
					return m, m.loadNote(selected.ID)
				}
			}
		}

	case key.Matches(msg, m.keys.Enter):
		if len(m.notes) > 0 {
			selectedItem := m.notes[m.cursor]
			if selectedItem.Type == "folder" {
				// Navigate into folder
				m.currentFolder = selectedItem.ID
				m.cursor = 0
				m.listOffset = 0
				m.currentNote = nil
				m.currentFolderData = nil
				return m, m.loadNotes()
			} else {
				// Load note
				return m, m.loadNote(selectedItem.ID)
			}
		}

	case key.Matches(msg, m.keys.Edit):
		// Only allow edit if currentNote is not a folder
		selected := m.currentSelectedItem()
		if m.currentNote != nil && !m.currentReadOnly && selected != nil && selected.Type != "folder" {
			m.mode = ModeEditing
			m.textarea.Focus()
		}

	case key.Matches(msg, m.keys.New):
		m.mode = ModeNewNote
		m.textinput.SetValue("")
		m.textinput.Placeholder = "Titolo nota..."
		m.textinput.Focus()
		m.currentItemType = "note"

	case key.Matches(msg, m.keys.NewFolder):
		m.mode = ModeNewNote
		m.textinput.SetValue("")
		m.textinput.Placeholder = "Nome cartella..."
		m.textinput.Focus()
		m.currentItemType = "folder"

	case key.Matches(msg, m.keys.Delete):
		selected := m.currentSelectedItem()
		if selected != nil {
			if selected.Type == "folder" {
				m.deleteTargetID = selected.ID
				m.deleteTargetType = "folder"
				m.deleteTargetTitle = strings.TrimPrefix(selected.Title, "@")
				m.mode = ModeConfirmDelete
			} else if m.currentNote != nil {
				m.deleteTargetID = m.currentNote.ID
				m.deleteTargetType = "note"
				m.deleteTargetTitle = m.currentNote.Title
				m.mode = ModeConfirmDelete
			}
		}

	case key.Matches(msg, m.keys.Search):
		m.mode = ModeSearch
		m.textinput.SetValue(m.searchQuery)
		m.textinput.Placeholder = t.Search + "..."
		m.textinput.Focus()

	case key.Matches(msg, m.keys.Tab):
		m.activePanel = (m.activePanel + 1) % 3

	case key.Matches(msg, m.keys.ShiftTab):
		m.activePanel = (m.activePanel + 2) % 3

	case key.Matches(msg, m.keys.GoToList):
		m.activePanel = PanelList

	case key.Matches(msg, m.keys.Sync):
		if m.apiClient != nil && m.online && !m.syncing {
			m.syncing = true
			m.syncStatus = t.Syncing
			return m, m.doSync()
		}

	case key.Matches(msg, m.keys.History):
		// Only allow history if not a folder
		selected := m.currentSelectedItem()
		if m.currentNote != nil && selected != nil && selected.Type != "folder" {
			m.mode = ModeHistory
			m.versionCursor = 0
			return m, m.loadNoteVersions(m.currentNote.ID)
		}

	case key.Matches(msg, m.keys.EditTags):
		// Only allow edit tags if not a folder
		selected := m.currentSelectedItem()
		if m.currentNote != nil && selected != nil && selected.Type != "folder" {
			m.mode = ModeEditTags
			// Prepend # to each tag for display
			tagsStr := ""
			for _, tag := range m.currentNote.Tags {
				if tagsStr != "" {
					tagsStr += ";"
				}
				tagsStr += "#" + tag
			}
			m.textinput.SetValue(tagsStr)
			m.textinput.Focus()
		}

	case key.Matches(msg, m.keys.SetPassword):
		if m.currentNote != nil {
			m.mode = ModeSetPassword
			m.passwordInput.SetValue("")
			m.passwordInput.Focus()
			m.passwordTarget = m.currentNote.ID
			m.passwordTargetType = "note"
		}

	case key.Matches(msg, m.keys.ParentFolder):
		if m.currentFolder != 0 {
			// Navigate to parent folder
			// For now, go back to root (0). In future, track parent IDs
			m.currentFolder = 0
			m.cursor = 0
			m.listOffset = 0
			m.currentNote = nil
			return m, m.loadNotes()
		}
	}

	return m, nil
}

func (m Model) handleEditingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.GoToList):
		m.mode = ModeNormal
		m.activePanel = PanelList
		m.textarea.Blur()
		return m, m.saveCurrentNote()

	case key.Matches(msg, m.keys.Save):
		return m, m.saveCurrentNote()

	case key.Matches(msg, m.keys.SaveAndClose):
		m.mode = ModeNormal
		m.activePanel = PanelList
		m.textarea.Blur()
		return m, m.saveCurrentNote()

	case key.Matches(msg, m.keys.Tab):
		// Insert tab as spaces (4 spaces)
		m.textarea.InsertString("    ")
		m.dirty = true
		return m, cmd

	default:
		m.textarea, cmd = m.textarea.Update(msg)
		m.dirty = true
	}

	return m, cmd
}

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.textinput.Blur()
		m.searchQuery = ""
		return m, m.loadNotes()

	case key.Matches(msg, m.keys.Enter):
		m.mode = ModeNormal
		m.searchQuery = m.textinput.Value()
		m.textinput.Blur()
		return m, m.searchNotes()

	default:
		m.textinput, cmd = m.textinput.Update(msg)
	}

	return m, cmd
}

func (m Model) handleNewNoteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.textinput.Blur()
		m.textinput.Placeholder = "Titolo..."
		m.currentItemType = ""

	case key.Matches(msg, m.keys.Enter):
		title := m.textinput.Value()
		if title != "" {
			m.mode = ModeNormal
			m.textinput.Blur()
			m.textinput.Placeholder = "Titolo..."

			if m.currentItemType == "folder" {
				m.currentItemType = ""
				return m, m.createFolder(title)
			} else {
				return m, m.createNote(title)
			}
		}

	default:
		m.textinput, cmd = m.textinput.Update(msg)
	}

	return m, cmd
}

func (m Model) handleEditTagsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.textinput.Blur()

	case key.Matches(msg, m.keys.Enter):
		tagsStr := m.textinput.Value()
		m.mode = ModeNormal
		m.textinput.Blur()
		return m, m.saveTags(tagsStr)

	default:
		m.textinput, cmd = m.textinput.Update(msg)
	}

	return m, cmd
}

func (m Model) handleSetPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.passwordInput.Blur()

	case key.Matches(msg, m.keys.Enter):
		password := m.passwordInput.Value()
		m.mode = ModeNormal
		m.passwordInput.Blur()
		return m, m.setPassword(password)

	default:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}

	return m, cmd
}

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = ModeNormal
		if m.deleteTargetType == "folder" {
			return m, m.deleteCurrentFolder()
		}
		return m, m.deleteCurrentNote()
	case "n", "N", "esc":
		m.mode = ModeNormal
		m.deleteTargetID = 0
		m.deleteTargetType = ""
		m.deleteTargetTitle = ""
	}
	return m, nil
}

func (m Model) handleHistoryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.versionCursor > 0 {
			m.versionCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.versionCursor < len(m.noteVersions)-1 {
			m.versionCursor++
		}
	case key.Matches(msg, m.keys.Enter):
		if len(m.noteVersions) > 0 {
			selected := m.noteVersions[m.versionCursor]
			return m, m.restoreNoteVersion(m.currentNote.ID, selected.ID)
		}
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.GoToList):
		m.mode = ModeNormal
		m.activePanel = PanelList
		if m.currentNote != nil {
			return m, m.loadNote(m.currentNote.ID)
		}
	}
	return m, nil
}

func (m Model) restoreNoteVersion(noteID, versionID int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.db.RestoreNoteVersion(noteID, versionID); err != nil {
			return errMsg(err)
		}
		m.mode = ModeNormal
		return m.loadNote(noteID)()
	}
}

func (m Model) saveCurrentNote() tea.Cmd {
	return func() tea.Msg {
		if m.currentNote == nil {
			return nil
		}

		content := m.textarea.Value()
		if m.encryptor != nil {
			encrypted, err := m.encryptor.Encrypt(content)
			if err != nil {
				return errMsg(err)
			}
			content = encrypted
		}

		err := m.db.UpdateNote(m.currentNote.ID, m.currentNote.Title, content, m.currentNote.Tags)
		if err != nil {
			return errMsg(err)
		}

		m.dirty = false
		m.lastSave = time.Now()

		// Trigger sync after saving
		if m.apiClient != nil && m.apiClient.IsAuthenticated() {
			return syncStartedMsg{}
		}
		return nil
	}
}

func (m Model) createNote(title string) tea.Cmd {
	return func() tea.Msg {
		content := ""
		if m.encryptor != nil {
			encrypted, err := m.encryptor.Encrypt(content)
			if err != nil {
				return errMsg(err)
			}
			content = encrypted
		}

		_, err := m.db.CreateNoteInFolder(title, content, []string{}, m.currentFolder)
		if err != nil {
			return errMsg(err)
		}

		// Reload notes in current folder
		return m.loadNotes()()
	}
}

func (m Model) createFolder(title string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.db.CreateFolder(title, m.currentFolder)
		if err != nil {
			return errMsg(err)
		}

		// Reload notes in current folder
		return m.loadNotes()()
	}
}

func (m Model) saveTags(tagsStr string) tea.Cmd {
	return func() tea.Msg {
		if m.currentNote == nil {
			return nil
		}

		// Parse tags: remove # and split by ;
		var tags []string
		for _, t := range strings.Split(tagsStr, ";") {
			t = strings.TrimSpace(t)
			t = strings.TrimPrefix(t, "#")
			if t != "" {
				tags = append(tags, t)
			}
		}

		// Update note with new tags
		err := m.db.UpdateNote(m.currentNote.ID, m.currentNote.Title, m.currentNote.Content, tags)
		if err != nil {
			return errMsg(err)
		}

		// Reload the note to update UI
		return m.loadNote(m.currentNote.ID)()
	}
}

func (m Model) setPassword(password string) tea.Cmd {
	return func() tea.Msg {
		var err error

		switch m.passwordTargetType {
		case "note":
			err = m.db.SetNotePassword(m.passwordTarget, password)
		case "folder":
			err = m.db.SetFolderPassword(m.passwordTarget, password)
		}

		if err != nil {
			return errMsg(err)
		}

		// Reload current note to update UI
		if m.currentNote != nil {
			return m.loadNote(m.currentNote.ID)()
		}
		return nil
	}
}

func (m Model) deleteCurrentNote() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTargetID == 0 {
			return nil
		}

		err := m.db.DeleteNote(m.deleteTargetID)
		if err != nil {
			return errMsg(err)
		}

		return m.loadNotes()()
	}
}

func (m Model) deleteCurrentFolder() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTargetID == 0 {
			return nil
		}

		err := m.db.DeleteFolder(m.deleteTargetID)
		if err != nil {
			return errMsg(err)
		}

		return m.loadNotes()()
	}
}

func (m Model) searchNotes() tea.Cmd {
	return func() tea.Msg {
		notes, err := m.db.SearchNotes(m.searchQuery, m.searchTags)
		if err != nil {
			return errMsg(err)
		}
		return notesLoadedMsg(notes)
	}
}

func (m Model) listWidth() int {
	return int(float64(m.width) * 0.25)
}

func (m Model) contentWidth() int {
	return int(float64(m.width) * 0.50)
}

func (m Model) metadataWidth() int {
	return m.width - m.listWidth() - m.contentWidth()
}

func (m Model) contentHeight() int {
	return m.height - 5
}

func (m Model) View() string {
	t := i18n.T()

	if m.width == 0 {
		return t.Loading
	}

	header := m.renderHeader()
	body := m.renderBody()
	status := m.renderStatus()

	if m.mode == ModeHelp {
		return m.renderHelp()
	}

	if m.mode == ModeHistory {
		return m.renderHistory()
	}

	if m.mode == ModeNewNote || m.mode == ModeSearch {
		dialog := m.renderInputDialog()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.mode == ModeEditTags {
		dialog := m.renderTagsDialog()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.mode == ModeSetPassword {
		dialog := m.renderPasswordDialog()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.mode == ModeConfirmDelete {
		dialog := m.renderConfirmDialog()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m Model) renderHeader() string {
	t := i18n.T()

	title := t.NoNoteSelected
	if m.currentNote != nil {
		title = m.currentNote.Title
	}

	headerContent := TitleStyle.Render(title)
	return HeaderStyle.Width(m.width - 2).Render(headerContent)
}

func (m Model) renderBody() string {
	listPanel := m.renderList()
	contentPanel := m.renderContent()
	metadataPanel := m.renderMetadata()

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, contentPanel, metadataPanel)
}

func (m Model) renderList() string {
	style := PanelStyle
	if m.activePanel == PanelList {
		style = ActivePanelStyle
	}

	var items []string
	listHeight := m.contentHeight() - 2

	for i := m.listOffset; i < len(m.notes) && i < m.listOffset+listHeight; i++ {
		note := m.notes[i]
		// Truncate title to fit with padding
		maxLen := m.listWidth() - 10
		title := truncate(note.Title, maxLen)

		if i == m.cursor {
			// Highlight selected item with full background color
			line := fmt.Sprintf("  ▶ %-*s  ", maxLen, title)
			line = lipgloss.NewStyle().
				Background(highlight).
				Foreground(lipgloss.Color("#000000")).
				Render(line)
			items = append(items, line)
		} else {
			line := fmt.Sprintf("    %-*s  ", maxLen, title)
			items = append(items, line)
		}
	}

	for len(items) < listHeight {
		items = append(items, "")
	}

	content := strings.Join(items, "\n")
	return style.Width(m.listWidth() - 2).Height(m.contentHeight()).Render(content)
}

func (m Model) renderContent() string {
	t := i18n.T()

	style := PanelStyle
	if m.activePanel == PanelContent {
		style = ActivePanelStyle
	}

	var content string
	if m.mode == ModeEditing {
		content = m.textarea.View()
	} else if m.currentNote != nil {
		content = m.currentNote.Content
	} else {
		content = MutedStyle.Render(t.NoNoteSelected)
	}

	return style.Width(m.contentWidth() - 2).Height(m.contentHeight()).Render(content)
}

func (m Model) renderMetadata() string {
	t := i18n.T()

	style := PanelStyle
	if m.activePanel == PanelMetadata {
		style = ActivePanelStyle
	}

	var lines []string

	if m.currentFolderData != nil {
		// Show folder metadata
		lines = append(lines, LabelStyle.Render("📁 Cartella"))
		lines = append(lines, "")

		// Count notes in folder
		count, err := m.db.CountNotesInFolder(m.currentFolderData.ID)
		if err == nil {
			lines = append(lines, LabelStyle.Render("Note"))
			lines = append(lines, MutedStyle.Render("  "+fmt.Sprintf("%d", count)))
		}

		lines = append(lines, "")
		lines = append(lines, LabelStyle.Render(t.CreatedAt))
		lines = append(lines, MutedStyle.Render("  "+m.currentFolderData.CreatedAt.Format("2006-01-02 15:04")))

		lines = append(lines, "")
		lines = append(lines, LabelStyle.Render(t.ModifiedAt))
		lines = append(lines, MutedStyle.Render("  "+m.currentFolderData.UpdatedAt.Format("2006-01-02 15:04")))

		if m.currentFolderData.Password != "" {
			lines = append(lines, "")
			lines = append(lines, LabelStyle.Render("🔒 Protetta"))
		}
	} else if m.currentNote != nil {
		lines = append(lines, LabelStyle.Render(t.Tags))
		if len(m.currentNote.Tags) > 0 {
			for _, tag := range m.currentNote.Tags {
				lines = append(lines, TagStyle.Render("  "+tag))
			}
		} else {
			lines = append(lines, MutedStyle.Render("  "+t.None))
		}

		lines = append(lines, "")
		lines = append(lines, LabelStyle.Render(t.CreatedAt))
		lines = append(lines, MutedStyle.Render("  "+m.currentNote.CreatedAt.Format("2006-01-02 15:04")))

		lines = append(lines, "")
		lines = append(lines, LabelStyle.Render(t.ModifiedAt))
		lines = append(lines, MutedStyle.Render("  "+m.currentNote.UpdatedAt.Format("2006-01-02 15:04")))
	}

	content := strings.Join(lines, "\n")
	return style.Width(m.metadataWidth() - 2).Height(m.contentHeight()).Render(content)
}

func (m Model) renderStatus() string {
	t := i18n.T()

	modeStr := t.ModeNormal
	switch m.mode {
	case ModeEditing:
		modeStr = t.ModeEdit
	case ModeSearch:
		modeStr = t.ModeSearch
	}

	left := fmt.Sprintf(" %s | %d %s", modeStr, len(m.notes), t.Notes)
	if m.currentReadOnly {
		left += " | " + ErrorStyle.Render(t.ReadOnly)
	}

	// Add sync status with visual indicator
	if m.apiClient != nil {
		// Always show online/offline status
		if m.online {
			left += " | " + TagStyle.Render("🟢 "+t.Online)
		} else {
			left += " | " + ErrorStyle.Render("🔴 "+t.Offline)
		}

		// Show syncing status with upload/download indicators
		if m.syncing {
			if m.uploadBytes > 0 || m.downloadBytes > 0 {
				upStr := formatBytes(m.uploadBytes)
				downStr := formatBytes(m.downloadBytes)
				left += " | " + MutedStyle.Render(fmt.Sprintf("↑ %s ↓ %s", upStr, downStr))
			} else {
				left += " | " + MutedStyle.Render("⟳ Syncing...")
			}
		}
	}

	right := fmt.Sprintf("Ctrl+H %s | Ctrl+Q %s", t.Help, t.Exit)
	if m.dirty {
		right = "* " + t.Unsaved + " | " + right
	}

	// Add backspace hint when in a folder
	if m.currentFolder != 0 {
		right = "Backspace ← | " + right
	}

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 0 {
		padding = 0
	}

	return StatusBarStyle.Render(left + strings.Repeat(" ", padding) + right)
}

func (m Model) renderInputDialog() string {
	t := i18n.T()

	title := t.NewNote
	if m.mode == ModeSearch {
		title = t.Search
	} else if m.currentItemType == "folder" {
		title = "Nuova cartella"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		TitleStyle.Render(title),
		"",
		m.textinput.View(),
		"",
		MutedStyle.Render(t.EnterConfirm+"  "+t.EscCancel),
	)

	return DialogStyle.Width(40).Render(content)
}

func (m Model) renderTagsDialog() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		TitleStyle.Render("Tag"),
		"",
		MutedStyle.Render("Esempio: #tag1;#tag2"),
		"",
		m.textinput.View(),
		"",
		MutedStyle.Render("[Enter] Salva  [Esc] Annulla"),
	)

	return DialogStyle.Width(50).Render(content)
}

func (m Model) renderPasswordDialog() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		TitleStyle.Render("Imposta Password"),
		"",
		MutedStyle.Render("Lascia vuoto per rimuovere"),
		"",
		m.passwordInput.View(),
		"",
		MutedStyle.Render("[Enter] Salva  [Esc] Annulla"),
	)

	return DialogStyle.Width(50).Render(content)
}

func (m Model) renderConfirmDialog() string {
	t := i18n.T()

	var title, message string
	if m.deleteTargetType == "folder" {
		title = t.DeleteFolder
		message = fmt.Sprintf(t.DeleteFolderConfirm, m.deleteTargetTitle)
	} else {
		title = t.DeleteNote
		message = fmt.Sprintf(t.DeleteConfirm, m.deleteTargetTitle)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		TitleStyle.Render(title),
		"",
		message,
		"",
		MutedStyle.Render("[Y] "+t.Yes+"  [N] "+t.No),
	)

	return DialogStyle.Width(40).Render(content)
}

func (m Model) renderHelp() string {
	t := i18n.T()

	var b strings.Builder

	// Navigation
	b.WriteString(LabelStyle.Render(t.HelpNavigation) + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "↑/k", t.HelpUp))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "↓/j", t.HelpDown))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Enter", t.HelpOpen))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Tab", t.HelpNextPanel))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Shift+Tab", t.HelpPrevPanel))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+L", t.KeyGoToList))
	b.WriteString("\n")

	// Editing
	b.WriteString(LabelStyle.Render(t.HelpEditing) + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "i", t.HelpEdit))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Esc", t.HelpExitEdit))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+S", t.HelpSave))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+X", t.HelpSaveAndClose))
	b.WriteString("\n")

	// Actions
	b.WriteString(LabelStyle.Render(t.HelpActions) + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+N", t.HelpNew))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "d", t.HelpDelete))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+F", t.HelpSearch))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "h", t.HelpHistory))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "t", t.HelpTags))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "p", t.HelpPassword))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+Y", t.HelpSync))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+E", t.HelpExport))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+I", t.HelpImport))
	b.WriteString("\n")

	// Folders
	b.WriteString(LabelStyle.Render(t.HelpFolders) + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+D", t.HelpNewFolder))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Backspace", t.HelpParentFolder))
	b.WriteString("\n")

	// General
	b.WriteString(LabelStyle.Render(t.HelpGeneral) + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+H/?", t.HelpHelp))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", "Ctrl+Q", t.HelpExit))
	b.WriteString("\n")

	b.WriteString(MutedStyle.Render(t.HelpClose))

	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		Padding(1, 2).
		Align(lipgloss.Left)

	return helpStyle.Render(b.String())
}

func (m Model) renderHistory() string {
	t := i18n.T()
	if len(m.noteVersions) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			ErrorStyle.Render(t.NoVersions))
	}

	header := m.renderHeader()
	historyBody := m.renderHistoryBody()
	footer := m.renderHistoryFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, historyBody, footer)
}

func (m Model) renderHistoryBody() string {
	// Left panel: list of versions
	var items []string
	for i, version := range m.noteVersions {
		line := fmt.Sprintf("%s %s", version.Hash, version.CreatedAt.Format("15:04"))
		if i == m.versionCursor {
			line = SelectedStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		items = append(items, line)
	}

	// Ensure minimum height
	listHeight := m.contentHeight() - 2
	for len(items) < listHeight {
		items = append(items, "")
	}

	listContent := strings.Join(items[:min(len(items), listHeight)], "\n")
	listPanel := PanelStyle.Width(25).Height(m.contentHeight()).Render(listContent)

	// Center panel: preview of selected version
	var previewContent string
	if m.versionCursor < len(m.noteVersions) {
		version := m.noteVersions[m.versionCursor]
		previewContent = version.Content
		// Decrypt if necessary
		if m.encryptor != nil && previewContent != "" {
			decrypted, err := m.encryptor.Decrypt(previewContent)
			if err == nil {
				previewContent = decrypted
			}
		}
	} else {
		previewContent = ""
	}

	previewPanel := PanelStyle.Width(m.width - 30).Height(m.contentHeight()).Render(previewContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, previewPanel)
}

func (m Model) renderHistoryFooter() string {
	t := i18n.T()
	footer := MutedStyle.Render(fmt.Sprintf("[↑/↓] %s  [Enter] %s  [Esc/Ctrl+L] %s", t.HistoryScroll, t.HistoryRestore, t.HistoryBack))
	return "\n" + footer
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
