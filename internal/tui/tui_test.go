package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/zburn/internal/tui"
)

func TestView(t *testing.T) {
	m := tui.New("0.1.0")
	view := m.View()

	if !strings.Contains(view, "zburn") {
		t.Error("view should contain zburn")
	}
	if !strings.Contains(view, "0.1.0") {
		t.Error("view should contain version")
	}
}

func TestQuitOnQ(t *testing.T) {
	m := tui.New("0.1.0")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("pressing q should return a quit command")
	}
}

func TestQuitOnCtrlC(t *testing.T) {
	m := tui.New("0.1.0")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("pressing ctrl+c should return a quit command")
	}
}

func TestInitReturnsNil(t *testing.T) {
	m := tui.New("0.1.0")
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should return nil")
	}
}

func TestIgnoresOtherKeys(t *testing.T) {
	m := tui.New("0.1.0")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("non-quit keys should not produce a command")
	}
}
