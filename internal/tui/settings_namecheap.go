package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/namecheap"
)

type ncField int

const (
	ncUsername ncField = iota
	ncAPIKey
	ncFieldCount
)

var ncLabels = [ncFieldCount]string{
	"username",
	"api key",
}

// saveNamecheapMsg requests saving namecheap settings.
type saveNamecheapMsg struct {
	settings NamecheapSettings
}

// ncValidateResultMsg carries the result of credential validation.
type ncValidateResultMsg struct {
	domains []string
	err     error
}

// namecheapModel is the form for configuring Namecheap credentials.
type namecheapModel struct {
	inputs    []textinput.Model
	focus     int
	flash     string
	saving    bool
	validateFn func(ctx context.Context, cfg namecheap.Config) ([]string, error)
}

func newNamecheapModel(cfg NamecheapSettings) namecheapModel {
	inputs := make([]textinput.Model, ncFieldCount)

	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		inputs[i] = ti
	}

	inputs[ncUsername].Placeholder = "username"
	inputs[ncUsername].SetValue(cfg.Username)

	inputs[ncAPIKey].Placeholder = "api key"
	inputs[ncAPIKey].SetValue(cfg.APIKey)
	inputs[ncAPIKey].EchoMode = textinput.EchoPassword
	inputs[ncAPIKey].EchoCharacter = '*'

	inputs[0].Focus()

	return namecheapModel{inputs: inputs}
}

func (m namecheapModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m namecheapModel) Update(msg tea.Msg) (namecheapModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}

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
				return m.startValidate()
			}
			return m.nextField(), nil
		}

		switch msg.String() {
		case "ctrl+s":
			return m.startValidate()
		}

	case ncValidateResultMsg:
		m.saving = false
		if msg.err != nil {
			m.flash = msg.err.Error()
			return m, clearFlashAfter()
		}
		s := NamecheapSettings{
			Username:      strings.TrimSpace(m.inputs[ncUsername].Value()),
			APIKey:        strings.TrimSpace(m.inputs[ncAPIKey].Value()),
			CachedDomains: msg.domains,
		}
		m.flash = fmt.Sprintf("saved — %d domains found", len(msg.domains))
		return m, func() tea.Msg { return saveNamecheapMsg{settings: s} }

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m.updateInput(msg)
}

func (m namecheapModel) startValidate() (namecheapModel, tea.Cmd) {
	username := strings.TrimSpace(m.inputs[ncUsername].Value())
	apiKey := strings.TrimSpace(m.inputs[ncAPIKey].Value())

	if username == "" || apiKey == "" {
		m.flash = "username and api key are required"
		return m, clearFlashAfter()
	}

	m.saving = true
	m.flash = "validating..."

	cfg := namecheap.Config{Username: username, APIKey: apiKey}

	validate := m.validateFn
	if validate == nil {
		validate = defaultValidate
	}

	return m, func() tea.Msg {
		domains, err := validate(context.Background(), cfg)
		return ncValidateResultMsg{domains: domains, err: err}
	}
}

func defaultValidate(ctx context.Context, cfg namecheap.Config) ([]string, error) {
	c := namecheap.NewClient(cfg)
	return c.ListDomains(ctx)
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
	accentStyle := lipgloss.NewStyle().Foreground(zstyle.ZburnAccent).Bold(true)

	s := "\n"

	for i, input := range m.inputs {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-12s", ncLabels[i]))
		if i == m.focus {
			s += accentStyle.Render("▸") + " " + label + input.View() + "\n"
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

	return s
}
