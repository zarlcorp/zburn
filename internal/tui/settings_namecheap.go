package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

type ncField int

const (
	ncAPIUser ncField = iota
	ncAPIKey
	ncUsername
	ncClientIP
	ncDomains
	ncFieldCount
)

var ncLabels = [ncFieldCount]string{
	"api user",
	"api key",
	"username",
	"client ip",
	"domains",
}

// saveNamecheapMsg requests saving namecheap settings.
type saveNamecheapMsg struct {
	settings NamecheapSettings
}

// namecheapModel is the form for configuring Namecheap credentials.
type namecheapModel struct {
	inputs []textinput.Model
	focus  int
	flash  string
}

func newNamecheapModel(cfg NamecheapSettings) namecheapModel {
	inputs := make([]textinput.Model, ncFieldCount)

	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		inputs[i] = ti
	}

	inputs[ncAPIUser].Placeholder = "api user"
	inputs[ncAPIUser].SetValue(cfg.APIUser)

	inputs[ncAPIKey].Placeholder = "api key"
	inputs[ncAPIKey].SetValue(cfg.APIKey)
	inputs[ncAPIKey].EchoMode = textinput.EchoPassword
	inputs[ncAPIKey].EchoCharacter = '*'

	inputs[ncUsername].Placeholder = "username"
	inputs[ncUsername].SetValue(cfg.Username)

	inputs[ncClientIP].Placeholder = "client ip"
	inputs[ncClientIP].SetValue(cfg.ClientIP)

	inputs[ncDomains].Placeholder = "example.com, other.io"
	inputs[ncDomains].SetValue(strings.Join(cfg.Domains, ", "))
	inputs[ncDomains].CharLimit = 1024

	inputs[0].Focus()

	return namecheapModel{inputs: inputs}
}

func (m namecheapModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m namecheapModel) Update(msg tea.Msg) (namecheapModel, tea.Cmd) {
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
			// enter on last field saves; otherwise advance
			if m.focus == int(ncFieldCount)-1 {
				return m, m.save()
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

	return m.updateInput(msg)
}

func (m namecheapModel) save() tea.Cmd {
	s := NamecheapSettings{}
	s.APIUser = strings.TrimSpace(m.inputs[ncAPIUser].Value())
	s.APIKey = strings.TrimSpace(m.inputs[ncAPIKey].Value())
	s.Username = strings.TrimSpace(m.inputs[ncUsername].Value())
	s.ClientIP = strings.TrimSpace(m.inputs[ncClientIP].Value())
	s.Domains = parseDomains(m.inputs[ncDomains].Value())

	return func() tea.Msg { return saveNamecheapMsg{settings: s} }
}

func (m namecheapModel) nextField() namecheapModel {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + 1) % int(ncFieldCount)
	m.inputs[m.focus].Focus()
	return m
}

func (m namecheapModel) prevField() namecheapModel {
	m.inputs[m.focus].Blur()
	m.focus--
	if m.focus < 0 {
		m.focus = int(ncFieldCount) - 1
	}
	m.inputs[m.focus].Focus()
	return m
}

func (m namecheapModel) updateInput(msg tea.Msg) (namecheapModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m namecheapModel) View() string {
	title := zstyle.Title.Render("namecheap settings")
	s := fmt.Sprintf("\n  %s\n\n", title)

	for i, input := range m.inputs {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-12s", ncLabels[i]))
		if i == m.focus {
			s += zstyle.Highlight.Render("> ") + label + input.View() + "\n"
		} else {
			s += "  " + label + input.View() + "\n"
		}
	}

	s += "\n"

	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	s += "  " + zstyle.MutedText.Render("tab next  ctrl+s save  esc back  q quit") + "\n"
	return s
}

// parseDomains splits a comma/space/newline separated string into domain names.
func parseDomains(s string) []string {
	s = strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(s, ",")
	var domains []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			domains = append(domains, p)
		}
	}
	return domains
}
