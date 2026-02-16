package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

func testCredential() credential.Credential {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	return credential.Credential{
		ID:         "cred-001",
		IdentityID: "abc12345",
		Label:      "GitHub",
		URL:        "https://github.com",
		Username:   "janedoe",
		Password:   "s3cret!Pass",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
		Notes:      "work account",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func testCredentialNoTOTP() credential.Credential {
	c := testCredential()
	c.ID = "cred-002"
	c.Label = "Netflix"
	c.TOTPSecret = ""
	c.Notes = ""
	return c
}

// credential list tests

func TestCredentialListViewEmpty(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	view := m.View()

	if !strings.Contains(view, "no credentials") {
		t.Error("should show empty state")
	}
	if !strings.Contains(view, "credentials (0)") {
		t.Error("should show zero count")
	}
}

func TestCredentialListViewShowsCredentials(t *testing.T) {
	creds := []credential.Credential{testCredential(), testCredentialNoTOTP()}
	m := newCredentialListModel("abc12345", creds)
	view := m.View()

	if !strings.Contains(view, "credentials (2)") {
		t.Error("should show count")
	}
	if !strings.Contains(view, "GitHub") {
		t.Error("should show GitHub label")
	}
	if !strings.Contains(view, "Netflix") {
		t.Error("should show Netflix label")
	}
}

func TestCredentialListSortsByLabel(t *testing.T) {
	creds := []credential.Credential{testCredentialNoTOTP(), testCredential()} // Netflix, GitHub
	m := newCredentialListModel("abc12345", creds)

	// after sort: GitHub, Netflix
	if m.credentials[0].Label != "GitHub" {
		t.Errorf("first credential = %q, want GitHub", m.credentials[0].Label)
	}
	if m.credentials[1].Label != "Netflix" {
		t.Errorf("second credential = %q, want Netflix", m.credentials[1].Label)
	}
}

func TestCredentialListNavigation(t *testing.T) {
	creds := []credential.Credential{testCredential(), testCredentialNoTOTP()}
	m := newCredentialListModel("abc12345", creds)

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

	// clamp at 0
	m, _ = m.Update(keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
}

func TestCredentialListCursorClampMax(t *testing.T) {
	creds := []credential.Credential{testCredential(), testCredentialNoTOTP()}
	m := newCredentialListModel("abc12345", creds)

	for range 5 {
		m, _ = m.Update(keyMsg('j'))
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", m.cursor)
	}
}

func TestCredentialListEnterViewsCredential(t *testing.T) {
	creds := []credential.Credential{testCredential()}
	m := newCredentialListModel("abc12345", creds)

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	view, ok := msg.(viewCredentialMsg)
	if !ok {
		t.Fatalf("should emit viewCredentialMsg, got %T", msg)
	}
	if view.credential.ID != "cred-001" {
		t.Errorf("credential ID = %q, want %q", view.credential.ID, "cred-001")
	}
}

func TestCredentialListEnterEmptyNoop(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("enter on empty list should be noop")
	}
}

func TestCredentialListAdd(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)

	_, cmd := m.Update(keyMsg('a'))
	if cmd == nil {
		t.Fatal("a should produce command")
	}
	msg := cmd()
	add, ok := msg.(addCredentialMsg)
	if !ok {
		t.Fatalf("should emit addCredentialMsg, got %T", msg)
	}
	if add.identityID != "abc12345" {
		t.Errorf("identityID = %q, want %q", add.identityID, "abc12345")
	}
}

func TestCredentialListDeleteConfirmation(t *testing.T) {
	creds := []credential.Credential{testCredential()}
	m := newCredentialListModel("abc12345", creds)

	m, _ = m.Update(keyMsg('d'))
	if !m.confirm {
		t.Error("should be in confirm state")
	}
	if !strings.Contains(m.View(), "delete credential") {
		t.Error("should show delete confirmation")
	}
	if !strings.Contains(m.View(), "GitHub") {
		t.Error("confirmation should include label")
	}

	// cancel
	m, _ = m.Update(keyMsg('n'))
	if m.confirm {
		t.Error("should cancel confirmation on n")
	}
}

func TestCredentialListDeleteConfirmed(t *testing.T) {
	creds := []credential.Credential{testCredential()}
	m := newCredentialListModel("abc12345", creds)

	m, _ = m.Update(keyMsg('d'))
	_, cmd := m.Update(keyMsg('y'))
	if cmd == nil {
		t.Fatal("y should produce delete command")
	}
	msg := cmd()
	del, ok := msg.(deleteCredentialMsg)
	if !ok {
		t.Fatalf("should emit deleteCredentialMsg, got %T", msg)
	}
	if del.id != "cred-001" {
		t.Errorf("delete ID = %q, want %q", del.id, "cred-001")
	}
}

func TestCredentialListDeleteEmptyNoop(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	m, _ = m.Update(keyMsg('d'))
	if m.confirm {
		t.Error("should not enter confirm on empty list")
	}
}

func TestCredentialListBackToDetail(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatalf("should emit navigateMsg, got %T", msg)
	}
	if nav.view != viewDetail {
		t.Errorf("view = %d, want viewDetail", nav.view)
	}
}

