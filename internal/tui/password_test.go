package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPasswordModel_QKeyDoesNotQuit(t *testing.T) {
	m := newPasswordModel(false)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		result := cmd()
		if _, ok := result.(tea.QuitMsg); ok {
			t.Fatal("pressing 'q' should not quit the password view")
		}
	}
}

func TestPasswordModel_CtrlCQuits(t *testing.T) {
	m := newPasswordModel(false)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("ctrl+c should produce a quit command")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c should produce QuitMsg, got %T", result)
	}
}

func TestPasswordModel_QKeyReachesTextInput(t *testing.T) {
	m := newPasswordModel(false)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, _ := m.Update(msg)

	if updated.password.Value() != "q" {
		t.Fatalf("expected textinput to contain %q, got %q", "q", updated.password.Value())
	}
}

func TestPasswordFirstRunTwoFieldLayout(t *testing.T) {
	m := newPasswordModel(true)
	view := m.View()

	if !strings.Contains(view, "confirm") {
		t.Error("first-run view should show confirm field")
	}
	if !strings.Contains(view, "password") {
		t.Error("first-run view should show password label")
	}
	if !strings.Contains(view, "create new store") {
		t.Error("first-run view should show 'create new store' title")
	}
}

func TestPasswordUnlockSingleFieldLayout(t *testing.T) {
	m := newPasswordModel(false)
	view := m.View()

	if strings.Contains(view, "confirm") {
		t.Error("unlock view should not show confirm field")
	}
	if !strings.Contains(view, "unlock store") {
		t.Error("unlock view should show 'unlock store' title")
	}
	if !strings.Contains(view, "enter your master password") {
		t.Error("unlock view should show unlock description")
	}
}

func TestPasswordTabSwitchesFields(t *testing.T) {
	m := newPasswordModel(true)

	if m.focused != pwFieldPassword {
		t.Fatal("should start focused on password")
	}

	// tab to confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != pwFieldConfirm {
		t.Error("tab should switch to confirm field")
	}

	// tab back to password
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != pwFieldPassword {
		t.Error("tab should switch back to password field")
	}
}

func TestPasswordTabIgnoredOnUnlock(t *testing.T) {
	m := newPasswordModel(false)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != pwFieldPassword {
		t.Error("tab should not switch fields on unlock")
	}
}

func TestPasswordMismatchError(t *testing.T) {
	m := newPasswordModel(true)

	m.password.SetValue("secret1")
	// enter moves to confirm field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	m.confirm.SetValue("secret2")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !strings.Contains(m.View(), "passwords do not match") {
		t.Error("should show mismatch error")
	}
	if m.confirm.Value() != "" {
		t.Error("confirm field should be cleared on mismatch")
	}
}

func TestPasswordMatchSubmits(t *testing.T) {
	m := newPasswordModel(true)

	m.password.SetValue("secret")
	// enter moves to confirm field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	m.confirm.SetValue("secret")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("should emit command on matching passwords")
	}
	msg := cmd()
	submit, ok := msg.(passwordSubmitMsg)
	if !ok {
		t.Fatal("should emit passwordSubmitMsg")
	}
	if submit.password != "secret" {
		t.Errorf("password = %q, want %q", submit.password, "secret")
	}
	_ = m
}

func TestPasswordEmptyShowsError(t *testing.T) {
	m := newPasswordModel(false)
	m.password.SetValue("")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("empty password should not emit command")
	}
	if !strings.Contains(m.View(), "password cannot be empty") {
		t.Error("should show empty password error")
	}
}

func TestPasswordUnlockSubmitsImmediately(t *testing.T) {
	m := newPasswordModel(false)
	m.password.SetValue("secret")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("should emit command on unlock submit")
	}
	msg := cmd()
	submit, ok := msg.(passwordSubmitMsg)
	if !ok {
		t.Fatal("should emit passwordSubmitMsg")
	}
	if submit.password != "secret" {
		t.Errorf("password = %q, want %q", submit.password, "secret")
	}
}

func TestPasswordErrorClearsOnKeyPress(t *testing.T) {
	m := newPasswordModel(false)
	m.errMsg = "some error"

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.errMsg != "" {
		t.Error("error should be cleared on key press")
	}
}

func TestPasswordErrMsgClearsFields(t *testing.T) {
	m := newPasswordModel(false)
	m.password.SetValue("wrong")

	m, _ = m.Update(passwordErrMsg{err: errTest("bad password")})

	if m.password.Value() != "" {
		t.Error("password should be cleared on error")
	}
	if !strings.Contains(m.View(), "bad password") {
		t.Error("should display error message")
	}
}
