package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type settingsChoice int

const (
	settingsNamecheap settingsChoice = iota
	settingsGmail
	settingsTwilio
	settingsForwarding
	settingsBack
)

var settingsItems = []string{
	"namecheap",
	"gmail",
	"twilio",
	"forwarding",
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
	case settingsForwarding:
		return func() tea.Msg { return navigateMsg{view: viewForwarding} }
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
	s := "\n"

	for i, item := range settingsItems {
		choice := settingsChoice(i)

		// build count/status string for service items
		var countStr string
		if choice != settingsForwarding && choice != settingsBack {
			status := m.statusFor(choice)
			statusStyle := zstyle.StatusErr
			if status == "configured" {
				statusStyle = zstyle.StatusOK
			}
			countStr = statusStyle.Render(status)
		}

		mi := zstyle.MenuItem{
			Label:  item,
			Active: m.cursor == i,
		}
		line := zstyle.RenderMenuItem(mi, zstyle.ZburnAccent)
		if countStr != "" {
			line += " " + countStr
		}
		s += line + "\n"
	}

	return s
}
