// Package tui implements the root Bubble Tea model for zburn.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/zburn/internal/identity"
	"github.com/zarlcorp/zburn/internal/store"
)

type viewID int

const (
	viewPassword viewID = iota
	viewMenu
	viewGenerate
	viewList
	viewDetail
)

// Model is the root TUI model.
type Model struct {
	version  string
	dataDir  string
	gen      *identity.Generator
	store    *store.Store
	firstRun bool

	active   viewID
	password passwordModel
	menu     menuModel
	generate generateModel
	list     listModel
	detail   detailModel
}

// New creates the root TUI model.
func New(version, dataDir string, gen *identity.Generator, firstRun bool) Model {
	return Model{
		version:  version,
		dataDir:  dataDir,
		gen:      gen,
		firstRun: firstRun,
		active:   viewPassword,
		password: newPasswordModel(firstRun),
		menu:     newMenuModel(version),
	}
}

func (m Model) Init() tea.Cmd {
	return m.password.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case passwordSubmitMsg:
		return m.openStore(msg.password)

	case navigateMsg:
		return m.navigate(msg.view)

	case quickEmailMsg:
		return m.handleQuickEmail()

	case saveIdentityMsg:
		return m.handleSave(msg.identity)

	case deleteIdentityMsg:
		return m.handleDelete(msg.id)

	case viewIdentityMsg:
		m.detail = newDetailModel(msg.identity)
		m.active = viewDetail
		return m, nil
	}

	return m.updateActive(msg)
}

func (m Model) View() string {
	switch m.active {
	case viewPassword:
		return m.password.View()
	case viewMenu:
		return m.menu.View()
	case viewGenerate:
		return m.generate.View()
	case viewList:
		return m.list.View()
	case viewDetail:
		return m.detail.View()
	}
	return ""
}

func (m Model) updateActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.active {
	case viewPassword:
		m.password, cmd = m.password.Update(msg)
	case viewMenu:
		m.menu, cmd = m.menu.Update(msg)
	case viewGenerate:
		m.generate, cmd = m.generate.Update(msg)
	case viewList:
		m.list, cmd = m.list.Update(msg)
	case viewDetail:
		m.detail, cmd = m.detail.Update(msg)
	}

	return m, cmd
}

func (m Model) openStore(password string) (tea.Model, tea.Cmd) {
	if err := os.MkdirAll(m.dataDir, 0o700); err != nil {
		m.password, _ = m.password.Update(passwordErrMsg{
			err: fmt.Errorf("create data dir: %w", err),
		})
		return m, nil
	}

	fsys := zfilesystem.NewOSFileSystem(m.dataDir)
	s, err := store.Open(fsys, password)
	if err != nil {
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	m.store = s
	m.active = viewMenu
	return m, nil
}

func (m Model) navigate(view viewID) (tea.Model, tea.Cmd) {
	switch view {
	case viewMenu:
		m.menu = newMenuModel(m.version)
		m.active = viewMenu
		return m, nil

	case viewGenerate:
		id := m.gen.Generate()
		m.generate = newGenerateModel(id)
		m.active = viewGenerate
		return m, nil

	case viewList:
		return m.loadList()

	case viewDetail:
		// detail is set by viewIdentityMsg, not navigateMsg
		m.active = viewDetail
		return m, nil
	}

	return m, nil
}

func (m Model) loadList() (tea.Model, tea.Cmd) {
	ids, err := m.store.List()
	if err != nil {
		// show empty list with error flash
		m.list = newListModel(nil)
		m.list.flash = "load: " + err.Error()
		m.active = viewList
		return m, clearFlashAfter()
	}

	m.list = newListModel(ids)
	m.active = viewList
	return m, nil
}

func (m Model) handleQuickEmail() (tea.Model, tea.Cmd) {
	email := m.gen.Email()
	if err := copyToClipboard(email); err != nil {
		// fall through to menu with no notification
		return m, nil
	}

	// show a brief flash on menu
	m.menu = newMenuModel(m.version)
	return m, func() tea.Msg {
		return flashMsg{}
	}
}

func (m Model) handleSave(id identity.Identity) (tea.Model, tea.Cmd) {
	if err := m.store.Save(id); err != nil {
		m.generate.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.generate, _ = m.generate.Update(identitySavedMsg{})
	return m, clearFlashAfter()
}

func (m Model) handleDelete(id string) (tea.Model, tea.Cmd) {
	if err := m.store.Delete(id); err != nil {
		if m.active == viewDetail {
			m.detail.flash = "delete: " + err.Error()
			return m, clearFlashAfter()
		}
		m.list.flash = "delete: " + err.Error()
		return m, clearFlashAfter()
	}

	if m.active == viewDetail {
		// go back to list after deleting from detail
		return m.loadList()
	}

	// reload list after delete
	return m.loadList()
}

// Close cleans up resources. Call after the program exits.
func (m Model) Close() {
	if m.store != nil {
		m.store.Close()
	}
}

