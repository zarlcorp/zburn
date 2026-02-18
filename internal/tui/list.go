package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/identity"
)

// listModel displays saved identities in a scrollable list.
type listModel struct {
	identities []identity.Identity
	credCounts map[string]int
	cursor     int
	flash      string
}

// loadIdentitiesMsg carries identities loaded from the store.
type loadIdentitiesMsg struct {
	identities []identity.Identity
}

// loadIdentitiesErrMsg signals a store error.
type loadIdentitiesErrMsg struct {
	err error
}

// deleteIdentityMsg requests deletion of an identity.
type deleteIdentityMsg struct {
	id string
}

// identityDeletedMsg confirms deletion.
type identityDeletedMsg struct{}

// viewIdentityMsg requests viewing a specific identity.
type viewIdentityMsg struct {
	identity identity.Identity
}

func newListModel(ids []identity.Identity) listModel {
	return listModel{identities: ids}
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case loadIdentitiesMsg:
		m.identities = msg.identities
		m.cursor = 0
		return m, nil

	case identityDeletedMsg:
		m.flash = "deleted"
		// reload will be handled by root
		return m, nil

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m listModel) handleKey(msg tea.KeyMsg) (listModel, tea.Cmd) {
	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		return m, func() tea.Msg { return navigateMsg{view: viewMenu} }
	}

	if len(m.identities) == 0 {
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyUp) {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyDown) {
		if m.cursor < len(m.identities)-1 {
			m.cursor++
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyEnter) {
		id := m.identities[m.cursor]
		return m, func() tea.Msg { return viewIdentityMsg{identity: id} }
	}

	if msg.String() == "d" {
		id := m.identities[m.cursor]
		return m, func() tea.Msg { return burnStartMsg{identity: id} }
	}

	return m, nil
}

func (m listModel) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(zstyle.ZburnAccent).Bold(true)

	s := "\n"

	if len(m.identities) == 0 {
		s += "  " + zstyle.MutedText.Render("no saved identities") + "\n"
		s += "\n"
		// reserved flash line (empty for empty state)
		s += "\n"
		return s
	}

	for i, id := range m.identities {
		name := truncate(id.FirstName+" "+id.LastName, 20)
		email := truncate(id.Email, 30)
		line := fmt.Sprintf("%-20s %-30s", name, email)

		if n := m.credCounts[id.ID]; n > 0 {
			line += "  " + zstyle.MutedText.Render(fmt.Sprintf("(%d)", n))
		}

		if i == m.cursor {
			s += "  " + accentStyle.Render("▸") + " " + line + "\n"
		} else {
			s += "    " + line + "\n"
		}
	}

	s += "\n"

	// always reserve a line for flash to prevent layout shift
	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
