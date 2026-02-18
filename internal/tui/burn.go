package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/burn"
	"github.com/zarlcorp/zburn/internal/identity"
)

type burnPhase int

const (
	burnConfirm burnPhase = iota
	burnRunning
	burnDone
)

// burnIdentityMsg requests a burn cascade for a specific identity.
type burnIdentityMsg struct {
	identity identity.Identity
}

// burnResultMsg carries the result of a completed burn cascade.
type burnResultMsg struct {
	result burn.Result
}

// burnModel manages the burn confirmation dialog and result display.
type burnModel struct {
	identity identity.Identity
	plan     []string
	phase    burnPhase
	result   burn.Result
}

func newBurnModel(id identity.Identity, plan []string) burnModel {
	return burnModel{
		identity: id,
		plan:     plan,
		phase:    burnConfirm,
	}
}

func (m burnModel) Init() tea.Cmd {
	return nil
}

func (m burnModel) Update(msg tea.Msg) (burnModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case burnResultMsg:
		m.result = msg.result
		m.phase = burnDone
		return m, clearFlashAfter3s()
	}

	return m, nil
}

func (m burnModel) handleKey(msg tea.KeyMsg) (burnModel, tea.Cmd) {
	switch m.phase {
	case burnConfirm:
		return m.handleConfirmKey(msg)
	case burnDone:
		// any key returns to list
		return m, func() tea.Msg { return navigateMsg{view: viewList} }
	}
	return m, nil
}

func (m burnModel) handleConfirmKey(msg tea.KeyMsg) (burnModel, tea.Cmd) {
	// quit always works
	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	switch msg.String() {
	case "y":
		m.phase = burnRunning
		id := m.identity
		return m, func() tea.Msg { return burnIdentityMsg{identity: id} }
	default:
		// any other key cancels, go back to detail
		return m, func() tea.Msg { return navigateMsg{view: viewDetail} }
	}
}

func (m burnModel) View() string {
	switch m.phase {
	case burnConfirm:
		return m.viewConfirm()
	case burnRunning:
		return m.viewRunning()
	case burnDone:
		return m.viewDone()
	}
	return ""
}

func (m burnModel) viewConfirm() string {
	name := m.identity.FirstName + " " + m.identity.LastName

	s := "\n  " + zstyle.Subtitle.Render("burn "+name+"?") + "\n\n"

	s += "  " + zstyle.MutedText.Render("this will:") + "\n"
	for _, step := range m.plan {
		s += fmt.Sprintf("  %s %s\n", zstyle.StatusWarn.Render("-"), step)
	}

	s += "\n"
	s += "  " + zstyle.StatusWarn.Render("this cannot be undone.") + " (y/n)\n"

	return s
}

func (m burnModel) viewRunning() string {
	name := m.identity.FirstName + " " + m.identity.LastName
	s := "\n  " + zstyle.MutedText.Render("burning "+name+"...") + "\n"
	return s
}

func (m burnModel) viewDone() string {
	var b strings.Builder

	title := m.result.Summary()
	lines := strings.Split(title, "\n")

	// first line is the header
	if m.result.HasErrors() {
		b.WriteString("\n  " + zstyle.StatusWarn.Render(lines[0]) + "\n\n")
	} else {
		b.WriteString("\n  " + zstyle.StatusOK.Render(lines[0]) + "\n\n")
	}

	// remaining lines are step details
	for _, line := range lines[1:] {
		if strings.Contains(line, ":") && strings.Contains(line, "- ") {
			b.WriteString("  " + zstyle.StatusWarn.Render(line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + zstyle.MutedText.Render("press any key to continue") + "\n")
	return b.String()
}

// clearFlashAfter3s returns to list after 3 seconds.
func clearFlashAfter3s() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return navigateMsg{view: viewList}
	})
}
