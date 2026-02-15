package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type menuChoice int

const (
	menuGenerate menuChoice = iota
	menuEmail
	menuBrowse
	menuQuit
)

var menuItems = []string{
	"Generate identity",
	"Generate email (quick)",
	"Browse saved identities",
	"Quit",
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

// quickEmailMsg tells the root to generate and copy an email.
type quickEmailMsg struct{}

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
	case menuEmail:
		return func() tea.Msg { return quickEmailMsg{} }
	case menuBrowse:
		return func() tea.Msg { return navigateMsg{view: viewList} }
	case menuQuit:
		return tea.Quit
	}
	return nil
}

func (m menuModel) View() string {
	title := zstyle.Title.Render("zburn")
	ver := zstyle.MutedText.Render(m.version)

	s := fmt.Sprintf("\n  %s %s\n\n", title, ver)

	for i, item := range menuItems {
		cursor := "  "
		if m.cursor == i {
			s += zstyle.Highlight.Render(fmt.Sprintf("  %s> %s", cursor, item)) + "\n"
		} else {
			s += fmt.Sprintf("  %s  %s\n", cursor, item)
		}
	}

	s += "\n  " + zstyle.MutedText.Render("j/k navigate  enter select  q quit") + "\n\n"
	return s
}
