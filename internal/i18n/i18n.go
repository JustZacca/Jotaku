package i18n

type Language string

const (
	Italian Language = "it"
	English Language = "en"
)

var currentLang = Italian

type Messages struct {
	// General
	Loading    string
	Error      string
	Confirm    string
	Cancel     string
	Yes        string
	No         string
	None       string
	Unsaved    string
	Notes      string
	Note       string
	Help       string
	Exit       string
	Folder     string
	Folders    string
	Protected  string

	// Modes
	ModeNormal  string
	ModeEdit    string
	ModeSearch  string
	ModeHistory string

	// Panels
	NoNoteSelected string

	// Metadata
	Tags       string
	CreatedAt  string
	ModifiedAt string
	NotesCount string

	// Dialogs
	NewNote             string
	NewFolder           string
	DeleteNote          string
	DeleteFolder        string
	DeleteConfirm       string
	DeleteFolderConfirm string
	Search              string
	NotePlaceholder     string
	TitlePlaceholder    string
	FolderPlaceholder   string
	SetPassword         string
	PasswordPlaceholder string
	PasswordRemoveHint  string
	EditTags            string
	TagsExample         string

	// Actions
	EnterConfirm string
	EscCancel    string
	SaveAction   string

	// Help sections
	HelpNavigation string
	HelpEditing    string
	HelpActions    string
	HelpFolders    string
	HelpGeneral    string

	// Help descriptions
	HelpUp           string
	HelpDown         string
	HelpOpen         string
	HelpNextPanel    string
	HelpPrevPanel    string
	HelpEdit         string
	HelpExitEdit     string
	HelpSave         string
	HelpSaveAndClose string
	HelpNew          string
	HelpNewFolder    string
	HelpDelete       string
	HelpSearch       string
	HelpExport       string
	HelpImport       string
	HelpSync         string
	HelpHistory      string
	HelpTags         string
	HelpPassword     string
	HelpParentFolder string
	HelpHelp         string
	HelpExit         string
	HelpClose        string

	// Keys descriptions (short)
	KeyUp           string
	KeyDown         string
	KeyEnter        string
	KeyEdit         string
	KeyEscape       string
	KeySave         string
	KeySaveAndClose string
	KeyNew          string
	KeyNewFolder    string
	KeyDelete       string
	KeySearch       string
	KeyExport       string
	KeyImport       string
	KeyQuit         string
	KeyHelp         string
	KeyTab          string
	KeyShiftTab     string
	KeyGoToList     string
	KeySync         string
	KeyHistory      string
	KeyTags         string
	KeyPassword     string
	KeyParentFolder string
	KeyCopy         string

	// Prompts
	MasterPassword string

	// Errors/Warnings
	EncryptedDifferentKey string
	ReadOnly              string
	NoVersions            string

	// Clipboard
	Copied      string
	CopyError   string

	// Sync
	Syncing     string
	Synced      string
	Offline     string
	Online      string
	SyncError   string
	SyncSuccess string
	Pending     string

	// History
	HistoryTitle   string
	HistoryRestore string
	HistoryScroll  string
	HistoryBack    string
}

