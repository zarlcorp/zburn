package tui

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zcrypto"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/credential"
)

const (
	fieldLabel = iota
	fieldURL
	fieldUsername
	fieldPassword
	fieldTOTPSecret
	fieldNotes
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"label",
	"url",
	"username",
	"password",
	"totp secret",
	"notes",
}

// credentialFormModel handles add/edit for a credential.
type credentialFormModel struct {
	inputs     [fieldCount]textinput.Model
	focus      int
	editing    bool // true = edit existing, false = add new
	existing   credential.Credential
	identityID string
	flash      string
}

// saveCredentialMsg requests saving a credential.
type saveCredentialMsg struct {
	credential credential.Credential
}

func newCredentialFormModel(identityID string, existing *credential.Credential) credentialFormModel {
	var inputs [fieldCount]textinput.Model
	for i := range fieldCount {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		ti.Prompt = ""
		inputs[i] = ti
	}

	// password field masking
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '*'

	m := credentialFormModel{
		inputs:     inputs,
		identityID: identityID,
	}

	if existing != nil {
		m.editing = true
		m.existing = *existing
		m.inputs[fieldLabel].SetValue(existing.Label)
		m.inputs[fieldURL].SetValue(existing.URL)
		m.inputs[fieldUsername].SetValue(existing.Username)
		m.inputs[fieldPassword].SetValue(existing.Password)
		m.inputs[fieldTOTPSecret].SetValue(existing.TOTPSecret)
		m.inputs[fieldNotes].SetValue(existing.Notes)
	}

	m.inputs[m.focus].Focus()
	return m
}

func (m credentialFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m credentialFormModel) Update(msg tea.Msg) (credentialFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m.updateInput(msg)
}

func (m credentialFormModel) handleKey(msg tea.KeyMsg) (credentialFormModel, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		if m.editing {
			c := m.existing
			return m, func() tea.Msg { return viewCredentialMsg{credential: c} }
		}
		return m, func() tea.Msg { return navigateMsg{view: viewCredentialList} }
	}

	switch msg.String() {
	case "tab":
		m.inputs[m.focus].Blur()
		m.focus = (m.focus + 1) % fieldCount
		m.inputs[m.focus].Focus()
		return m, textinput.Blink

	case "shift+tab":
		m.inputs[m.focus].Blur()
		m.focus = (m.focus - 1 + fieldCount) % fieldCount
		m.inputs[m.focus].Focus()
		return m, textinput.Blink
	}

	if msg.String() == "ctrl+g" && m.focus == fieldPassword {
		pw := zcrypto.GeneratePassword(20)
		m.inputs[fieldPassword].SetValue(pw)
		m.flash = "password generated"
		return m, clearFlashAfter()
	}

	if key.Matches(msg, zstyle.KeyEnter) {
		return m.submit()
	}

	return m.updateInput(msg)
}

func (m credentialFormModel) updateInput(msg tea.Msg) (credentialFormModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m credentialFormModel) submit() (credentialFormModel, tea.Cmd) {
	label := strings.TrimSpace(m.inputs[fieldLabel].Value())
	if label == "" {
		m.flash = "label is required"
		return m, clearFlashAfter()
	}

	totpSecret := strings.TrimSpace(m.inputs[fieldTOTPSecret].Value())
	if totpSecret != "" {
		if !isValidBase32(totpSecret) {
			m.flash = "invalid totp secret (must be base32)"
			return m, clearFlashAfter()
		}
	}

	now := time.Now().UTC()

	var c credential.Credential
	if m.editing {
		c = m.existing
		c.UpdatedAt = now
	} else {
		c.ID = credentialHexID()
		c.IdentityID = m.identityID
		c.CreatedAt = now
		c.UpdatedAt = now
	}

	c.Label = label
	c.URL = strings.TrimSpace(m.inputs[fieldURL].Value())
	c.Username = strings.TrimSpace(m.inputs[fieldUsername].Value())
	c.Password = m.inputs[fieldPassword].Value()
	c.TOTPSecret = totpSecret
	c.Notes = strings.TrimSpace(m.inputs[fieldNotes].Value())

	return m, func() tea.Msg { return saveCredentialMsg{credential: c} }
}

func (m credentialFormModel) View() string {
	action := "add credential"
	if m.editing {
		action = "edit credential"
	}
	title := zstyle.Title.Render(action)
	s := fmt.Sprintf("\n  %s\n\n", title)

	for i := range fieldCount {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-12s", fieldLabels[i]))
		cursor := "  "
		if i == m.focus {
			cursor = "> "
		}
		s += fmt.Sprintf("  %s%s %s\n", cursor, label, m.inputs[i].View())
	}

	s += "\n"

	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "tab next  shift+tab prev  ctrl+g generate password  enter save  esc cancel"
	s += "  " + zstyle.MutedText.Render(help) + "\n"
	return s
}

func isValidBase32(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	if pad := len(s) % 8; pad != 0 {
		s += strings.Repeat("=", 8-pad)
	}
	_, err := base32.StdEncoding.DecodeString(s)
	return err == nil
}

func credentialHexID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}
