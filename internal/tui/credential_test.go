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
	m := newCredentialListModel(testIdentity(), nil)
	view := m.View()

	if !strings.Contains(view, "no credentials") {
		t.Error("should show empty state")
	}
	if !strings.Contains(view, "(0) credentials") {
		t.Error("should show zero count")
	}
}

func TestCredentialListViewShowsCredentials(t *testing.T) {
	creds := []credential.Credential{testCredential(), testCredentialNoTOTP()}
	m := newCredentialListModel(testIdentity(), creds)
	view := m.View()

	if !strings.Contains(view, "(2) credentials") {
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
	m := newCredentialListModel(testIdentity(), creds)

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
	m := newCredentialListModel(testIdentity(), creds)

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
	m := newCredentialListModel(testIdentity(), creds)

	for range 5 {
		m, _ = m.Update(keyMsg('j'))
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", m.cursor)
	}
}

func TestCredentialListEnterViewsCredential(t *testing.T) {
	creds := []credential.Credential{testCredential()}
	m := newCredentialListModel(testIdentity(), creds)

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
	m := newCredentialListModel(testIdentity(), nil)
	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("enter on empty list should be noop")
	}
}

func TestCredentialListAdd(t *testing.T) {
	m := newCredentialListModel(testIdentity(), nil)

	_, cmd := m.Update(keyMsg('a'))
	if cmd == nil {
		t.Fatal("a should produce command")
	}
	msg := cmd()
	add, ok := msg.(addCredentialMsg)
	if !ok {
		t.Fatalf("should emit addCredentialMsg, got %T", msg)
	}
	if add.identity.ID != "abc12345" {
		t.Errorf("identity ID = %q, want %q", add.identity.ID, "abc12345")
	}
}

func TestCredentialListDeleteConfirmation(t *testing.T) {
	creds := []credential.Credential{testCredential()}
	m := newCredentialListModel(testIdentity(), creds)

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
	m := newCredentialListModel(testIdentity(), creds)

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
	m := newCredentialListModel(testIdentity(), nil)
	m, _ = m.Update(keyMsg('d'))
	if m.confirm {
		t.Error("should not enter confirm on empty list")
	}
}

