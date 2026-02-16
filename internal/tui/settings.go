package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type settingsChoice int

const (
	settingsNamecheap settingsChoice = iota
	settingsGmail
	settingsTwilio
	settingsBack
)

var settingsItems = []string{
	"namecheap",
	"gmail",
	"twilio",
	"back",
}

// settingsModel displays the settings menu with integration status.
type settingsModel struct {
	cursor    int
	namecheap NamecheapSettings
	gmail     GmailSettings
	twilio    TwilioSettings
}

func newSettingsModel(nc NamecheapSettings, gm GmailSettings, tw TwilioSettings) settingsModel {
	return settingsModel{
		namecheap: nc,
		gmail:     gm,
		twilio:    tw,
	}
}

func (m settingsModel) Init() tea.Cmd {
	return nil
}

func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
			if m.cursor < len(settingsItems)-1 {
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

func (m settingsModel) selectItem() tea.Cmd {
	switch settingsChoice(m.cursor) {
	case settingsNamecheap:
		return func() tea.Msg { return navigateMsg{view: viewSettingsNamecheap} }
	case settingsGmail:
		return func() tea.Msg { return navigateMsg{view: viewSettingsGmail} }
	case settingsTwilio:
		return func() tea.Msg { return navigateMsg{view: viewSettingsTwilio} }
	case settingsBack:
		return func() tea.Msg { return navigateMsg{view: viewMenu} }
	}
	return nil
}

func (m settingsModel) statusFor(choice settingsChoice) string {
	switch choice {
	case settingsNamecheap:
		if m.namecheap.Configured() {
			return "configured"
		}
	case settingsGmail:
		if m.gmail.Configured() {
			return "configured"
		}
	case settingsTwilio:
		if m.twilio.Configured() {
			return "configured"
		}
	}
	return "not configured"
}

func (m settingsModel) View() string {
	title := zstyle.Title.Render("settings")
	s := fmt.Sprintf("\n  %s\n\n", title)

	for i, item := range settingsItems {
		if i == len(settingsItems)-1 {
			// "back" has no status
			if m.cursor == i {
				s += zstyle.Highlight.Render(fmt.Sprintf("    > %s", item)) + "\n"
			} else {
				s += fmt.Sprintf("      %s\n", item)
			}
			continue
		}

		status := m.statusFor(settingsChoice(i))
		statusStyle := zstyle.StatusErr
		if status == "configured" {
			statusStyle = zstyle.StatusOK
		}

		if m.cursor == i {
			s += zstyle.Highlight.Render(fmt.Sprintf("    > %-14s", item)) +
				" " + statusStyle.Render(status) + "\n"
		} else {
			s += fmt.Sprintf("      %-14s", item) +
				" " + statusStyle.Render(status) + "\n"
		}
	}

	s += "\n  " + zstyle.MutedText.Render("j/k navigate  enter select  esc back  q quit") + "\n"
	return s
}
