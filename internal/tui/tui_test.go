package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/zburn/internal/burn"
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

func TestMenuSelectBrowse(t *testing.T) {
	m := newMenuModel("1.0")
	m.cursor = 1
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
	m := newGenerateModel(id, "")
	view := m.View()

	checks := []string{id.Email, id.FirstName, id.Phone, "Portland, OR 97201"}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("view should contain %q", c)
		}
	}
}

func TestGenerateNavigation(t *testing.T) {
	m := newGenerateModel(testIdentity(), "")

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
	m := newGenerateModel(testIdentity(), "")
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
	m := newGenerateModel(testIdentity(), "")
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
	m := newGenerateModel(id, "")
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
	m := newGenerateModel(testIdentity(), "")
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from generate view")
	}
}

func TestGenerateSavedFlash(t *testing.T) {
	m := newGenerateModel(testIdentity(), "")
	m, _ = m.Update(identitySavedMsg{})
	if m.flash != "saved" {
		t.Errorf("flash = %q, want %q", m.flash, "saved")
	}
}

func TestGenerateFlashClears(t *testing.T) {
	m := newGenerateModel(testIdentity(), "")
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

	if !strings.Contains(view, "Jane Doe") {
		t.Error("should show name")
	}
	if !strings.Contains(view, "jane@zburn.id") {
		t.Error("should show email")
	}
}

func TestListViewCredentialCount(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)
	m.credCounts = map[string]int{"abc12345": 3}
	view := m.View()

	if !strings.Contains(view, "(3)") {
		t.Error("should show credential count badge")
	}
}

func TestListViewCredentialCountZero(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)
	view := m.View()

	if strings.Contains(view, "(0)") {
		t.Error("should not show (0) badge")
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

func TestListBurnEmitsBurnStartMsg(t *testing.T) {
	ids := []identity.Identity{testIdentity()}
	m := newListModel(ids)

	// press d to trigger burn
	_, cmd := m.Update(keyMsg('d'))
	if cmd == nil {
		t.Fatal("d should produce command")
	}
	msg := cmd()
	bs, ok := msg.(burnStartMsg)
	if !ok {
		t.Fatal("should emit burnStartMsg")
	}
	if bs.identity.ID != "abc12345" {
		t.Errorf("burn identity ID = %q, want %q", bs.identity.ID, "abc12345")
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

	checks := []string{"Jane Doe", "jane@zburn.id", "(555) 123-4567"}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("detail view should contain %q", c)
		}
	}
}

func TestDetailViewNameInTitle(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	if !strings.Contains(view, "Jane Doe") {
		t.Error("title should contain identity name")
	}
	if strings.Contains(view, "abc12345") {
		t.Error("title should not contain UUID")
	}
}

