package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zcrypto"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/credential"
)

// credentialDetailModel displays a single credential with live TOTP.
type credentialDetailModel struct {
	credential credential.Credential
	revealed   bool // password reveal toggle
	flash      string
	confirm    bool
	totpCode   string
	totpErr    string
}

// totpTickMsg triggers a TOTP refresh.
type totpTickMsg struct{}

func newCredentialDetailModel(c credential.Credential) credentialDetailModel {
	m := credentialDetailModel{credential: c}
	m.refreshTOTP()
	return m
}

func (m *credentialDetailModel) refreshTOTP() {
	if m.credential.TOTPSecret == "" {
		m.totpCode = ""
		m.totpErr = ""
		return
	}
	code, err := zcrypto.TOTPCode(m.credential.TOTPSecret)
	if err != nil {
		m.totpErr = err.Error()
		m.totpCode = ""
		return
	}
	m.totpCode = code
	m.totpErr = ""
}

func (m credentialDetailModel) Init() tea.Cmd {
	if m.credential.TOTPSecret == "" {
		return nil
	}
	return m.tickTOTP()
}

func (m credentialDetailModel) tickTOTP() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return totpTickMsg{}
	})
}

func (m credentialDetailModel) Update(msg tea.Msg) (credentialDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case totpTickMsg:
		m.refreshTOTP()
		if m.credential.TOTPSecret != "" {
			return m, m.tickTOTP()
		}
		return m, nil

	case flashMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func (m credentialDetailModel) handleKey(msg tea.KeyMsg) (credentialDetailModel, tea.Cmd) {
	if m.confirm {
		return m.handleConfirm(msg)
	}

	if key.Matches(msg, zstyle.KeyQuit) {
		return m, tea.Quit
	}

	if key.Matches(msg, zstyle.KeyBack) {
		return m, func() tea.Msg { return navigateMsg{view: viewCredentialList} }
	}

	switch msg.String() {
	case "r":
		m.revealed = !m.revealed
		return m, nil

	case "c":
		pw := m.credential.Password
		if err := copyToClipboard(pw); err != nil {
			m.flash = "copy: " + err.Error()
			return m, clearFlashAfter()
		}
		m.flash = "password copied"
		return m, clearFlashAfter()

	case "t":
		if m.credential.TOTPSecret == "" {
			return m, nil
		}
		code, err := zcrypto.TOTPCode(m.credential.TOTPSecret)
		if err != nil {
			m.flash = "totp: " + err.Error()
			return m, clearFlashAfter()
		}
		if err := copyToClipboard(code); err != nil {
			m.flash = "copy: " + err.Error()
			return m, clearFlashAfter()
		}
		m.flash = "totp code copied"
		return m, clearFlashAfter()

	case "e":
		c := m.credential
		return m, func() tea.Msg { return editCredentialMsg{credential: c} }

	case "d":
		m.confirm = true
		return m, nil
	}

	return m, nil
}

func (m credentialDetailModel) handleConfirm(msg tea.KeyMsg) (credentialDetailModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		id := m.credential.ID
		m.confirm = false
		return m, func() tea.Msg { return deleteCredentialMsg{id: id} }
	default:
		m.confirm = false
		return m, nil
	}
}

func (m credentialDetailModel) totpCountdown() int {
	return 30 - int(time.Now().Unix()%30)
}

func (m credentialDetailModel) View() string {
	// sub-header with credential label
	s := "\n  " + zstyle.Subtitle.Render(m.credential.Label) + "\n\n"

	// fields
	s += m.fieldLine("url", m.credential.URL)
	s += m.fieldLine("username", m.credential.Username)

	if m.revealed {
		s += m.fieldLine("password", m.credential.Password)
	} else {
		s += m.fieldLine("password", "••••••••")
	}

	if m.credential.TOTPSecret != "" {
		if m.totpErr != "" {
			s += m.fieldLine("totp", "error: "+m.totpErr)
		} else {
			countdown := m.totpCountdown()
			s += m.fieldLine("totp", fmt.Sprintf("%s (%ds)", m.totpCode, countdown))
		}
	}

	if m.credential.Notes != "" {
		s += m.fieldLine("notes", m.credential.Notes)
	}

	s += "\n"
	s += "  " + zstyle.MutedText.Render(fmt.Sprintf("created  %s", m.credential.CreatedAt.Format(time.RFC3339))) + "\n"
	s += "  " + zstyle.MutedText.Render(fmt.Sprintf("updated  %s", m.credential.UpdatedAt.Format(time.RFC3339))) + "\n"

	s += "\n"

	if m.confirm {
		label := m.credential.Label
		s += "  " + zstyle.StatusWarn.Render(fmt.Sprintf("delete credential %q? this cannot be undone. (y/n)", label)) + "\n"
	} else if m.flash != "" {
		s += "  " + zstyle.StatusOK.Render(m.flash) + "\n"
	} else {
		s += "\n"
	}

	return s
}

func (m credentialDetailModel) fieldLine(label, value string) string {
	l := zstyle.MutedText.Render(fmt.Sprintf("  %-10s", label))
	return fmt.Sprintf("  %s %s\n", l, value)
}
