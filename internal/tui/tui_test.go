package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/zburn/internal/identity"
)

// helpers

func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func specialKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func enterKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func escKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

func testIdentity() identity.Identity {
	return identity.Identity{
		ID:        "abc12345",
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane@zburn.id",
		Phone:     "(555) 123-4567",
		Street:    "123 Oak Ave",
		City:      "Portland",
		State:     "OR",
		Zip:       "97201",
		DOB:       time.Date(1990, 6, 15, 0, 0, 0, 0, time.UTC),
		Password:  "s3cret!Pass",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// password view tests

func TestPasswordViewShowsPrompt(t *testing.T) {
	m := newPasswordModel(false)
	view := m.View()

	if !strings.Contains(view, "master password") {
		t.Error("view should show master password prompt")
	}
	if strings.Contains(view, "create") {
		t.Error("non-first-run view should not contain 'create'")
	}
	if !strings.Contains(view, "zburn") {
		t.Error("view should show title")
	}
}

func TestPasswordFirstRunShowsCreate(t *testing.T) {
	m := newPasswordModel(true)
	view := m.View()

	if !strings.Contains(view, "create master password") {
		t.Error("first-run view should show 'create master password'")
	}
}

func TestPasswordFirstRunShowsConfirm(t *testing.T) {
	m := newPasswordModel(true)

	// type password
	m.input.SetValue("secret")
	m, _ = m.Update(enterKey())

	if !m.confirming {
		t.Error("should be in confirming state after first entry")
	}
	if !strings.Contains(m.View(), "confirm password") {
		t.Error("view should show confirm prompt")
	}
}

func TestPasswordFirstRunMismatch(t *testing.T) {
	m := newPasswordModel(true)

	// first entry
	m.input.SetValue("secret1")
	m, _ = m.Update(enterKey())

	// second entry (mismatch)
	m.input.SetValue("secret2")
	m, _ = m.Update(enterKey())

	if !strings.Contains(m.View(), "passwords do not match") {
		t.Error("should show mismatch error")
	}
	if m.confirming {
		t.Error("should reset confirming state")
	}
}

func TestPasswordFirstRunMatch(t *testing.T) {
	m := newPasswordModel(true)

	// first entry
	m.input.SetValue("secret")
	m, _ = m.Update(enterKey())

	// confirm
	m.input.SetValue("secret")
	m, cmd := m.Update(enterKey())

	if cmd == nil {
		t.Fatal("should emit command on matching passwords")
	}

	msg := cmd()
	if submit, ok := msg.(passwordSubmitMsg); !ok {
		t.Error("should emit passwordSubmitMsg")
	} else if submit.password != "secret" {
		t.Errorf("password = %q, want %q", submit.password, "secret")
	}
	_ = m
}

func TestPasswordSubmitEmptyIgnored(t *testing.T) {
	m := newPasswordModel(false)
	m.input.SetValue("")
	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("empty password should not emit command")
	}
}

func TestPasswordQuit(t *testing.T) {
	m := newPasswordModel(false)
	_, cmd := m.Update(specialKey(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestPasswordErrMsgClearsInput(t *testing.T) {
	m := newPasswordModel(false)
	m.input.SetValue("wrong")

	m, _ = m.Update(passwordErrMsg{err: errTest("bad password")})

	if m.input.Value() != "" {
		t.Error("input should be cleared on error")
	}
	if !strings.Contains(m.View(), "bad password") {
		t.Error("should display error message")
	}
}

// menu view tests

func TestMenuViewShowsItems(t *testing.T) {
	m := newMenuModel("1.0")
	view := m.View()

	for _, item := range menuItems {
		if !strings.Contains(view, item) {
			t.Errorf("menu should contain %q", item)
		}
	}
	if !strings.Contains(view, "1.0") {
		t.Error("menu should show version")
	}
}

func TestMenuNavigation(t *testing.T) {
	m := newMenuModel("1.0")

	if m.cursor != 0 {
		t.Fatal("cursor should start at 0")
	}

	// move down
	m, _ = m.Update(keyMsg('j'))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	// move down with arrow
	m, _ = m.Update(specialKey(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}

	// move up
	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	// up arrow
	m, _ = m.Update(specialKey(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}

	// don't go below 0
	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
}

func TestMenuCursorClampMax(t *testing.T) {
	m := newMenuModel("1.0")
	// go to last item
	for i := 0; i < len(menuItems); i++ {
		m, _ = m.Update(keyMsg('j'))
	}
	if m.cursor != len(menuItems)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(menuItems)-1)
	}
}

func TestMenuSelectGenerate(t *testing.T) {
	m := newMenuModel("1.0")
	// cursor at 0 = Generate
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewGenerate {
		t.Errorf("view = %d, want viewGenerate", nav.view)
	}
}

func TestMenuSelectEmail(t *testing.T) {
	m := newMenuModel("1.0")
	m.cursor = 1
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	if _, ok := msg.(quickEmailMsg); !ok {
		t.Error("should emit quickEmailMsg")
	}
}

func TestMenuSelectBrowse(t *testing.T) {
	m := newMenuModel("1.0")
	m.cursor = 2
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewList {
		t.Errorf("view = %d, want viewList", nav.view)
	}
}

func TestMenuQuitOnQ(t *testing.T) {
	m := newMenuModel("1.0")
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestMenuQuitFromLastItem(t *testing.T) {
	m := newMenuModel("1.0")
	m.cursor = len(menuItems) - 1 // Quit item
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("selecting Quit should produce command")
	}
}

// generate view tests

func TestGenerateViewShowsFields(t *testing.T) {
	id := testIdentity()
	m := newGenerateModel(id)
	view := m.View()

	checks := []string{id.FirstName, id.Email, id.Phone, id.Password, id.City}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("view should contain %q", c)
		}
	}
}

func TestGenerateNavigation(t *testing.T) {
	m := newGenerateModel(testIdentity())

	if m.cursor != 0 {
		t.Fatal("cursor should start at 0")
	}

	m, _ = m.Update(keyMsg('j'))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestGenerateBackToMenu(t *testing.T) {
	m := newGenerateModel(testIdentity())
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewMenu {
		t.Errorf("view = %d, want viewMenu", nav.view)
	}
}

func TestGenerateNewIdentity(t *testing.T) {
	m := newGenerateModel(testIdentity())
	_, cmd := m.Update(keyMsg('n'))
	if cmd == nil {
		t.Fatal("n should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewGenerate {
		t.Errorf("view = %d, want viewGenerate", nav.view)
	}
}

func TestGenerateSave(t *testing.T) {
	id := testIdentity()
	m := newGenerateModel(id)
	_, cmd := m.Update(keyMsg('s'))
	if cmd == nil {
		t.Fatal("s should produce command")
	}
	msg := cmd()
	save, ok := msg.(saveIdentityMsg)
	if !ok {
		t.Fatal("should emit saveIdentityMsg")
	}
	if save.identity.ID != id.ID {
		t.Errorf("saved identity ID = %q, want %q", save.identity.ID, id.ID)
	}
}

func TestGenerateQuit(t *testing.T) {
	m := newGenerateModel(testIdentity())
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from generate view")
	}
}

func TestGenerateSavedFlash(t *testing.T) {
	m := newGenerateModel(testIdentity())
	m, _ = m.Update(identitySavedMsg{})
	if m.flash != "saved" {
		t.Errorf("flash = %q, want %q", m.flash, "saved")
	}
}

func TestGenerateFlashClears(t *testing.T) {
	m := newGenerateModel(testIdentity())
	m.flash = "saved"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty after flashMsg, got %q", m.flash)
	}
}

// list view tests

func TestListViewEmpty(t *testing.T) {
	m := newListModel(nil)
	view := m.View()

	if !strings.Contains(view, "no saved identities") {
		t.Error("should show empty state")
	}
}

func TestListViewShowsIdentities(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)
	view := m.View()

	if !strings.Contains(view, "abc12345") {
		t.Error("should show identity ID")
	}
	if !strings.Contains(view, "jane@zburn.id") {
		t.Error("should show email")
	}
}

func TestListNavigation(t *testing.T) {
	ids := []identity.Identity{
		testIdentity(),
		{ID: "second", FirstName: "Bob", LastName: "Smith", Email: "bob@zburn.id", CreatedAt: time.Now()},
	}
	m := newListModel(ids)

	m, _ = m.Update(keyMsg('j'))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestListSelectIdentity(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	view, ok := msg.(viewIdentityMsg)
	if !ok {
		t.Fatal("should emit viewIdentityMsg")
	}
	if view.identity.ID != "abc12345" {
		t.Errorf("identity ID = %q, want %q", view.identity.ID, "abc12345")
	}
}

func TestListDeleteConfirmation(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)

	// press d to start delete
	m, _ = m.Update(keyMsg('d'))
	if !m.confirming {
		t.Error("should be in confirming state")
	}
	if !strings.Contains(m.View(), "delete? y/n") {
		t.Error("should show delete confirmation")
	}

	// press n to cancel
	m, _ = m.Update(keyMsg('n'))
	if m.confirming {
		t.Error("should cancel confirmation on n")
	}
}

func TestListDeleteConfirmed(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)

	m, _ = m.Update(keyMsg('d'))
	_, cmd := m.Update(keyMsg('y'))
	if cmd == nil {
		t.Fatal("y should produce delete command")
	}
	msg := cmd()
	del, ok := msg.(deleteIdentityMsg)
	if !ok {
		t.Fatal("should emit deleteIdentityMsg")
	}
	if del.id != "abc12345" {
		t.Errorf("delete ID = %q, want %q", del.id, "abc12345")
	}
}

