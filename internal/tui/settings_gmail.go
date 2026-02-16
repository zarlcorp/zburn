package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/gmail"
)

type gmailField int

const (
	gmailClientID gmailField = iota
	gmailClientSecret
	gmailFieldCount
)

var gmailLabels = [gmailFieldCount]string{
	"client id",
	"client secret",
}

// gmailAction represents the current form mode.
type gmailAction int

const (
	gmailActionForm gmailAction = iota
	gmailActionWaiting // waiting for OAuth callback
)

// saveGmailMsg requests saving Gmail settings.
type saveGmailMsg struct {
	settings GmailSettings
}

// gmailOAuthResultMsg carries the result of an OAuth flow.
type gmailOAuthResultMsg struct {
	token *gmail.Token
	err   error
}

// disconnectGmailMsg requests clearing Gmail tokens.
type disconnectGmailMsg struct{}

// gmailModel is the form for configuring Gmail OAuth2 credentials.
type gmailModel struct {
	inputs  []textinput.Model
	focus   int
	flash   string
	action  gmailAction
	current GmailSettings // current saved state

	// authenticateFn allows injecting a test double for the OAuth flow
	authenticateFn func(ctx context.Context, cfg gmail.OAuthConfig) (*gmail.Token, error)
}

func newGmailModel(cfg GmailSettings) gmailModel {
	inputs := make([]textinput.Model, gmailFieldCount)

	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		inputs[i] = ti
	}

	inputs[gmailClientID].Placeholder = "client id"
	inputs[gmailClientID].SetValue(cfg.ClientID)

	inputs[gmailClientSecret].Placeholder = "client secret"
	inputs[gmailClientSecret].SetValue(cfg.ClientSecret)
	inputs[gmailClientSecret].EchoMode = textinput.EchoPassword
	inputs[gmailClientSecret].EchoCharacter = '*'

	inputs[0].Focus()

	return gmailModel{
		inputs:         inputs,
		current:        cfg,
		authenticateFn: gmail.Authenticate,
	}
}

func (m gmailModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m gmailModel) Update(msg tea.Msg) (gmailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.action == gmailActionWaiting {
			// only allow quit while waiting
			if key.Matches(msg, zstyle.KeyQuit) {
				return m, tea.Quit
			}
			if msg.Type == tea.KeyEsc {
				m.action = gmailActionForm
				return m, nil
			}
			return m, nil
		}

		return m.handleKey(msg)

	case gmailOAuthResultMsg:
		m.action = gmailActionForm
		if msg.err != nil {
			m.flash = "oauth: " + msg.err.Error()
			return m, clearFlashAfter()
		}
		s := m.buildSettings()
		s.Token = msg.token
		return m, func() tea.Msg { return saveGmailMsg{settings: s} }

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m.updateInput(msg)
}

func (m gmailModel) handleKey(msg tea.KeyMsg) (gmailModel, tea.Cmd) {
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
		if m.focus == int(gmailFieldCount)-1 {
			return m.startOAuth()
		}
		return m.nextField(), nil
	}

	switch msg.String() {
	case "ctrl+s":
		return m.startOAuth()
	case "ctrl+d":
		if m.current.Configured() {
			return m, func() tea.Msg { return disconnectGmailMsg{} }
		}
		return m, nil
	}

	return m.updateInput(msg)
}

func (m gmailModel) startOAuth() (gmailModel, tea.Cmd) {
	clientID := strings.TrimSpace(m.inputs[gmailClientID].Value())
	clientSecret := strings.TrimSpace(m.inputs[gmailClientSecret].Value())

	if clientID == "" || clientSecret == "" {
		m.flash = "client id and secret required"
		return m, clearFlashAfter()
	}

	m.action = gmailActionWaiting
	cfg := gmail.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	authFn := m.authenticateFn
	return m, func() tea.Msg {
		tok, err := authFn(context.Background(), cfg)
		return gmailOAuthResultMsg{token: tok, err: err}
	}
}

func (m gmailModel) buildSettings() GmailSettings {
	return GmailSettings{
		ClientID:     strings.TrimSpace(m.inputs[gmailClientID].Value()),
		ClientSecret: strings.TrimSpace(m.inputs[gmailClientSecret].Value()),
		Token:        m.current.Token,
	}
}

func (m gmailModel) nextField() gmailModel {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + 1) % int(gmailFieldCount)
	m.inputs[m.focus].Focus()
	return m
}

func (m gmailModel) prevField() gmailModel {
	m.inputs[m.focus].Blur()
	m.focus--
	if m.focus < 0 {
		m.focus = int(gmailFieldCount) - 1
	}
	m.inputs[m.focus].Focus()
	return m
}

func (m gmailModel) updateInput(msg tea.Msg) (gmailModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m gmailModel) View() string {
	title := zstyle.Title.Render("gmail settings")
	s := fmt.Sprintf("\n  %s\n\n", title)

	// show connection status
	if m.current.Configured() {
		s += "  " + zstyle.StatusOK.Render("connected") + "\n\n"
	} else {
		s += "  " + zstyle.MutedText.Render("not connected") + "\n\n"
	}

	if m.action == gmailActionWaiting {
		s += "  " + zstyle.StatusWarn.Render("waiting for browser authorization...") + "\n"
		s += "\n"
		s += "  " + zstyle.MutedText.Render("esc cancel  q quit") + "\n"
		return s
	}

	for i, input := range m.inputs {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-16s", gmailLabels[i]))
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

	help := "tab next  enter connect  esc back  q quit"
	if m.current.Configured() {
		help = "tab next  enter connect  ctrl+d disconnect  esc back  q quit"
	}
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}
