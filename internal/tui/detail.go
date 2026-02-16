package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/identity"
)

// viewCredentialsMsg requests viewing credentials for an identity.
type viewCredentialsMsg struct {
	identity identity.Identity
}

// burnStartMsg tells the root model to show the burn confirmation for an identity.
type burnStartMsg struct {
	identity identity.Identity
}

// detailModel displays all fields of a saved identity.
type detailModel struct {
	identity        identity.Identity
	fields          []identityField
	cursor          int
	flash           string
	credentialCount int
	avatar          string
}

func newDetailModel(id identity.Identity) detailModel {
	return detailModel{
		identity: id,
		fields:   identityFields(id),
		avatar:   renderAvatar(),
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
		id := m.identity
		return m, func() tea.Msg { return viewCredentialsMsg{identity: id} }

	case "d":
		id := m.identity
		return m, func() tea.Msg { return burnStartMsg{identity: id} }
	}

	return m, nil
}

func (m detailModel) allFieldsText() string {
	var b strings.Builder
	for _, f := range m.fields {
		fmt.Fprintf(&b, "%s: %s\n", f.label, f.value)
	}
	return b.String()
}

func (m detailModel) View() string {
	title := zstyle.Title.Render(m.identity.FirstName + " " + m.identity.LastName)
	s := fmt.Sprintf("\n  %s\n\n", title)

	if m.avatar != "" {
		s += "  " + m.avatar + "\n\n"
	}

	for i, f := range m.fields {
		if sectionBreaks[i] {
			s += "\n"
		}
		label := zstyle.MutedText.Render(fmt.Sprintf("%-10s", f.label))
		if i == m.cursor {
			s += zstyle.ActiveBorder.Render(fmt.Sprintf("  > %s %s", label, f.value)) + "\n"
		} else {
			s += fmt.Sprintf("    %s %s\n", label, f.value)
		}
	}

	s += "\n"

	// credentials section
	if m.credentialCount > 0 {
		s += "  " + zstyle.MutedText.Render(fmt.Sprintf("(%d) credentials  w to view", m.credentialCount)) + "\n"
	} else {
		s += "  " + zstyle.MutedText.Render("no credentials  a to add") + "\n"
	}

	s += "\n"

	// always reserve a line for flash to prevent layout shift
	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "enter copy field  c copy all  w credentials  d burn  esc back  q quit"
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}
