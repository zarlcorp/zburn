package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/identity"
)

// listModel displays saved identities in a scrollable list.
type listModel struct {
	identities []identity.Identity
	cursor     int
	confirming bool // delete confirmation
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
		m.confirming = false
		return m, nil

	case identityDeletedMsg:
		m.flash = "deleted"
		m.confirming = false
		// reload will be handled by root
		return m, nil

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m listModel) handleKey(msg tea.KeyMsg) (listModel, tea.Cmd) {
	// handle delete confirmation first
	if m.confirming {
		return m.handleConfirm(msg)
	}

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
		m.confirming = true
		return m, nil
	}

	return m, nil
}

func (m listModel) handleConfirm(msg tea.KeyMsg) (listModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		id := m.identities[m.cursor].ID
		m.confirming = false
		return m, func() tea.Msg { return deleteIdentityMsg{id: id} }
	default:
		m.confirming = false
		return m, nil
	}
}

func (m listModel) View() string {
	title := zstyle.Title.Render("saved identities")
	s := fmt.Sprintf("\n  %s\n\n", title)

	if len(m.identities) == 0 {
		s += "  " + zstyle.MutedText.Render("no saved identities") + "\n"
		s += "\n  " + zstyle.MutedText.Render("esc back  q quit") + "\n\n"
		return s
	}

	// header
	header := fmt.Sprintf("  %-10s %-20s %-30s %s", "id", "name", "email", "created")
	s += zstyle.Subtitle.Render(header) + "\n"

	for i, id := range m.identities {
		name := id.FirstName + " " + id.LastName
		line := fmt.Sprintf("  %-10s %-20s %-30s %s",
			id.ID, truncate(name, 18), truncate(id.Email, 28), id.CreatedAt.Format("2006-01-02"))

		if i == m.cursor {
			s += zstyle.Highlight.Render("> "+line) + "\n"
		} else {
			s += "  " + line + "\n"
		}
	}

	s += "\n"

	if m.confirming {
		s += "  " + zstyle.StatusWarn.Render("delete? y/n") + "\n\n"
	} else if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n\n"
	}

	help := "j/k navigate  enter view  d delete  esc back  q quit"
	s += "  " + zstyle.MutedText.Render(help) + "\n\n"
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
