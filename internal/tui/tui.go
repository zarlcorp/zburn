// Package tui implements the root Bubble Tea model for zburn.
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

// Model is the root TUI model.
type Model struct {
	version string
}

// New creates the root TUI model.
func New(version string) Model {
	return Model{version: version}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, zstyle.KeyQuit) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	title := zstyle.Title.Render("zburn")
	ver := zstyle.MutedText.Render(m.version)
	help := zstyle.MutedText.Render("press q to quit")
	return fmt.Sprintf("\n  %s %s\n\n  %s\n\n", title, ver, help)
}