func TestCredentialListQuit(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestCredentialListFlashClears(t *testing.T) {
	m := newCredentialListModel("abc12345", nil)
	m.flash = "something"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

// credential detail tests

func TestCredentialDetailViewShowsFields(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	checks := []string{"GitHub", "https://github.com", "janedoe", "work account"}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("detail view should contain %q", c)
		}
	}
}

func TestCredentialDetailPasswordMasked(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if strings.Contains(view, "s3cret!Pass") {
		t.Error("password should be masked by default")
	}
	if !strings.Contains(view, "••••••••") {
		t.Error("should show masked password")
	}
}

func TestCredentialDetailPasswordReveal(t *testing.T) {
	m := newCredentialDetailModel(testCredential())

	m, _ = m.Update(keyMsg('r'))
	if !m.revealed {
		t.Error("r should toggle reveal")
	}
	view := m.View()
	if !strings.Contains(view, "s3cret!Pass") {
		t.Error("password should be visible when revealed")
	}

	// toggle back
	m, _ = m.Update(keyMsg('r'))
	if m.revealed {
		t.Error("r should toggle back to masked")
	}
}

func TestCredentialDetailTOTPShown(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if !strings.Contains(view, "totp") {
		t.Error("should show totp section")
	}
	// should show a 6-digit code
	if m.totpCode == "" {
		t.Error("totp code should be generated")
	}
	if len(m.totpCode) != 6 {
		t.Errorf("totp code length = %d, want 6", len(m.totpCode))
	}
}

func TestCredentialDetailTOTPHiddenWhenNoSecret(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	view := m.View()

	if strings.Contains(view, "copy totp") {
		t.Error("should not show totp copy when no secret")
	}
}

func TestCredentialDetailTOTPTickRefreshes(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	code1 := m.totpCode

	m, cmd := m.Update(totpTickMsg{})
	if cmd == nil {
		t.Error("tick should schedule next tick when TOTP secret present")
	}
	// code should still be valid (same 30s window)
	if m.totpCode == "" {
		t.Error("totp code should still be set after tick")
	}
	_ = code1
}

func TestCredentialDetailTOTPTickNoSecretNoCmd(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	_, cmd := m.Update(totpTickMsg{})
	if cmd != nil {
		t.Error("tick should not schedule next tick when no TOTP secret")
	}
}

func TestCredentialDetailCopyTOTPNoSecret(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	m, _ = m.Update(keyMsg('t'))
	// should be noop
	if m.flash != "" {
		t.Errorf("flash should be empty for TOTP copy with no secret, got %q", m.flash)
	}
}

func TestCredentialDetailEdit(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	_, cmd := m.Update(keyMsg('e'))
	if cmd == nil {
		t.Fatal("e should produce command")
	}
	msg := cmd()
	edit, ok := msg.(editCredentialMsg)
	if !ok {
		t.Fatalf("should emit editCredentialMsg, got %T", msg)
	}
	if edit.credential.ID != "cred-001" {
		t.Errorf("credential ID = %q, want %q", edit.credential.ID, "cred-001")
	}
}

