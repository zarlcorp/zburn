package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

// credentialListModel displays credentials for a single identity.
type credentialListModel struct {
	identity    identity.Identity
	credentials []credential.Credential
	cursor      int
	flash       string
	confirm     bool
}

// viewCredentialMsg requests viewing a specific credential.
type viewCredentialMsg struct {
	credential credential.Credential
}

// deleteCredentialMsg requests deletion of a credential.
type deleteCredentialMsg struct {
	id string
}

// addCredentialMsg requests adding a new credential for the identity.
type addCredentialMsg struct {
	identity identity.Identity
}

// editCredentialMsg requests editing an existing credential.
type editCredentialMsg struct {
	credential credential.Credential
}

func newCredentialListModel(id identity.Identity, creds []credential.Credential) credentialListModel {
	sort.Slice(creds, func(i, j int) bool {
		return creds[i].Label < creds[j].Label
	})
	return credentialListModel{
		identity:    id,
		credentials: creds,
	}
}

func (m credentialListModel) Init() tea.Cmd {
	return nil
}

func (m credentialListModel) Update(msg tea.Msg) (credentialListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m credentialListModel) handleKey(msg tea.KeyMsg) (credentialListModel, tea.Cmd) {
	if m.confirm {
		return m.handleConfirm(msg)
	}

	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		return m, func() tea.Msg { return navigateMsg{view: viewDetail} }
	}

	if key.Matches(msg, zstyle.KeyUp) {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyDown) {
		if len(m.credentials) > 0 && m.cursor < len(m.credentials)-1 {
			m.cursor++
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyEnter) {
		if len(m.credentials) == 0 {
			return m, nil
		}
		c := m.credentials[m.cursor]
		return m, func() tea.Msg { return viewCredentialMsg{credential: c} }
	}

	switch msg.String() {
	case "a":
		id := m.identity
		return m, func() tea.Msg { return addCredentialMsg{identity: id} }

	case "d":
		if len(m.credentials) == 0 {
			return m, nil
		}
		m.confirm = true
		return m, nil
	}

	return m, nil
}

func (m credentialListModel) handleConfirm(msg tea.KeyMsg) (credentialListModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		id := m.credentials[m.cursor].ID
		m.confirm = false
		return m, func() tea.Msg { return deleteCredentialMsg{id: id} }
	default:
		m.confirm = false
		return m, nil
	}
}

func (m credentialListModel) View() string {
	title := zstyle.Title.Render(fmt.Sprintf("credentials (%d)", len(m.credentials)))
	s := fmt.Sprintf("\n  %s\n\n", title)

	if len(m.credentials) == 0 {
		s += "  " + zstyle.MutedText.Render("no credentials") + "\n"
		s += "\n"
		s += "\n"
		s += "  " + zstyle.MutedText.Render("a add  esc back  q quit") + "\n"
		return s
	}

	for i, c := range m.credentials {
		line := fmt.Sprintf("  %-20s %-30s %s",
			truncate(c.Label, 18), truncate(c.Username, 28), truncate(c.URL, 40))

		if i == m.cursor {
			s += zstyle.Highlight.Render("> " + line) + "\n"
		} else {
			s += "  " + line + "\n"
		}
	}

	s += "\n"

	if m.confirm {
		label := m.credentials[m.cursor].Label
		s += "  " + zstyle.StatusWarn.Render(fmt.Sprintf("delete credential %q? this cannot be undone. (y/n)", label)) + "\n"
	} else if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "j/k navigate  enter view  a add  d delete  esc back  q quit"
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}