func TestCredentialListBackToDetail(t *testing.T) {
	m := newCredentialListModel(testIdentity(), nil)
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
	m := newCredentialListModel(testIdentity(), nil)
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestCredentialListFlashClears(t *testing.T) {
	m := newCredentialListModel(testIdentity(), nil)
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

func TestCredentialDetailTOTPSectionShown(t *testing.T) {
	m := newCredentialDetailModel(testCredential())
	view := m.View()

	if !strings.Contains(view, "totp") {
		t.Error("view should show TOTP section when secret present")
	}
}

func TestCredentialDetailTOTPSectionHiddenWhenNoSecret(t *testing.T) {
	m := newCredentialDetailModel(testCredentialNoTOTP())
	view := m.View()

	// totp section should not appear (only totp label line would show it)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "totp") {
			t.Error("view should not show totp section when no secret")
		}
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
	m := newCredentialFormModel(testIdentity(), nil)
	view := m.View()

	// title is now in root chrome; form shows identity header and field labels
	if !strings.Contains(view, "Jane Doe") {
		t.Error("should show identity name in add mode")
	}
	for _, label := range fieldLabels {
		if !strings.Contains(view, label) {
			t.Errorf("form should contain %q", label)
		}
	}
}

func TestCredentialFormEditView(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel(testIdentity(), &c)
	view := m.View()

	// title is now in root chrome; verify editing mode and field labels
	if !m.editing {
		t.Error("should be in editing mode")
	}
	for _, label := range fieldLabels {
		if !strings.Contains(view, label) {
			t.Errorf("form should contain %q", label)
		}
	}
}

func TestCredentialFormEditPreloads(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel(testIdentity(), &c)

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
	m := newCredentialFormModel(testIdentity(), nil)

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
	m := newCredentialFormModel(testIdentity(), nil)
	// label is empty
	m, _ = m.Update(enterKey())
	if m.flash != "label is required" {
		t.Errorf("flash = %q, want %q", m.flash, "label is required")
	}
}

func TestCredentialFormSubmitInvalidTOTP(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	m.inputs[fieldLabel].SetValue("Test")
	m.inputs[fieldTOTPSecret].SetValue("!!!invalid!!!")

	m, _ = m.Update(enterKey())
	if m.flash != "invalid totp secret (must be base32)" {
		t.Errorf("flash = %q, want totp error", m.flash)
	}
}

func TestCredentialFormSubmitValidAdd(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
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
	m := newCredentialFormModel(testIdentity(), &c)
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
	m := newCredentialFormModel(testIdentity(), nil)
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
	m := newCredentialFormModel(testIdentity(), nil)
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
	m := newCredentialFormModel(testIdentity(), &c)
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
	m := newCredentialFormModel(testIdentity(), nil)
	_, cmd := m.Update(specialKey(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestCredentialFormFlashClears(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	m.flash = "something"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

// credential form dual-mode tests

func TestCredentialFormIdentityHeader(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	view := m.View()

	if !strings.Contains(view, "Jane Doe") {
		t.Error("should show identity name in header")
	}
	if !strings.Contains(view, "jane@zburn.id") {
		t.Error("should show identity email in header")
	}
}

func TestCredentialFormIdentityHeaderHiddenOnEdit(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel(testIdentity(), &c)
	view := m.View()

	// the identity header is only for add mode
	if strings.Contains(view, "Jane Doe") && strings.Contains(view, "jane@zburn.id") {
		// check that it's not rendered as the identity header line
		// in edit mode, the header should not appear
		lines := strings.Split(view, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Jane Doe") && strings.Contains(line, "jane@zburn.id") {
				t.Error("edit mode should not show identity header")
			}
		}
	}
}

func TestCredentialFormUsernameCycle(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)

	// default: identity's email
	if m.inputs[fieldUsername].Value() != "jane@zburn.id" {
		t.Errorf("default username = %q, want %q", m.inputs[fieldUsername].Value(), "jane@zburn.id")
	}

	// focus on username field
	for m.focus != fieldUsername {
		m, _ = m.Update(tabKey())
	}

	// cycle through options: jane.doe, jdoe, janedoe, random handle, back to email
	m, _ = m.Update(spaceKey())
	if m.inputs[fieldUsername].Value() != "jane.doe" {
		t.Errorf("after 1st space = %q, want %q", m.inputs[fieldUsername].Value(), "jane.doe")
	}

	m, _ = m.Update(spaceKey())
	if m.inputs[fieldUsername].Value() != "jdoe" {
		t.Errorf("after 2nd space = %q, want %q", m.inputs[fieldUsername].Value(), "jdoe")
	}

	m, _ = m.Update(spaceKey())
	if m.inputs[fieldUsername].Value() != "janedoe" {
		t.Errorf("after 3rd space = %q, want %q", m.inputs[fieldUsername].Value(), "janedoe")
	}

	// 4th space: random handle (adjective + noun + 4 digits)
	m, _ = m.Update(spaceKey())
	handle := m.inputs[fieldUsername].Value()
	if len(handle) < 5 {
		t.Errorf("random handle too short: %q", handle)
	}
	// should not match any of the static options
	if handle == "jane@zburn.id" || handle == "jane.doe" || handle == "jdoe" || handle == "janedoe" {
		t.Errorf("handle should be random, got %q", handle)
	}

	// 5th space: wraps back to email
	m, _ = m.Update(spaceKey())
	if m.inputs[fieldUsername].Value() != "jane@zburn.id" {
		t.Errorf("after wrap = %q, want %q", m.inputs[fieldUsername].Value(), "jane@zburn.id")
	}
}

func TestCredentialFormPasswordGenerated(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)

	// password should be pre-filled with a generated password
	pw := m.inputs[fieldPassword].Value()
	if pw == "" {
		t.Error("password should be pre-filled on new form")
	}
	if len(pw) != 20 {
		t.Errorf("generated password length = %d, want 20", len(pw))
	}
}

func TestCredentialFormPasswordCycle(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	pw1 := m.inputs[fieldPassword].Value()

	// focus on password field
	for m.focus != fieldPassword {
		m, _ = m.Update(tabKey())
	}

	// space regenerates password
	m, _ = m.Update(spaceKey())
	pw2 := m.inputs[fieldPassword].Value()
	if pw2 == "" {
		t.Error("password should be regenerated")
	}
	if len(pw2) != 20 {
		t.Errorf("regenerated password length = %d, want 20", len(pw2))
	}
	// extremely unlikely to be the same
	if pw1 == pw2 {
		t.Error("regenerated password should differ from original")
	}
}

func TestCredentialFormEditModeNoGeneration(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel(testIdentity(), &c)

	// username should be the existing value, not generated
	if m.inputs[fieldUsername].Value() != "janedoe" {
		t.Errorf("username = %q, want %q", m.inputs[fieldUsername].Value(), "janedoe")
	}
	// password should be the existing value
	if m.inputs[fieldPassword].Value() != "s3cret!Pass" {
		t.Errorf("password = %q, want %q", m.inputs[fieldPassword].Value(), "s3cret!Pass")
	}

	// focus on username and press space -- should type a space, not cycle
	for m.focus != fieldUsername {
		m, _ = m.Update(tabKey())
	}
	// in edit mode, space should be handled by the text input (typing a space)
	m, _ = m.Update(spaceKey())
	// the value should now have a space appended
	if !strings.Contains(m.inputs[fieldUsername].Value(), " ") {
		t.Error("space in edit mode should type a literal space")
	}
}

func TestCredentialFormUsernameEditMode(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)

	// focus on username
	for m.focus != fieldUsername {
		m, _ = m.Update(tabKey())
	}

	// verify we start in cycle mode
	if m.usernameMode != modeCycle {
		t.Error("should start in cycle mode")
	}

	// type a character to switch to edit mode
	m, _ = m.Update(keyMsg('x'))
	if m.usernameMode != modeEdit {
		t.Error("typing should switch to edit mode")
	}
	// the input should contain the typed character
	if !strings.Contains(m.inputs[fieldUsername].Value(), "x") {
		t.Errorf("input should contain typed character, got %q", m.inputs[fieldUsername].Value())
	}
}

func TestCredentialFormUsernameEscCycle(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)

	// focus on username
	for m.focus != fieldUsername {
		m, _ = m.Update(tabKey())
	}

	// enter edit mode
	m, _ = m.Update(keyMsg('x'))
	if m.usernameMode != modeEdit {
		t.Fatal("should be in edit mode")
	}

	// esc returns to cycle mode with the generated value
	m, _ = m.Update(escKey())
	if m.usernameMode != modeCycle {
		t.Error("esc should return to cycle mode")
	}
	if m.inputs[fieldUsername].Value() != "jane@zburn.id" {
		t.Errorf("should restore generated value, got %q", m.inputs[fieldUsername].Value())
	}
}