func TestCredentialDetailDeleteConfirmation(t *testing.T) {
	m := newCredentialDetailModel(testCredential())

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

func TestCredentialDetailDeleteConfirmed(t *testing.T) {
	m := newCredentialDetailModel(testCredential())

	m, _ = m.Update(keyMsg('d'))
	_, cmd := m.Update(keyMsg('y'))
	if cmd == nil {
		t.Fatal("y should produce delete command")
	}
	msg := cmd()
	del, ok := msg.(deleteCredentialMsg)
	if !ok {
		t.Fatalf("should emit deleteCredentialMsg, got %T", msg)
	}
	if del.id != "cred-001" {
		t.Errorf("delete ID = %q, want %q", del.id, "cred-001")
	}
}

func TestCredentialDetailBackToList(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatalf("should emit navigateMsg, got %T", msg)
	}
	if nav.view != viewCredentialList {
		t.Errorf("view = %d, want viewCredentialList", nav.view)
	}
}

func TestCredentialDetailQuit(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestCredentialDetailFlashClears(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	m.flash = "something"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

func TestCredentialDetailTimestamps(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if !strings.Contains(view, "created") {
		t.Error("should show created timestamp")
	}
	if !strings.Contains(view, "updated") {
		t.Error("should show updated timestamp")
	}
}

func TestCredentialDetailHelpShowsTOTP(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if !strings.Contains(view, "t copy totp") {
		t.Error("help should show TOTP copy when secret present")
	}
}

func TestCredentialDetailHelpNoTOTP(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	view := m.View()

	if strings.Contains(view, "t copy totp") {
		t.Error("help should not show TOTP copy when no secret")
	}
}

func TestCredentialDetailNotesShown(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if !strings.Contains(view, "work account") {
		t.Error("should show notes")
	}
}

func TestCredentialDetailNotesHiddenWhenEmpty(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	view := m.View()

	// Notes is empty, should not show a notes label with empty value
	// (fieldLine still shows "notes" label, but the credential has no notes)
	// The view only adds notes line if Notes != ""
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "notes") && strings.Contains(line, "work") {
			t.Error("should not show notes content when empty")
		}
	}
}

// credential form tests

func TestCredentialFormAddView(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	view := m.View()

	if !strings.Contains(view, "add credential") {
		t.Error("should show add title")
	}
	for _, label := range fieldLabels {
		if !strings.Contains(view, label) {
			t.Errorf("form should contain %q", label)
		}
	}
}

func TestCredentialFormEditView(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel("abc12345", &c)
	view := m.View()

	if !strings.Contains(view, "edit credential") {
		t.Error("should show edit title")
	}
	if !m.editing {
		t.Error("should be in editing mode")
	}
}

func TestCredentialFormEditPreloads(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel("abc12345", &c)

	if m.inputs[fieldLabel].Value() != "GitHub" {
		t.Errorf("label = %q, want %q", m.inputs[fieldLabel].Value(), "GitHub")
	}
	if m.inputs[fieldURL].Value() != "https://github.com" {
		t.Errorf("url = %q, want %q", m.inputs[fieldURL].Value(), "https://github.com")
	}
	if m.inputs[fieldUsername].Value() != "janedoe" {
		t.Errorf("username = %q, want %q", m.inputs[fieldUsername].Value(), "janedoe")
	}
	if m.inputs[fieldPassword].Value() != "s3cret!Pass" {
		t.Errorf("password = %q, want %q", m.inputs[fieldPassword].Value(), "s3cret!Pass")
	}
	if m.inputs[fieldTOTPSecret].Value() != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp = %q, want %q", m.inputs[fieldTOTPSecret].Value(), "JBSWY3DPEHPK3PXP")
	}
	if m.inputs[fieldNotes].Value() != "work account" {
		t.Errorf("notes = %q, want %q", m.inputs[fieldNotes].Value(), "work account")
	}
}