var translations = map[Language]Messages{
	Italian: {
		// General
		Loading:   "Caricamento...",
		Error:     "Errore",
		Confirm:   "Conferma",
		Cancel:    "Annulla",
		Yes:       "Sì",
		No:        "No",
		None:      "nessuno",
		Unsaved:   "Non salvato",
		Notes:     "note",
		Note:      "nota",
		Help:      "Aiuto",
		Exit:      "Esci",
		Folder:    "Cartella",
		Folders:   "cartelle",
		Protected: "Protetta",

		// Modes
		ModeNormal:  "NORMALE",
		ModeEdit:    "MODIFICA",
		ModeSearch:  "CERCA",
		ModeHistory: "STORICO",

		// Panels
		NoNoteSelected: "Nessuna nota selezionata",

		// Metadata
		Tags:       "Tags:",
		CreatedAt:  "Creata:",
		ModifiedAt: "Modificata:",
		NotesCount: "Note:",

		// Dialogs
		NewNote:             "Nuova Nota",
		NewFolder:           "Nuova Cartella",
		DeleteNote:          "Elimina Nota",
		DeleteFolder:        "Elimina Cartella",
		DeleteConfirm:       "Eliminare '%s'?",
		DeleteFolderConfirm: "Eliminare la cartella '%s'?",
		Search:              "Cerca",
		NotePlaceholder:     "Scrivi qui...",
		TitlePlaceholder:    "Titolo nota...",
		FolderPlaceholder:   "Nome cartella...",
		SetPassword:         "Imposta Password",
		PasswordPlaceholder: "Password...",
		PasswordRemoveHint:  "Lascia vuoto per rimuovere",
		EditTags:            "Modifica Tag",
		TagsExample:         "Esempio: #tag1;#tag2",

		// Actions
		EnterConfirm: "[Enter] Conferma",
		EscCancel:    "[Esc] Annulla",
		SaveAction:   "Salva",

		// Help sections
		HelpNavigation: "NAVIGAZIONE",
		HelpEditing:    "EDITING",
		HelpActions:    "AZIONI",
		HelpFolders:    "CARTELLE",
		HelpGeneral:    "GENERALE",

		// Help descriptions
		HelpUp:           "Su",
		HelpDown:         "Giù",
		HelpOpen:         "Apri nota/cartella",
		HelpNextPanel:    "Pannello successivo",
		HelpPrevPanel:    "Pannello precedente",
		HelpEdit:         "Modifica nota",
		HelpExitEdit:     "Esci dalla modifica",
		HelpSave:         "Salva",
		HelpSaveAndClose: "Salva e chiudi",
		HelpNew:          "Nuova nota",
		HelpNewFolder:    "Nuova cartella",
		HelpDelete:       "Elimina nota/cartella",
		HelpSearch:       "Cerca",
		HelpExport:       "Esporta in Markdown",
		HelpImport:       "Importa Markdown",
		HelpSync:         "Sincronizza con server",
		HelpHistory:      "Storico versioni",
		HelpTags:         "Modifica tag",
		HelpPassword:     "Imposta password",
		HelpParentFolder: "Cartella superiore",
		HelpHelp:         "Mostra aiuto",
		HelpExit:         "Esci",
		HelpClose:        "Premi Esc o Ctrl+H per chiudere",

		// Keys descriptions (short)
		KeyUp:           "su",
		KeyDown:         "giù",
		KeyEnter:        "apri",
		KeyEdit:         "modifica",
		KeyEscape:       "esci/annulla",
		KeySave:         "salva",
		KeySaveAndClose: "salva e chiudi",
		KeyNew:          "nuova nota",
		KeyNewFolder:    "nuova cartella",
		KeyDelete:       "elimina",
		KeySearch:       "cerca",
		KeyExport:       "esporta",
		KeyImport:       "importa",
		KeyQuit:         "esci",
		KeyHelp:         "aiuto",
		KeyTab:          "pannello succ.",
		KeyShiftTab:     "pannello prec.",
		KeyGoToList:     "vai alla lista",
		KeySync:         "sincronizza",
		KeyHistory:      "storico",
		KeyTags:         "tag",
		KeyPassword:     "password",
		KeyParentFolder: "indietro",
		KeyCopy:         "copia",

		// Prompts
		MasterPassword: "Password master: ",

		// Errors/Warnings
		EncryptedDifferentKey: "Cifrata con altra chiave",
		ReadOnly:              "Sola lettura",
		NoVersions:            "Nessuna versione disponibile",

		// Clipboard
		Copied:    "Copiato nella clipboard",
		CopyError: "Errore copia",

		// Sync
		Syncing:     "Sincronizzazione...",
		Synced:      "Sincronizzato",
		Offline:     "Offline",
		Online:      "Online",
		SyncError:   "Errore sync",
		SyncSuccess: "Sync completato",
		Pending:     "In attesa",

		// History
		HistoryTitle:   "Storico Versioni",
		HistoryRestore: "Ripristina",
		HistoryScroll:  "Scorri",
		HistoryBack:    "Lista",
	},

	English: {
		// General
		Loading:   "Loading...",
		Error:     "Error",
		Confirm:   "Confirm",
		Cancel:    "Cancel",
		Yes:       "Yes",
		No:        "No",
		None:      "none",
		Unsaved:   "Unsaved",
		Notes:     "notes",
		Note:      "note",
		Help:      "Help",
		Exit:      "Exit",
		Folder:    "Folder",
		Folders:   "folders",
		Protected: "Protected",

		// Modes
		ModeNormal:  "NORMAL",
		ModeEdit:    "EDIT",
		ModeSearch:  "SEARCH",
		ModeHistory: "HISTORY",

		// Panels
		NoNoteSelected: "No note selected",

		// Metadata
		Tags:       "Tags:",
		CreatedAt:  "Created:",
		ModifiedAt: "Modified:",
		NotesCount: "Notes:",

		// Dialogs
		NewNote:             "New Note",
		NewFolder:           "New Folder",
		DeleteNote:          "Delete Note",
		DeleteFolder:        "Delete Folder",
		DeleteConfirm:       "Delete '%s'?",
		DeleteFolderConfirm: "Delete folder '%s'?",
		Search:              "Search",
		NotePlaceholder:     "Write here...",
		TitlePlaceholder:    "Note title...",
		FolderPlaceholder:   "Folder name...",
		SetPassword:         "Set Password",
		PasswordPlaceholder: "Password...",
		PasswordRemoveHint:  "Leave empty to remove",
		EditTags:            "Edit Tags",
		TagsExample:         "Example: #tag1;#tag2",

		// Actions
		EnterConfirm: "[Enter] Confirm",
		EscCancel:    "[Esc] Cancel",
		SaveAction:   "Save",

		// Help sections
		HelpNavigation: "NAVIGATION",
		HelpEditing:    "EDITING",
		HelpActions:    "ACTIONS",
		HelpFolders:    "FOLDERS",
		HelpGeneral:    "GENERAL",

		// Help descriptions
		HelpUp:           "Up",
		HelpDown:         "Down",
		HelpOpen:         "Open note/folder",
		HelpNextPanel:    "Next panel",
		HelpPrevPanel:    "Previous panel",
		HelpEdit:         "Edit note",
		HelpExitEdit:     "Exit edit mode",
		HelpSave:         "Save",
		HelpSaveAndClose: "Save and close",
		HelpNew:          "New note",
		HelpNewFolder:    "New folder",
		HelpDelete:       "Delete note/folder",
		HelpSearch:       "Search",
		HelpExport:       "Export to Markdown",
		HelpImport:       "Import Markdown",
		HelpSync:         "Sync with server",
		HelpHistory:      "Version history",
		HelpTags:         "Edit tags",
		HelpPassword:     "Set password",
		HelpParentFolder: "Parent folder",
		HelpHelp:         "Show help",
		HelpExit:         "Exit",
		HelpClose:        "Press Esc or Ctrl+H to close",

		// Keys descriptions (short)
		KeyUp:           "up",
		KeyDown:         "down",
		KeyEnter:        "open",
		KeyEdit:         "edit",
		KeyEscape:       "exit/cancel",
		KeySave:         "save",
		KeySaveAndClose: "save & close",
		KeyNew:          "new note",
		KeyNewFolder:    "new folder",
		KeyDelete:       "delete",
		KeySearch:       "search",
		KeyExport:       "export",
		KeyImport:       "import",
		KeyQuit:         "quit",
		KeyHelp:         "help",
		KeyTab:          "next panel",
		KeyShiftTab:     "prev panel",
		KeyGoToList:     "go to list",
		KeySync:         "sync",
		KeyHistory:      "history",
		KeyTags:         "tags",
		KeyPassword:     "password",
		KeyParentFolder: "back",
		KeyCopy:         "copy",

		// Prompts
		MasterPassword: "Master password: ",

		// Errors/Warnings
		EncryptedDifferentKey: "Encrypted with different key",
		ReadOnly:              "Read only",
		NoVersions:            "No versions available",

		// Clipboard
		Copied:    "Copied to clipboard",
		CopyError: "Copy error",

		// Sync
		Syncing:     "Syncing...",
		Synced:      "Synced",
		Offline:     "Offline",
		Online:      "Online",
		SyncError:   "Sync error",
		SyncSuccess: "Sync completed",
		Pending:     "Pending",

		// History
		HistoryTitle:   "Version History",
		HistoryRestore: "Restore",
		HistoryScroll:  "Scroll",
		HistoryBack:    "List",
	},
}

func SetLanguage(lang Language) {
	if _, ok := translations[lang]; ok {
		currentLang = lang
	}
}

func GetLanguage() Language {
	return currentLang
}

func T() Messages {
	return translations[currentLang]
}