func TestListBackToMenu(t *testing.T) {
	m := newListModel(nil)
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewMenu {
		t.Errorf("view = %d, want viewMenu", nav.view)
	}
}

func TestListQuit(t *testing.T) {
	m := newListModel(nil)
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from list view")
	}
}

func TestListLoadIdentities(t *testing.T) {
	m := newListModel(nil)
	ids := []identity.Identity{testIdentity()}
	m, _ = m.Update(loadIdentitiesMsg{identities: ids})

	if len(m.identities) != 1 {
		t.Errorf("identities length = %d, want 1", len(m.identities))
	}
	if m.cursor != 0 {
		t.Error("cursor should reset to 0 on load")
	}
}

// detail view tests

func TestDetailViewShowsFields(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	checks := []string{"abc12345", "Jane Doe", "jane@zburn.id", "(555) 123-4567"}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("detail view should contain %q", c)
		}
	}
}

func TestDetailNavigation(t *testing.T) {
	m := newDetailModel(testIdentity())

	m, _ = m.Update(keyMsg('j'))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestDetailBackToList(t *testing.T) {
	m := newDetailModel(testIdentity())
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewList {
		t.Errorf("view = %d, want viewList", nav.view)
	}
}

func TestDetailDeleteConfirmation(t *testing.T) {
	m := newDetailModel(testIdentity())

	m, _ = m.Update(keyMsg('d'))
	if !m.confirm {
		t.Error("should be in confirm state")
	}

	// cancel
	m, _ = m.Update(keyMsg('n'))
	if m.confirm {
		t.Error("should cancel confirmation")
	}
}

func TestDetailDeleteConfirmed(t *testing.T) {
	m := newDetailModel(testIdentity())

	m, _ = m.Update(keyMsg('d'))
	_, cmd := m.Update(keyMsg('y'))
	if cmd == nil {
		t.Fatal("y should produce delete command")
	}
	msg := cmd()
	del, ok := msg.(deleteIdentityMsg)
	if !ok {
		t.Fatal("should emit deleteIdentityMsg")
	}
	if del.id != "abc12345" {
		t.Errorf("delete ID = %q, want %q", del.id, "abc12345")
	}
}

func TestDetailQuit(t *testing.T) {
	m := newDetailModel(testIdentity())
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from detail view")
	}
}