func TestCredentialFormTabCycles(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)

	if m.focus != 0 {
		t.Fatalf("focus should start at 0")
	}

	m, _ = m.Update(tabKey())
	if m.focus != 1 {
		t.Errorf("focus = %d, want 1", m.focus)
	}

	// shift+tab goes back
	m, _ = m.Update(shiftTabKey())
	if m.focus != 0 {
		t.Errorf("focus = %d, want 0", m.focus)
	}

	// shift+tab wraps to last
	m, _ = m.Update(shiftTabKey())
	if m.focus != fieldCount-1 {
		t.Errorf("focus = %d, want %d", m.focus, fieldCount-1)
	}
}

func TestCredentialFormSubmitRequiresLabel(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	// label is empty
	m, _ = m.Update(enterKey())
	if m.flash != "label is required" {
		t.Errorf("flash = %q, want %q", m.flash, "label is required")
	}
}

func TestCredentialFormSubmitInvalidTOTP(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	m.inputs[fieldLabel].SetValue("Test")
	m.inputs[fieldTOTPSecret].SetValue("!!!invalid!!!")

	m, _ = m.Update(enterKey())
	if m.flash != "invalid totp secret (must be base32)" {
		t.Errorf("flash = %q, want totp error", m.flash)
	}
}

func TestCredentialFormSubmitValidAdd(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	m.inputs[fieldLabel].SetValue("GitHub")
	m.inputs[fieldURL].SetValue("https://github.com")
	m.inputs[fieldUsername].SetValue("janedoe")
	m.inputs[fieldPassword].SetValue("pass123")

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	save, ok := msg.(saveCredentialMsg)
	if !ok {
		t.Fatalf("should emit saveCredentialMsg, got %T", msg)
	}
	if save.credential.Label != "GitHub" {
		t.Errorf("label = %q, want %q", save.credential.Label, "GitHub")
	}
	if save.credential.IdentityID != "abc12345" {
		t.Errorf("identityID = %q, want %q", save.credential.IdentityID, "abc12345")
	}
	if save.credential.ID == "" {
		t.Error("should generate an ID")
	}
	if save.credential.CreatedAt.IsZero() {
		t.Error("should set CreatedAt")
	}
}

func TestCredentialFormSubmitValidEdit(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel("abc12345", &c)
	m.inputs[fieldLabel].SetValue("GitHub Updated")

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	save, ok := msg.(saveCredentialMsg)
	if !ok {
		t.Fatalf("should emit saveCredentialMsg, got %T", msg)
	}
	if save.credential.ID != "cred-001" {
		t.Errorf("should keep existing ID, got %q", save.credential.ID)
	}
	if save.credential.Label != "GitHub Updated" {
		t.Errorf("label = %q, want %q", save.credential.Label, "GitHub Updated")
	}
	if save.credential.CreatedAt != c.CreatedAt {
		t.Error("should preserve original CreatedAt")
	}
	if save.credential.UpdatedAt.Equal(c.UpdatedAt) {
		t.Error("should update UpdatedAt")
	}
}

func TestCredentialFormSubmitValidBase32TOTP(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	m.inputs[fieldLabel].SetValue("Test")
	m.inputs[fieldTOTPSecret].SetValue("JBSWY3DPEHPK3PXP")

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce save command")
	}
	msg := cmd()
	save, ok := msg.(saveCredentialMsg)
	if !ok {
		t.Fatalf("should emit saveCredentialMsg, got %T", msg)
	}
	if save.credential.TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp = %q, want %q", save.credential.TOTPSecret, "JBSWY3DPEHPK3PXP")
	}
}

func TestCredentialFormBackFromAdd(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatalf("should emit navigateMsg, got %T", msg)
	}
	if nav.view != viewCredentialList {
		t.Errorf("view = %d, want viewCredentialList", nav.view)
	}
}

func TestCredentialFormBackFromEdit(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel("abc12345", &c)
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	view, ok := msg.(viewCredentialMsg)
	if !ok {
		t.Fatalf("should emit viewCredentialMsg from edit, got %T", msg)
	}
	if view.credential.ID != "cred-001" {
		t.Errorf("credential ID = %q, want %q", view.credential.ID, "cred-001")
	}
}

