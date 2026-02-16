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

// identityField represents a labeled field for display and selection.
type identityField struct {
	label string
	value string
}

// generateModel displays a generated identity with actions.
type generateModel struct {
	identity identity.Identity
	fields   []identityField
	cursor   int
	flash    string
	flashAt  time.Time
}

// saveIdentityMsg requests saving the current identity.
type saveIdentityMsg struct {
	identity identity.Identity
}

// identitySavedMsg confirms the identity was saved.
type identitySavedMsg struct{}

// flashMsg clears the flash after a timeout.
type flashMsg struct{}

func newGenerateModel(id identity.Identity) generateModel {
	m := generateModel{identity: id}
	m.fields = identityFields(id)
	return m
}

func identityFields(id identity.Identity) []identityField {
	return []identityField{
		{"id", id.ID},
		{"name", id.FirstName + " " + id.LastName},
		{"email", id.Email},
		{"phone", id.Phone},
		{"street", id.Street},
		{"city", id.City},
		{"state", id.State},
		{"zip", id.Zip},
		{"dob", id.DOB.Format("2006-01-02")},
	}
}

func (m generateModel) Init() tea.Cmd {
	return nil
}

func (m generateModel) Update(msg tea.Msg) (generateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case identitySavedMsg:
		return m.setFlash("saved"), clearFlashAfter()

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m generateModel) handleKey(msg tea.KeyMsg) (generateModel, tea.Cmd) {
	// quit takes priority but not for 's', 'c', 'n' etc
	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		return m, func() tea.Msg { return navigateMsg{view: viewMenu} }
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
		// copy selected field
		val := m.fields[m.cursor].value
		if err := copyToClipboard(val); err != nil {
			return m.setFlash("copy: " + err.Error()), clearFlashAfter()
		}
		return m.setFlash("copied!"), clearFlashAfter()
	}

	switch msg.String() {
	case "s":
		return m, func() tea.Msg { return saveIdentityMsg{identity: m.identity} }

	case "c":
		all := m.allFieldsText()
		if err := copyToClipboard(all); err != nil {
			return m.setFlash("copy: " + err.Error()), clearFlashAfter()
		}
		return m.setFlash("copied all!"), clearFlashAfter()

	case "n":
		return m, func() tea.Msg { return navigateMsg{view: viewGenerate} }
	}

	return m, nil
}

func (m generateModel) setFlash(msg string) generateModel {
	m.flash = msg
	m.flashAt = time.Now()
	return m
}

func clearFlashAfter() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return flashMsg{}
	})
}

func (m generateModel) allFieldsText() string {
	var b strings.Builder
	for _, f := range m.fields {
		fmt.Fprintf(&b, "%s: %s\n", f.label, f.value)
	}
	return b.String()
}

func (m generateModel) View() string {
	title := zstyle.Title.Render("generated identity")
	s := fmt.Sprintf("\n  %s\n\n", title)

	for i, f := range m.fields {
		label := zstyle.MutedText.Render(fmt.Sprintf("%-10s", f.label))
		if i == m.cursor {
			s += zstyle.ActiveBorder.Render(fmt.Sprintf("  > %s %s", label, f.value)) + "\n"
		} else {
			s += fmt.Sprintf("    %s %s\n", label, f.value)
		}
	}

	s += "\n"

	// always reserve a line for flash to prevent layout shift
	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "s save  c copy all  enter copy field  n new  esc back  q quit"
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}
