package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
)

// passwordModel handles the master password prompt.
type passwordModel struct {
	input      textinput.Model
	firstRun   bool
	confirming bool
	firstPass  string
	errMsg     string
}

// passwordSubmitMsg is sent when the user submits a password.
type passwordSubmitMsg struct {
	password string
}

// passwordErrMsg is sent when the password is wrong.
type passwordErrMsg struct {
	err error
}

func newPasswordModel(firstRun bool) passwordModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 40

	return passwordModel{
		input:    ti,
		firstRun: firstRun,
	}
}

func (m passwordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m passwordModel) Update(msg tea.Msg) (passwordModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		if key.Matches(msg, zstyle.KeyEnter) {
			return m.handleSubmit()
		}

	case passwordErrMsg:
		m.errMsg = msg.err.Error()
		m.input.SetValue("")
		m.confirming = false
		m.firstPass = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m passwordModel) handleSubmit() (passwordModel, tea.Cmd) {
	val := m.input.Value()
	if val == "" {
		return m, nil
	}

	// first run: need to confirm password
	if m.firstRun && !m.confirming {
		m.firstPass = val
		m.confirming = true
		m.input.SetValue("")
		m.errMsg = ""
		return m, nil
	}

	if m.firstRun && m.confirming {
		if val != m.firstPass {
			m.errMsg = "passwords do not match"
			m.confirming = false
			m.firstPass = ""
			m.input.SetValue("")
			return m, nil
		}
	}

	m.errMsg = ""
	return m, func() tea.Msg {
		return passwordSubmitMsg{password: val}
	}
}

func (m passwordModel) View() string {
	title := zstyle.Title.Render("zburn")

	prompt := "master password:"
	if m.confirming {
		prompt = "confirm password:"
	}

	s := fmt.Sprintf("\n  %s\n\n  %s\n  %s\n", title, prompt, m.input.View())

	if m.errMsg != "" {
		s += "\n  " + zstyle.StatusErr.Render(m.errMsg) + "\n"
	}

	s += "\n"
	return s
}