func TestCredentialFormCtrlCQuits(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	_, cmd := m.Update(specialKey(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestCredentialFormFlashClears(t *testing.T) {
	m := newCredentialFormModel("abc12345", nil)
	m.flash = "something"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

// isValidBase32 tests

func TestIsValidBase32(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"JBSWY3DPEHPK3PXP", true},
		{"jbswy3dpehpk3pxp", true},
		{"JBSW Y3DP EHPK 3PXP", true},
		{"!!!invalid!!!", false},
		{"", true}, // empty is valid base32
	}

	for _, tt := range tests {
		got := isValidBase32(tt.input)
		if got != tt.want {
			t.Errorf("isValidBase32(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// credentialHexID tests

func TestCredentialHexIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		id := credentialHexID()
		if len(id) != 8 {
			t.Fatalf("hex ID length = %d, want 8", len(id))
		}
		if seen[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

// detail view credential count tests

func TestDetailViewShowsCredentialCount(t *testing.T) {
	m := newDetailModel(testIdentity())
	m.credentialCount = 3
	view := m.View()

	if !strings.Contains(view, "(3) credentials") {
		t.Error("should show credential count inline")
	}
	if !strings.Contains(view, "w to view") {
		t.Error("should show w shortcut hint")
	}
}

func TestDetailViewCredentialCountZero(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	if !strings.Contains(view, "no credentials") {
		t.Error("should show no credentials message")
	}
	if !strings.Contains(view, "a to add") {
		t.Error("should show add hint when no credentials")
	}
}

func TestDetailWNavigatesToCredentials(t *testing.T) {
	m := newDetailModel(testIdentity())
	_, cmd := m.Update(keyMsg('w'))
	if cmd == nil {
		t.Fatal("w should produce command")
	}
	msg := cmd()
	view, ok := msg.(viewCredentialsMsg)
	if !ok {
		t.Fatalf("should emit viewCredentialsMsg, got %T", msg)
	}
	if view.identityID != "abc12345" {
		t.Errorf("identityID = %q, want %q", view.identityID, "abc12345")
	}
}

func TestDetailHelpShowsCredentials(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	if !strings.Contains(view, "w credentials") {
		t.Error("help should show w credentials shortcut")
	}
}

// root model credential view tests

func TestRootViewCredentialMsg(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialList

	c := testCredential()
	result, cmd := m.Update(viewCredentialMsg{credential: c})
	rm := result.(Model)
	if rm.active != viewCredentialDetail {
		t.Errorf("active = %d, want viewCredentialDetail", rm.active)
	}
	if rm.credentialDetail.credential.ID != c.ID {
		t.Errorf("credential ID = %q, want %q", rm.credentialDetail.credential.ID, c.ID)
	}
	_ = cmd
}

func TestRootAddCredentialMsg(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialList

	result, _ := m.Update(addCredentialMsg{identityID: "abc12345"})
	rm := result.(Model)
	if rm.active != viewCredentialForm {
		t.Errorf("active = %d, want viewCredentialForm", rm.active)
	}
	if rm.credentialForm.identityID != "abc12345" {
		t.Errorf("identityID = %q, want %q", rm.credentialForm.identityID, "abc12345")
	}
	if rm.credentialForm.editing {
		t.Error("should not be in editing mode for add")
	}
}

func TestRootEditCredentialMsg(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialDetail

	c := testCredential()
	result, _ := m.Update(editCredentialMsg{credential: c})
	rm := result.(Model)
	if rm.active != viewCredentialForm {
		t.Errorf("active = %d, want viewCredentialForm", rm.active)
	}
	if !rm.credentialForm.editing {
		t.Error("should be in editing mode")
	}
	if rm.credentialForm.existing.ID != c.ID {
		t.Errorf("existing ID = %q, want %q", rm.credentialForm.existing.ID, c.ID)
	}
}

func TestRootQuitFromCredentialList(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialList
	m.credentialList = newCredentialListModel("abc12345", nil)

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from credential list")
	}
}

func TestRootQuitFromCredentialDetail(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialDetail
	m.credentialDetail = newCredentialDetailModel(testCredential())

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from credential detail")
	}
}

// helpers

func tabKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

func shiftTabKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyShiftTab}
}