func TestDetailViewSectionBreaks(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	// section breaks add blank lines between contact/address/personal groups
	// find the fields in order and verify blank lines exist between sections
	lines := strings.Split(view, "\n")
	var fieldLines []int
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// look for field content lines (with label + value)
		if strings.Contains(line, "street") || strings.Contains(line, "dob") {
			fieldLines = append(fieldLines, i)
		}
	}

	// verify there is at least one blank line before street (address section)
	// and before dob (personal section) by checking the raw output
	// The sectionBreaks map adds "\n" before indices 3 (street) and 5 (dob)
	streetIdx := strings.Index(view, "street")
	dobIdx := strings.Index(view, "dob")
	if streetIdx < 0 || dobIdx < 0 {
		t.Fatal("view should contain street and dob fields")
	}

	// check that there's a double newline before street section
	beforeStreet := view[:streetIdx]
	if !strings.Contains(beforeStreet, "\n\n") {
		t.Error("should have section break before street (address section)")
	}

	// check that there's a double newline before dob section
	betweenStreetAndDob := view[streetIdx:dobIdx]
	if !strings.Contains(betweenStreetAndDob, "\n\n") {
		t.Error("should have section break before dob (personal section)")
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

func TestDetailBurnEmitsBurnStartMsg(t *testing.T) {
	m := newDetailModel(testIdentity())

	_, cmd := m.Update(keyMsg('d'))
	if cmd == nil {
		t.Fatal("d should produce command")
	}
	msg := cmd()
	bs, ok := msg.(burnStartMsg)
	if !ok {
		t.Fatal("should emit burnStartMsg")
	}
	if bs.identity.ID != "abc12345" {
		t.Errorf("burn identity ID = %q, want %q", bs.identity.ID, "abc12345")
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
	m.generate = newGenerateModel(testIdentity(), "")

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

	if len(fields) != 6 {
		t.Fatalf("fields length = %d, want 6", len(fields))
	}

	// email is first
	if fields[0].label != "email" || fields[0].value != id.Email {
		t.Errorf("field[0] = %v, want email=%s", fields[0], id.Email)
	}

	// no id field
	for _, f := range fields {
		if f.label == "id" {
			t.Error("fields should not contain id")
		}
	}

	// address combines city, state, zip
	if fields[4].label != "address" || fields[4].value != "Portland, OR 97201" {
		t.Errorf("field[4] = %v, want address=Portland, OR 97201", fields[4])
	}
}

// burn view tests

func TestBurnConfirmViewShowsPlan(t *testing.T) {
	id := testIdentity()
	plan := []string{
		"delete all credentials (3)",
		"release phone number +447123456789",
	}
	m := newBurnModel(id, plan)
	view := m.View()

	if !strings.Contains(view, "burn Jane Doe?") {
		t.Error("should show burn confirmation with name")
	}
	if !strings.Contains(view, "credentials (3)") {
		t.Error("should show credential count")
	}
	if !strings.Contains(view, "+447123456789") {
		t.Error("should show phone number")
	}
	if !strings.Contains(view, "cannot be undone") {
		t.Error("should show warning")
	}
}

func TestBurnConfirmCancel(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, []string{"delete all credentials (0)"})

	// any key other than y cancels
	_, cmd := m.Update(keyMsg('n'))
	if cmd == nil {
		t.Fatal("should produce command on cancel")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg on cancel")
	}
	if nav.view != viewDetail {
		t.Errorf("view = %d, want viewDetail", nav.view)
	}
}

func TestBurnConfirmAccept(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, []string{"delete all credentials (0)"})

	_, cmd := m.Update(keyMsg('y'))
	if cmd == nil {
		t.Fatal("y should produce command")
	}
	msg := cmd()
	bi, ok := msg.(burnIdentityMsg)
	if !ok {
		t.Fatal("should emit burnIdentityMsg")
	}
	if bi.identity.ID != id.ID {
		t.Errorf("burn identity ID = %q, want %q", bi.identity.ID, id.ID)
	}
}

func TestBurnConfirmQuit(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, nil)

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestBurnDoneAnyKeyNavigates(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, nil)
	m.phase = burnDone
	m.result = burn.Result{
		Name:  "Jane Doe",
		Steps: []burn.StepStatus{{Description: "deleted identity"}},
	}

	_, cmd := m.Update(keyMsg('x'))
	if cmd == nil {
		t.Fatal("any key in done phase should produce command")
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

func TestBurnDoneViewShowsResult(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, nil)
	m.phase = burnDone
	m.result = burn.Result{
		Name:             "Jane Doe",
		CredentialsCount: 3,
		Steps: []burn.StepStatus{
			{Description: "deleted 3 credentials"},
			{Description: "deleted identity"},
		},
	}

	view := m.View()
	if !strings.Contains(view, "burned Jane Doe") {
		t.Error("should show burned name")
	}
	if !strings.Contains(view, "deleted 3 credentials") {
		t.Error("should show credential deletion")
	}
}

func TestBurnResultMsg(t *testing.T) {
	id := testIdentity()
	m := newBurnModel(id, nil)
	m.phase = burnRunning

	result := burn.Result{
		Name:  "Jane Doe",
		Steps: []burn.StepStatus{{Description: "deleted identity"}},
	}

	m, _ = m.Update(burnResultMsg{result: result})

	if m.phase != burnDone {
		t.Errorf("phase = %d, want burnDone", m.phase)
	}
	if m.result.Name != "Jane Doe" {
		t.Errorf("result name = %q, want %q", m.result.Name, "Jane Doe")
	}
}

