package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type twField int

const (
	twAccountSID twField = iota
	twAuthToken
	twFieldCount
)

var twLabels = [twFieldCount]string{
	"account sid",
	"auth token",
}

// preferred country options
var countryOptions = []struct {
	code string
	name string
}{
	{"GB", "UK"},
	{"US", "US"},
}

// saveTwilioMsg requests saving Twilio settings.
type saveTwilioMsg struct {
	settings TwilioSettings
}

// twilioModel is the form for configuring Twilio credentials.
type twilioModel struct {
	inputs    []textinput.Model
	focus     int
	flash     string
	countries map[string]bool // selected country codes
}

func newTwilioModel(cfg TwilioSettings) twilioModel {
	inputs := make([]textinput.Model, twFieldCount)

	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		inputs[i] = ti
	}

	inputs[twAccountSID].Placeholder = "account sid"
	inputs[twAccountSID].SetValue(cfg.AccountSID)

	inputs[twAuthToken].Placeholder = "auth token"
	inputs[twAuthToken].SetValue(cfg.AuthToken)
	inputs[twAuthToken].EchoMode = textinput.EchoPassword
	inputs[twAuthToken].EchoCharacter = '*'

	inputs[0].Focus()

	countries := make(map[string]bool)
	for _, c := range cfg.PreferredCountries {
		countries[c] = true
	}
	// default to both if none set
	if len(countries) == 0 {
		countries["GB"] = true
		countries["US"] = true
	}

	return twilioModel{
		inputs:    inputs,
		countries: countries,
	}
}

func (m twilioModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m twilioModel) Update(msg tea.Msg) (twilioModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, zstyle.KeyQuit) {
			return m, tea.Quit
		}

		if msg.Type == tea.KeyEsc {
			return m, func() tea.Msg { return navigateMsg{view: viewSettings} }
		}

		if key.Matches(msg, zstyle.KeyTab) || msg.Type == tea.KeyDown {
			return m.nextField(), nil
		}

		if msg.Type == tea.KeyUp || msg.Type == tea.KeyShiftTab {
			return m.prevField(), nil
		}

		if key.Matches(msg, zstyle.KeyEnter) {
			// enter on country toggles while in country section
			if m.focus >= int(twFieldCount) {
				idx := m.focus - int(twFieldCount)
				code := countryOptions[idx].code
				m.countries[code] = !m.countries[code]
				return m, nil
			}
			return m.nextField(), nil
		}

		switch msg.String() {
		case "ctrl+s":
			return m, m.save()
		}

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	// only update text inputs when focused on one
	if m.focus < int(twFieldCount) {
		return m.updateInput(msg)
	}

	return m, nil
}

func (m twilioModel) save() tea.Cmd {
	s := TwilioSettings{}
	s.AccountSID = strings.TrimSpace(m.inputs[twAccountSID].Value())
	s.AuthToken = strings.TrimSpace(m.inputs[twAuthToken].Value())

	for code, selected := range m.countries {
		if selected {
			s.PreferredCountries = append(s.PreferredCountries, code)
		}
	}

	return func() tea.Msg { return saveTwilioMsg{settings: s} }
}

func (m twilioModel) totalFields() int {
	return int(twFieldCount) + len(countryOptions)
}

func (m twilioModel) nextField() twilioModel {
	if m.focus < int(twFieldCount) {
		m.inputs[m.focus].Blur()
	}
	m.focus = (m.focus + 1) % m.totalFields()
	if m.focus < int(twFieldCount) {
		m.inputs[m.focus].Focus()
	}
	return m
}

func (m twilioModel) prevField() twilioModel {
	if m.focus < int(twFieldCount) {
		m.inputs[m.focus].Blur()
	}
	m.focus--
	if m.focus < 0 {
		m.focus = m.totalFields() - 1
	}
	if m.focus < int(twFieldCount) {
		m.inputs[m.focus].Focus()
	}
	return m
}

func (m twilioModel) updateInput(msg tea.Msg) (twilioModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m twilioModel) View() string {
	title := zstyle.Title.Render("twilio settings")
	s := fmt.Sprintf("\n  %s\n\n", title)

	for i, input := range m.inputs {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-14s", twLabels[i]))
		if i == m.focus {
			s += zstyle.Highlight.Render("> ") + label + input.View() + "\n"
		} else {
			s += "  " + label + input.View() + "\n"
		}
	}

	s += "\n"
	s += "  " + zstyle.Subtitle.Render("preferred countries") + "\n"

	for i, opt := range countryOptions {
		idx := int(twFieldCount) + i
		check := "[ ]"
		if m.countries[opt.code] {
			check = "[x]"
		}

		if idx == m.focus {
			s += zstyle.Highlight.Render(fmt.Sprintf("  > %s %s", check, opt.name)) + "\n"
		} else {
			s += fmt.Sprintf("    %s %s\n", check, opt.name)
		}
	}

	s += "\n"

	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	s += "  " + zstyle.MutedText.Render("tab next  enter toggle  ctrl+s save  esc back  q quit") + "\n"
	return s
}
