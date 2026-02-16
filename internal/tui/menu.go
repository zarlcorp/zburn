package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type menuChoice int

const (
	menuGenerate menuChoice = iota
	menuBrowse
	menuSettings
	menuQuit
)

var menuItems = []string{
	"generate identity",
	"browse saved identities",
	"settings",
	"quit",
}

// menuModel is the main menu view.
type menuModel struct {
	cursor  int
	version string
}

// navigateMsg tells the root model to switch views.
type navigateMsg struct {
	view viewID
}

func newMenuModel(version string) menuModel {
	return menuModel{version: version}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, zstyle.KeyQuit) {
			return m, tea.Quit
		}

		if key.Matches(msg, zstyle.KeyUp) {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}

		if key.Matches(msg, zstyle.KeyDown) {
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
			return m, nil
		}

		if key.Matches(msg, zstyle.KeyEnter) {
			return m, m.selectItem()
		}
	}

	return m, nil
}

func (m menuModel) selectItem() tea.Cmd {
	switch menuChoice(m.cursor) {
	case menuGenerate:
		return func() tea.Msg { return navigateMsg{view: viewGenerate} }
	case menuBrowse:
		return func() tea.Msg { return navigateMsg{view: viewList} }
	case menuSettings:
		return func() tea.Msg { return navigateMsg{view: viewSettings} }
	case menuQuit:
		return tea.Quit
	}
	return nil
}

func (m menuModel) View() string {
	indent := lipgloss.NewStyle().MarginLeft(2)
	logo := indent.Render(
		zstyle.StyledLogo(lipgloss.NewStyle().Foreground(zstyle.ZburnAccent)),
	)
	ver := indent.Render(zstyle.MutedText.Render("zburn " + m.version))

	s := fmt.Sprintf("\n%s\n%s\n\n", logo, ver)

	for i, item := range menuItems {
		if m.cursor == i {
			s += zstyle.Highlight.Render(fmt.Sprintf("    > %s", item)) + "\n"
		} else {
			s += fmt.Sprintf("      %s\n", item)
		}
	}

	s += "\n  " + zstyle.MutedText.Render("j/k navigate  enter select  q quit") + "\n"
	return s
}