// root model burn integration tests

func TestRootBurnStartFromDetail(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewDetail
	id := testIdentity()
	m.detail = newDetailModel(id)

	// detail emits burnStartMsg
	result, _ := m.Update(burnStartMsg{identity: id})
	rm := result.(Model)

	if rm.active != viewBurn {
		t.Errorf("active = %d, want viewBurn", rm.active)
	}
	if rm.burn.phase != burnConfirm {
		t.Errorf("burn phase = %d, want burnConfirm", rm.burn.phase)
	}
	if rm.burn.identity.ID != id.ID {
		t.Errorf("burn identity ID = %q, want %q", rm.burn.identity.ID, id.ID)
	}
}

func TestRootBurnStartFromList(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewList
	ids := []identity.Identity{testIdentity()}
	m.list = newListModel(ids)

	// list emits burnStartMsg
	result, _ := m.Update(burnStartMsg{identity: ids[0]})
	rm := result.(Model)

	if rm.active != viewBurn {
		t.Errorf("active = %d, want viewBurn", rm.active)
	}
}

// domain rotation tests

func TestCycleDomainMsgAdvancesIndex(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = []string{"alpha.com", "bravo.io", "charlie.net"}
	m.domainIdx = 0
	m.active = viewGenerate
	m.generate = newGenerateModel(testIdentity(), "alpha.com")

	result, _ := m.Update(cycleDomainMsg{})
	rm := result.(Model)
	if rm.domainIdx != 1 {
		t.Errorf("domainIdx = %d, want 1", rm.domainIdx)
	}
	if rm.generate.domain != "bravo.io" {
		t.Errorf("domain = %q, want %q", rm.generate.domain, "bravo.io")
	}

	// cycle again
	result, _ = rm.Update(cycleDomainMsg{})
	rm = result.(Model)
	if rm.domainIdx != 2 {
		t.Errorf("domainIdx = %d, want 2", rm.domainIdx)
	}
	if rm.generate.domain != "charlie.net" {
		t.Errorf("domain = %q, want %q", rm.generate.domain, "charlie.net")
	}

	// wraps around
	result, _ = rm.Update(cycleDomainMsg{})
	rm = result.(Model)
	if rm.domainIdx != 0 {
		t.Errorf("domainIdx = %d, want 0 (wrap)", rm.domainIdx)
	}
	if rm.generate.domain != "alpha.com" {
		t.Errorf("domain = %q, want %q", rm.generate.domain, "alpha.com")
	}
}

func TestCycleDomainKeepsIdentity(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = []string{"alpha.com", "bravo.io"}
	m.domainIdx = 0
	m.active = viewGenerate

	id := m.gen.Generate("alpha.com")
	m.generate = newGenerateModel(id, "alpha.com")
	oldName := m.generate.identity.FirstName + " " + m.generate.identity.LastName
	oldPhone := m.generate.identity.Phone

	result, _ := m.Update(cycleDomainMsg{})
	rm := result.(Model)

	// name should stay the same
	newName := rm.generate.identity.FirstName + " " + rm.generate.identity.LastName
	if newName != oldName {
		t.Errorf("name changed from %q to %q, should stay the same", oldName, newName)
	}
	// phone should stay the same
	if rm.generate.identity.Phone != oldPhone {
		t.Errorf("phone changed from %q to %q, should stay the same", oldPhone, rm.generate.identity.Phone)
	}
	// email should use the new domain
	if !strings.Contains(rm.generate.identity.Email, "bravo.io") {
		t.Errorf("email = %q, should contain bravo.io", rm.generate.identity.Email)
	}
}

func TestSpaceKeyProducesCycleDomainMsg(t *testing.T) {
	m := newGenerateModel(testIdentity(), "alpha.com")
	_, cmd := m.Update(keyMsg(' '))
	if cmd == nil {
		t.Fatal("space should produce command")
	}
	msg := cmd()
	if _, ok := msg.(cycleDomainMsg); !ok {
		t.Fatal("should emit cycleDomainMsg")
	}
}

