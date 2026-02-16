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
	"github.com/zarlcorp/zburn/internal/identity"
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

// fieldMode tracks whether a dual-mode field is in cycle or edit mode.
type fieldMode int

const (
	modeCycle fieldMode = iota
	modeEdit
)

// credentialFormModel handles add/edit for a credential.
type credentialFormModel struct {
	inputs   [fieldCount]textinput.Model
	focus    int
	editing  bool // true = edit existing, false = add new
	existing credential.Credential
	identity identity.Identity
	flash    string

	// dual-mode state (only active when !editing)
	usernameMode    fieldMode
	usernameOptions []string
	usernameIdx     int
	generatedUser   string // current generated username value (for esc restore)
	passwordMode    fieldMode
	generatedPW     string
}

// saveCredentialMsg requests saving a credential.
type saveCredentialMsg struct {
	credential credential.Credential
}

func newCredentialFormModel(id identity.Identity, existing *credential.Credential) credentialFormModel {
	var inputs [fieldCount]textinput.Model
	for i := range fieldCount {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		ti.Prompt = ""
		inputs[i] = ti
	}

	// password field masking (default for edit mode)
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '*'

	m := credentialFormModel{
		inputs:   inputs,
		identity: id,
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
	} else {
		// new credential: set up dual-mode fields
		m.usernameOptions = buildUsernameOptions(id)
		m.usernameIdx = 0
		m.usernameMode = modeCycle
		m.generatedUser = m.usernameOptions[0]
		m.inputs[fieldUsername].SetValue(m.generatedUser)

		m.generatedPW = zcrypto.GeneratePassword(20)
		m.passwordMode = modeCycle
		m.inputs[fieldPassword].SetValue(m.generatedPW)
		// show password in cycle mode
		m.inputs[fieldPassword].EchoMode = textinput.EchoNormal
	}

	m.inputs[m.focus].Focus()
	return m
}

// buildUsernameOptions generates the cycle options for the username field.
func buildUsernameOptions(id identity.Identity) []string {
	first := strings.ToLower(id.FirstName)
	last := strings.ToLower(id.LastName)
	initial := ""
	if len(first) > 0 {
		initial = string(first[0])
	}

	return []string{
		id.Email,
		first + "." + last,
		initial + last,
		first + last,
		// random handle is generated fresh each time via cycling
	}
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

	// esc behavior depends on dual-mode state
	if key.Matches(msg, zstyle.KeyBack) {
		return m.handleEsc()
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

	if key.Matches(msg, zstyle.KeyEnter) {
		return m.submit()
	}

	// dual-mode space handling (only for new credentials, not editing)
	if !m.editing && msg.String() == " " {
		if m.focus == fieldUsername && m.usernameMode == modeCycle {
			return m.cycleUsername(), nil
		}
		if m.focus == fieldPassword && m.passwordMode == modeCycle {
			return m.cyclePassword(), nil
		}
	}

	// dual-mode: typing a printable character in cycle mode switches to edit mode
	if !m.editing && len(msg.Runes) > 0 {
		if m.focus == fieldUsername && m.usernameMode == modeCycle {
			m.usernameMode = modeEdit
			m.inputs[fieldUsername].SetValue("")
			// fall through to updateInput to type the character
		}
		if m.focus == fieldPassword && m.passwordMode == modeCycle {
			m.passwordMode = modeEdit
			m.inputs[fieldPassword].SetValue("")
			m.inputs[fieldPassword].EchoMode = textinput.EchoPassword
			m.inputs[fieldPassword].EchoCharacter = '*'
			// fall through to updateInput to type the character
		}
	}

	return m.updateInput(msg)
}

func (m credentialFormModel) handleEsc() (credentialFormModel, tea.Cmd) {
	// when a dual-mode field is in edit mode, esc returns to cycle mode
	if !m.editing {
		if m.focus == fieldUsername && m.usernameMode == modeEdit {
			m.usernameMode = modeCycle
			m.inputs[fieldUsername].SetValue(m.generatedUser)
			return m, nil
		}
		if m.focus == fieldPassword && m.passwordMode == modeEdit {
			m.passwordMode = modeCycle
			m.inputs[fieldPassword].SetValue(m.generatedPW)
			m.inputs[fieldPassword].EchoMode = textinput.EchoNormal
			return m, nil
		}
	}

	// normal esc behavior: navigate back
	if m.editing {
		c := m.existing
		return m, func() tea.Msg { return viewCredentialMsg{credential: c} }
	}
	return m, func() tea.Msg { return navigateMsg{view: viewCredentialList} }
}

func (m credentialFormModel) cycleUsername() credentialFormModel {
	nextIdx := m.usernameIdx + 1
	// options 0..3 are static name-based; index 4 is random handle
	if nextIdx == len(m.usernameOptions) {
		// show random handle (regenerated each cycle)
		handle := identity.RandomHandle()
		m.generatedUser = handle
		m.inputs[fieldUsername].SetValue(handle)
		m.usernameIdx = nextIdx
		return m
	}
	// after random handle, wrap to start
	if nextIdx > len(m.usernameOptions) {
		nextIdx = 0
	}
	m.usernameIdx = nextIdx
	m.generatedUser = m.usernameOptions[m.usernameIdx]
	m.inputs[fieldUsername].SetValue(m.generatedUser)
	return m
}

func (m credentialFormModel) cyclePassword() credentialFormModel {
	m.generatedPW = zcrypto.GeneratePassword(20)
	m.inputs[fieldPassword].SetValue(m.generatedPW)
	return m
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
		c.IdentityID = m.identity.ID
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
	s := fmt.Sprintf("\n  %s\n", title)

	// identity header
	if !m.editing {
		name := m.identity.FirstName + " " + m.identity.LastName
		header := name + "  " + m.identity.Email
		s += "  " + zstyle.MutedText.Render(header) + "\n"
	}
	s += "\n"

	for i := range fieldCount {
		label := zstyle.MutedText.Render(fmt.Sprintf("  %-12s", fieldLabels[i]))
		cursor := "  "
		if i == m.focus {
			cursor = "> "
		}

		fieldView := m.inputs[i].View()

		// show [generated] indicator for dual-mode fields in cycle mode
		if !m.editing {
			if i == fieldUsername && m.usernameMode == modeCycle {
				fieldView += " " + zstyle.MutedText.Render("[generated]")
			}
			if i == fieldPassword && m.passwordMode == modeCycle {
				fieldView += " " + zstyle.MutedText.Render("[generated]")
			}
		}

		s += fmt.Sprintf("  %s%s %s\n", cursor, label, fieldView)
	}

	s += "\n"

	if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	help := "tab next  shift+tab prev  space cycle  enter save  esc cancel"
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
