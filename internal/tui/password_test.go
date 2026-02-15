package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPasswordModel_QKeyDoesNotQuit(t *testing.T) {
	m := newPasswordModel(false)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		// execute the cmd to see what message it produces
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

	if updated.input.Value() != "q" {
		t.Fatalf("expected textinput to contain %q, got %q", "q", updated.input.Value())
	}
}