func TestDetailFlashClears(t *testing.T) {
	m := newDetailModel(testIdentity())
	m.flash = "copied!"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty after flashMsg, got %q", m.flash)
	}
}

// root model navigation tests

func TestRootStartsAtPassword(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), true)
	if m.active != viewPassword {
		t.Errorf("active = %d, want viewPassword", m.active)
	}
}

func TestRootNavigateToMenu(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewMenu // simulate post-password

	result, _ := m.Update(navigateMsg{view: viewMenu})
	rm := result.(Model)
	if rm.active != viewMenu {
		t.Errorf("active = %d, want viewMenu", rm.active)
	}
}

func TestRootNavigateToGenerate(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewMenu

	result, _ := m.Update(navigateMsg{view: viewGenerate})
	rm := result.(Model)
	if rm.active != viewGenerate {
		t.Errorf("active = %d, want viewGenerate", rm.active)
	}
	// should have generated an identity
	if rm.generate.identity.ID == "" {
		t.Error("should have generated an identity")
	}
}

func TestRootViewIdentityMsg(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewList

	id := testIdentity()
	result, _ := m.Update(viewIdentityMsg{identity: id})
	rm := result.(Model)
	if rm.active != viewDetail {
		t.Errorf("active = %d, want viewDetail", rm.active)
	}
	if rm.detail.identity.ID != id.ID {
		t.Errorf("detail identity ID = %q, want %q", rm.detail.identity.ID, id.ID)
	}
}

