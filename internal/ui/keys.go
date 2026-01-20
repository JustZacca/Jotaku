package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/JustZacca/jotaku/internal/i18n"
)

type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Edit         key.Binding
	Escape       key.Binding
	Save         key.Binding
	SaveAndClose key.Binding
	New          key.Binding
	NewFolder    key.Binding
	Delete       key.Binding
	Search       key.Binding
	Export       key.Binding
	Import       key.Binding
	Quit         key.Binding
	Help         key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	GoToList     key.Binding
	Sync         key.Binding
	History      key.Binding
	EditTags     key.Binding
	SetPassword  key.Binding
	ParentFolder key.Binding
	Copy         key.Binding
}

func NewKeyMap() KeyMap {
	t := i18n.T()
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", t.KeyUp),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", t.KeyDown),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", t.KeyEnter),
		),
		Edit: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", t.KeyEdit),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", t.KeyEscape),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("Ctrl+S", t.KeySave),
		),
		SaveAndClose: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("Ctrl+X", t.KeySaveAndClose),
		),
		New: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("Ctrl+N", t.KeyNew),
		),
		NewFolder: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("Ctrl+D", t.KeyNewFolder),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", t.KeyDelete),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("Ctrl+F", t.KeySearch),
		),
		Export: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("Ctrl+E", t.KeyExport),
		),
		Import: key.NewBinding(
			key.WithKeys("ctrl+i"),
			key.WithHelp("Ctrl+I", t.KeyImport),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("Ctrl+Q", t.KeyQuit),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h", "?"),
			key.WithHelp("Ctrl+H/?", t.KeyHelp),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", t.KeyTab),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", t.KeyShiftTab),
		),
		GoToList: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("Ctrl+L", t.KeyGoToList),
		),
		Sync: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("Ctrl+Y", t.KeySync),
		),
		History: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", t.KeyHistory),
		),
		EditTags: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", t.KeyTags),
		),
		SetPassword: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", t.KeyPassword),
		),
		ParentFolder: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("Backspace", t.KeyParentFolder),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", t.KeyCopy),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Edit, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Edit, k.Escape},
		{k.New, k.NewFolder, k.Delete, k.Save, k.Search},
		{k.History, k.EditTags, k.SetPassword, k.Sync, k.Copy},
		{k.Export, k.Import, k.Help, k.Quit},
	}
}