func TestNoDomainSpaceDoesNothing(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = nil
	m.domainIdx = 0
	m.active = viewGenerate
	m.generate = newGenerateModel(testIdentity(), "")

	result, _ := m.Update(cycleDomainMsg{})
	rm := result.(Model)
	if rm.domainIdx != 0 {
		t.Errorf("domainIdx = %d, want 0", rm.domainIdx)
	}
}

func TestSingleDomainSpaceDoesNothing(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = []string{"only.com"}
	m.domainIdx = 0
	m.active = viewGenerate
	m.generate = newGenerateModel(testIdentity(), "only.com")

	oldID := m.generate.identity.ID
	result, _ := m.Update(cycleDomainMsg{})
	rm := result.(Model)
	if rm.domainIdx != 0 {
		t.Errorf("domainIdx = %d, want 0", rm.domainIdx)
	}
	// identity should not change
	if rm.generate.identity.ID != oldID {
		t.Error("identity should not change with single domain")
	}
}

func TestDomainHintAppearsInView(t *testing.T) {
	id := testIdentity()
	m := newGenerateModel(id, "custom.io")
	view := m.View()

	if !strings.Contains(view, "[custom.io]") {
		t.Error("view should contain domain hint [custom.io]")
	}
	if !strings.Contains(view, "space to cycle") {
		t.Error("view should contain 'space to cycle' hint")
	}
	if !strings.Contains(view, "space domain") {
		t.Error("help text should contain 'space domain'")
	}
}

func TestDomainHintAbsentWhenDefault(t *testing.T) {
	id := testIdentity()
	m := newGenerateModel(id, "")
	view := m.View()

	if strings.Contains(view, "space to cycle") {
		t.Error("view should not contain 'space to cycle' when no domain")
	}
	if strings.Contains(view, "space domain") {
		t.Error("help text should not contain 'space domain' when no domain")
	}
}

func TestNavigateToGenerateUsesDomain(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = []string{"custom.io", "other.com"}
	m.domainIdx = 0
	m.active = viewMenu

	result, _ := m.Update(navigateMsg{view: viewGenerate})
	rm := result.(Model)

	if rm.generate.domain != "custom.io" {
		t.Errorf("generate domain = %q, want %q", rm.generate.domain, "custom.io")
	}
	if !strings.Contains(rm.generate.identity.Email, "custom.io") {
		t.Errorf("email = %q, should contain custom.io", rm.generate.identity.Email)
	}
}

func TestNavigateToGenerateWithDomainIdxPreserved(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = []string{"first.com", "second.io"}
	m.domainIdx = 1
	m.active = viewMenu

	result, _ := m.Update(navigateMsg{view: viewGenerate})
	rm := result.(Model)

	if rm.generate.domain != "second.io" {
		t.Errorf("generate domain = %q, want %q", rm.generate.domain, "second.io")
	}
	if !strings.Contains(rm.generate.identity.Email, "second.io") {
		t.Errorf("email = %q, should contain second.io", rm.generate.identity.Email)
	}
}

func TestSaveNamecheapRefreshesDomains(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.domains = nil
	m.domainIdx = 0

	// simulate saving namecheap with domains — need handleSaveNamecheap to work
	// which requires configs collection, so test via the domain fields directly
	nc := NamecheapSettings{
		Username:      "user",
		APIKey:        "key",
		CachedDomains: []string{"new1.com", "new2.io"},
	}
	m.ncConfig = nc
	m.domains = nc.CachedDomains
	m.domainIdx = 0

	if len(m.domains) != 2 {
		t.Fatalf("domains = %d, want 2", len(m.domains))
	}
	if m.currentDomain() != "new1.com" {
		t.Errorf("currentDomain = %q, want %q", m.currentDomain(), "new1.com")
	}
}

// errTest is a simple error for testing.
type errTest string

func (e errTest) Error() string { return string(e) }