func TestRootQuitFromPassword(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	result, cmd := m.Update(specialKey(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("ctrl+c should quit from password view")
	}
	_ = result
}

func TestRootQuitFromMenu(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewMenu

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from menu")
	}
}

func TestRootQuitFromGenerate(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewGenerate
	m.generate = newGenerateModel(testIdentity())

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from generate")
	}
}

func TestRootQuitFromList(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewList
	m.list = newListModel(nil)

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from list")
	}
}

func TestRootQuitFromDetail(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewDetail
	m.detail = newDetailModel(testIdentity())

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from detail")
	}
}

// navigation flow: menu -> generate -> menu
func TestNavigationMenuGenerateMenu(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewMenu

	// menu -> generate
	result, _ := m.Update(navigateMsg{view: viewGenerate})
	rm := result.(Model)
	if rm.active != viewGenerate {
		t.Fatalf("active = %d, want viewGenerate", rm.active)
	}

	// generate -> menu (via esc)
	result, cmd := rm.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()

	result, _ = result.(Model).Update(msg)
	rm = result.(Model)
	if rm.active != viewMenu {
		t.Errorf("active = %d, want viewMenu", rm.active)
	}
}

// navigation flow: menu -> list -> detail -> list -> menu
func TestNavigationFullLoop(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewList

	id := testIdentity()
	m.list = newListModel([]identity.Identity{id})

	// list -> detail
	result, _ := m.Update(viewIdentityMsg{identity: id})
	rm := result.(Model)
	if rm.active != viewDetail {
		t.Fatalf("active = %d, want viewDetail", rm.active)
	}

	// detail -> list (via esc)
	result, cmd := rm.Update(escKey())
	if cmd == nil {
		t.Fatal("esc from detail should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should be navigateMsg")
	}
	if nav.view != viewList {
		t.Errorf("view = %d, want viewList", nav.view)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is t…"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

func TestIdentityFields(t *testing.T) {
	id := testIdentity()
	fields := identityFields(id)

	if len(fields) != 10 {
		t.Fatalf("fields length = %d, want 10", len(fields))
	}

	// spot check
	if fields[0].label != "id" || fields[0].value != id.ID {
		t.Errorf("field[0] = %v, want id=%s", fields[0], id.ID)
	}
	if fields[2].label != "email" || fields[2].value != id.Email {
		t.Errorf("field[2] = %v, want email=%s", fields[2], id.Email)
	}
}

// errTest is a simple error for testing.
type errTest string

func (e errTest) Error() string { return string(e) }
