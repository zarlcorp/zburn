package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zarlcorp/core/pkg/zstyle"
)

// pwField identifies which input is focused on the password screen.
type pwField int

const (
	pwFieldPassword pwField = iota
	pwFieldConfirm
)

// passwordModel handles the master password prompt.
type passwordModel struct {
	password textinput.Model
	confirm  textinput.Model
	focused  pwField
	firstRun bool
	errMsg   string
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
	pw := textinput.New()
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '\u2022'
	pw.Focus()
	pw.PromptStyle = lipgloss.NewStyle().Foreground(zstyle.ZburnAccent)
	pw.TextStyle = lipgloss.NewStyle().Foreground(zstyle.Text)

	cf := textinput.New()
	cf.EchoMode = textinput.EchoPassword
	cf.EchoCharacter = '\u2022'
	cf.PromptStyle = lipgloss.NewStyle().Foreground(zstyle.ZburnAccent)
	cf.TextStyle = lipgloss.NewStyle().Foreground(zstyle.Text)

	return passwordModel{
		password: pw,
		confirm:  cf,
		focused:  pwFieldPassword,
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

		// clear error on any key
		m.errMsg = ""

		switch {
		case key.Matches(msg, zstyle.KeyEnter):
			return m.submit()
		case key.Matches(msg, zstyle.KeyTab):
			if m.firstRun {
				return m.nextField(), nil
			}
		}

	case passwordErrMsg:
		m.errMsg = msg.err.Error()
		m.password.SetValue("")
		m.confirm.SetValue("")
		return m, nil
	}

	return m.updateInputs(msg)
}

func (m passwordModel) submit() (passwordModel, tea.Cmd) {
	pw := m.password.Value()
	if pw == "" {
		m.errMsg = "password cannot be empty"
		return m, nil
	}

	if m.firstRun {
		if m.focused == pwFieldPassword {
			return m.nextField(), nil
		}
		if pw != m.confirm.Value() {
			m.errMsg = "passwords do not match"
			m.confirm.SetValue("")
			return m, nil
		}
	}

	return m, func() tea.Msg {
		return passwordSubmitMsg{password: pw}
	}
}

func (m passwordModel) nextField() passwordModel {
	if m.focused == pwFieldPassword {
		m.focused = pwFieldConfirm
		m.password.Blur()
		m.confirm.Focus()
	} else {
		m.focused = pwFieldPassword
		m.confirm.Blur()
		m.password.Focus()
	}
	return m
}

func (m passwordModel) updateInputs(msg tea.Msg) (passwordModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.focused == pwFieldPassword {
		m.password, cmd = m.password.Update(msg)
	} else {
		m.confirm, cmd = m.confirm.Update(msg)
	}
	return m, cmd
}

func (m passwordModel) View() string {
	var b strings.Builder

	// logo
	indent := lipgloss.NewStyle().MarginLeft(2)
	logo := indent.Render(
		zstyle.StyledLogo(lipgloss.NewStyle().Foreground(zstyle.ZburnAccent)),
	)
	toolName := indent.Render(zstyle.MutedText.Render("zburn"))
	b.WriteString(fmt.Sprintf("\n%s\n%s\n\n", logo, toolName))

	// title and description
	if m.firstRun {
		title := lipgloss.NewStyle().
			Foreground(zstyle.ZburnAccent).
			Bold(true).
			Render("create new store")
		b.WriteString(fmt.Sprintf("  %s\n\n", title))
		desc := zstyle.MutedText.Render("choose a master password to protect your store.")
		b.WriteString(fmt.Sprintf("  %s\n\n", desc))
	} else {
		title := lipgloss.NewStyle().
			Foreground(zstyle.ZburnAccent).
			Bold(true).
			Render("unlock store")
		b.WriteString(fmt.Sprintf("  %s\n\n", title))
		desc := zstyle.MutedText.Render("enter your master password.")
		b.WriteString(fmt.Sprintf("  %s\n\n", desc))
	}

	// password field
	label := zstyle.Subtext1
	pwLabel := lipgloss.NewStyle().Foreground(label).Render("password")
	b.WriteString(fmt.Sprintf("  %s\n", pwLabel))
	b.WriteString(fmt.Sprintf("  %s\n", m.password.View()))

	// confirm field (first-run only)
	if m.firstRun {
		b.WriteString("\n")
		cfLabel := lipgloss.NewStyle().Foreground(label).Render("confirm")
		b.WriteString(fmt.Sprintf("  %s\n", cfLabel))
		b.WriteString(fmt.Sprintf("  %s\n", m.confirm.View()))
	}

	// error display
	if m.errMsg != "" {
		b.WriteString("\n")
		errText := zstyle.StatusErr.Render("  " + m.errMsg)
		b.WriteString(errText)
		b.WriteString("\n")
	}

	return b.String()
}