func TestCredentialFormPasswordEscCycle(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	generatedPW := m.generatedPW

	// focus on password
	for m.focus != fieldPassword {
		m, _ = m.Update(tabKey())
	}

	// enter edit mode
	m, _ = m.Update(keyMsg('x'))
	if m.passwordMode != modeEdit {
		t.Fatal("should be in edit mode")
	}

	// esc returns to cycle mode with generated password
	m, _ = m.Update(escKey())
	if m.passwordMode != modeCycle {
		t.Error("esc should return to cycle mode")
	}
	if m.inputs[fieldPassword].Value() != generatedPW {
		t.Errorf("should restore generated password, got %q", m.inputs[fieldPassword].Value())
	}
}

func TestCredentialFormGeneratedIndicator(t *testing.T) {
	m := newCredentialFormModel(testIdentity(), nil)
	view := m.View()

	if !strings.Contains(view, "[generated]") {
		t.Error("should show [generated] indicator for cycle mode fields")
	}
}

func TestCredentialFormGeneratedIndicatorHiddenOnEdit(t *testing.T) {
	c := testCredential()
	m := newCredentialFormModel(testIdentity(), &c)
	view := m.View()

	if strings.Contains(view, "[generated]") {
		t.Error("should not show [generated] indicator in edit mode")
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
	if view.identity.ID != "abc12345" {
		t.Errorf("identity ID = %q, want %q", view.identity.ID, "abc12345")
	}
}

func TestDetailShowsCredentialSection(t *testing.T) {
	m := newDetailModel(testIdentity())
	view := m.View()

	// help is now in root chrome, but credential section info should still appear
	if !strings.Contains(view, "credentials") {
		t.Error("view should show credentials section")
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

	id := testIdentity()
	result, _ := m.Update(addCredentialMsg{identity: id})
	rm := result.(Model)
	if rm.active != viewCredentialForm {
		t.Errorf("active = %d, want viewCredentialForm", rm.active)
	}
	if rm.credentialForm.identity.ID != "abc12345" {
		t.Errorf("identity ID = %q, want %q", rm.credentialForm.identity.ID, "abc12345")
	}
	if rm.credentialForm.editing {
		t.Error("should not be in editing mode for add")
	}
}

func TestRootEditCredentialMsg(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewCredentialDetail
	m.detail = newDetailModel(testIdentity())

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
	m.credentialList = newCredentialListModel(testIdentity(), nil)

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

func spaceKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
}
