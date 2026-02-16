package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/identity"
)

// viewCredentialsMsg requests viewing credentials for an identity.
type viewCredentialsMsg struct {
	identityID string
}

// detailModel displays all fields of a saved identity.
type detailModel struct {
	identity       identity.Identity
	fields         []identityField
	cursor         int
	flash          string
	confirm        bool
	credentialCount int
}

func newDetailModel(id identity.Identity) detailModel {
	return detailModel{
		identity: id,
		fields:   identityFields(id),
	}
}

func (m detailModel) Init() tea.Cmd {
	return nil
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case identityDeletedMsg:
		return m, func() tea.Msg { return navigateMsg{view: viewList} }

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m detailModel) handleKey(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	if m.confirm {
		return m.handleConfirm(msg)
	}

	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		return m, func() tea.Msg { return navigateMsg{view: viewList} }
	}

	if key.Matches(msg, zstyle.KeyUp) {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyDown) {
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}
		return m, nil
	}

	if key.Matches(msg, zstyle.KeyEnter) {
		val := m.fields[m.cursor].value
		if err := copyToClipboard(val); err != nil {
			m.flash = "copy: " + err.Error()
			return m, clearFlashAfter()
		}
		m.flash = "copied!"
		return m, clearFlashAfter()
	}

	switch msg.String() {
	case "c":
		all := m.allFieldsText()
		if err := copyToClipboard(all); err != nil {
			m.flash = "copy: " + err.Error()
			return m, clearFlashAfter()
		}
		m.flash = "copied all!"
		return m, clearFlashAfter()

	case "w":
		id := m.identity.ID
		return m, func() tea.Msg { return viewCredentialsMsg{identityID: id} }

	case "d":
		m.confirm = true
		return m, nil
	}

	return m, nil
}

func (m detailModel) handleConfirm(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		id := m.identity.ID
		m.confirm = false
		return m, func() tea.Msg { return deleteIdentityMsg{id: id} }
	default:
		m.confirm = false
		return m, nil
	}
}

func (m detailModel) allFieldsText() string {
	var b strings.Builder
	for _, f := range m.fields {
		fmt.Fprintf(&b, "%s: %s\n", f.label, f.value)
	}
	return b.String()
}

func (m detailModel) View() string {
	title := zstyle.Title.Render("identity " + m.identity.ID)
	created := zstyle.MutedText.Render(m.identity.CreatedAt.Format(time.RFC3339))
	s := fmt.Sprintf("\n  %s  %s\n\n", title, created)

	for i, f := range m.fields {
		label := zstyle.MutedText.Render(fmt.Sprintf("%-10s", f.label))
		if i == m.cursor {
			s += zstyle.ActiveBorder.Render(fmt.Sprintf("  > %s %s", label, f.value)) + "\n"
		} else {
			s += fmt.Sprintf("    %s %s\n", label, f.value)
		}
	}

	s += "\n"

	// credentials section
	credLabel := fmt.Sprintf("credentials (%d)", m.credentialCount)
	s += "  " + zstyle.Subtitle.Render(credLabel) + "  " + zstyle.MutedText.Render("w to view") + "\n"

	s += "\n"

	// always reserve a line for flash/confirm to prevent layout shift
	if m.confirm {
		s += "  " + zstyle.StatusWarn.Render("delete? y/n") + "\n"
	} else if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "enter copy field  c copy all  w credentials  d delete  esc back  q quit"
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}
