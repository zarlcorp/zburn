// Package tui implements the root Bubble Tea model for zburn.
package tui

import (
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

type viewID int

const (
	viewPassword viewID = iota
	viewMenu
	viewGenerate
	viewList
	viewDetail
	viewCredentialList
	viewCredentialDetail
	viewCredentialForm
)

// Model is the root TUI model.
type Model struct {
	version    string
	dataDir    string
	gen        *identity.Generator
	store      *zstore.Store
	identities *zstore.Collection[identity.Identity]
	credentials *zstore.Collection[credential.Credential]
	firstRun   bool

	active           viewID
	password         passwordModel
	menu             menuModel
	generate         generateModel
	list             listModel
	detail           detailModel
	credentialList   credentialListModel
	credentialDetail credentialDetailModel
	credentialForm   credentialFormModel
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
		return m.handleViewIdentity(msg.identity)

	case viewCredentialsMsg:
		return m.loadCredentialList(msg.identityID)

	case viewCredentialMsg:
		m.credentialDetail = newCredentialDetailModel(msg.credential)
		m.active = viewCredentialDetail
		return m, m.credentialDetail.Init()

	case addCredentialMsg:
		m.credentialForm = newCredentialFormModel(msg.identityID, nil)
		m.active = viewCredentialForm
		return m, m.credentialForm.Init()

	case editCredentialMsg:
		c := msg.credential
		m.credentialForm = newCredentialFormModel(c.IdentityID, &c)
		m.active = viewCredentialForm
		return m, m.credentialForm.Init()

	case saveCredentialMsg:
		return m.handleSaveCredential(msg.credential)

	case deleteCredentialMsg:
		return m.handleDeleteCredential(msg.id)
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
	case viewCredentialList:
		return m.credentialList.View()
	case viewCredentialDetail:
		return m.credentialDetail.View()
	case viewCredentialForm:
		return m.credentialForm.View()
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
	case viewCredentialList:
		m.credentialList, cmd = m.credentialList.Update(msg)
	case viewCredentialDetail:
		m.credentialDetail, cmd = m.credentialDetail.Update(msg)
	case viewCredentialForm:
		m.credentialForm, cmd = m.credentialForm.Update(msg)
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
	s, err := zstore.Open(fsys, []byte(password))
	if err != nil {
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	idCol, err := zstore.NewCollection[identity.Identity](s, "identities")
	if err != nil {
		s.Close()
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	credCol, err := zstore.NewCollection[credential.Credential](s, "credentials")
	if err != nil {
		s.Close()
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	m.store = s
	m.identities = idCol
	m.credentials = credCol
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

	case viewCredentialList:
		return m.loadCredentialList(m.credentialList.identityID)
	}

	return m, nil
}

func (m Model) loadList() (tea.Model, tea.Cmd) {
	ids, err := m.identities.List()
	if err != nil {
		// show empty list with error flash
		m.list = newListModel(nil)
		m.list.flash = "load: " + err.Error()
		m.active = viewList
		return m, clearFlashAfter()
	}

	// sort by CreatedAt descending — zstore.List does not guarantee order
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].CreatedAt.After(ids[j].CreatedAt)
	})

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
	if err := m.identities.Put(id.ID, id); err != nil {
		m.generate.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.generate, _ = m.generate.Update(identitySavedMsg{})
	return m, clearFlashAfter()
}

func (m Model) handleDelete(id string) (tea.Model, tea.Cmd) {
	if err := m.identities.Delete(id); err != nil {
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

func (m Model) handleViewIdentity(id identity.Identity) (tea.Model, tea.Cmd) {
	m.detail = newDetailModel(id)

	// count credentials for this identity
	if m.credentials != nil {
		count, err := m.countCredentials(id.ID)
		if err == nil {
			m.detail.credentialCount = count
		}
	}

	m.active = viewDetail
	return m, nil
}

func (m Model) countCredentials(identityID string) (int, error) {
	all, err := m.credentials.List()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range all {
		if c.IdentityID == identityID {
			count++
		}
	}
	return count, nil
}

func (m Model) loadCredentialList(identityID string) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		m.credentialList = newCredentialListModel(identityID, nil)
		m.active = viewCredentialList
		return m, nil
	}

	all, err := m.credentials.List()
	if err != nil {
		m.credentialList = newCredentialListModel(identityID, nil)
		m.credentialList.flash = "load: " + err.Error()
		m.active = viewCredentialList
		return m, clearFlashAfter()
	}

	// filter by identity
	var creds []credential.Credential
	for _, c := range all {
		if c.IdentityID == identityID {
			creds = append(creds, c)
		}
	}

	m.credentialList = newCredentialListModel(identityID, creds)
	m.active = viewCredentialList
	return m, nil
}

func (m Model) handleSaveCredential(c credential.Credential) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		return m, nil
	}

	if err := m.credentials.Put(c.ID, c); err != nil {
		m.credentialForm.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	// after save, go to credential detail
	m.credentialDetail = newCredentialDetailModel(c)
	m.active = viewCredentialDetail
	return m, m.credentialDetail.Init()
}

func (m Model) handleDeleteCredential(id string) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		return m, nil
	}

	// remember the identity ID before deleting
	identityID := ""
	if m.active == viewCredentialDetail {
		identityID = m.credentialDetail.credential.IdentityID
	} else if m.active == viewCredentialList {
		identityID = m.credentialList.identityID
	}

	if err := m.credentials.Delete(id); err != nil {
		if m.active == viewCredentialDetail {
			m.credentialDetail.flash = "delete: " + err.Error()
			return m, clearFlashAfter()
		}
		m.credentialList.flash = "delete: " + err.Error()
		return m, clearFlashAfter()
	}

	// go back to credential list
	if identityID != "" {
		return m.loadCredentialList(identityID)
	}
	return m, nil
}

// Close cleans up resources. Call after the program exits.
func (m Model) Close() {
	if m.store != nil {
		m.store.Close()
	}
}
