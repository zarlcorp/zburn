package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
	"github.com/zarlcorp/zburn/internal/gmail"
	"github.com/zarlcorp/zburn/internal/identity"
	"github.com/zarlcorp/zburn/internal/namecheap"
)

// config round-trip tests

func openTestStore(t *testing.T) *zstore.Store {
	t.Helper()
	fs := zfilesystem.NewMemFS()
	s, err := zstore.Open(fs, []byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNamecheapSettingsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		t.Fatal(err)
	}

	want := NamecheapSettings{
		Username:      "user1",
		APIKey:        "key1",
		CachedDomains: []string{"foo.com", "bar.io"},
	}

	if err := saveConfig(col, "namecheap", want); err != nil {
		t.Fatal(err)
	}

	got := loadConfig[NamecheapSettings](col, "namecheap")
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if len(got.CachedDomains) != len(want.CachedDomains) {
		t.Fatalf("CachedDomains length = %d, want %d", len(got.CachedDomains), len(want.CachedDomains))
	}
	for i := range want.CachedDomains {
		if got.CachedDomains[i] != want.CachedDomains[i] {
			t.Errorf("CachedDomains[%d] = %q, want %q", i, got.CachedDomains[i], want.CachedDomains[i])
		}
	}
}

func TestGmailSettingsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		t.Fatal(err)
	}

	want := GmailSettings{
		ClientID:     "cid",
		ClientSecret: "csecret",
		Token: &gmail.Token{
			AccessToken:  "at",
			RefreshToken: "rt",
			Expiry:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := saveConfig(col, "gmail", want); err != nil {
		t.Fatal(err)
	}

	got := loadConfig[GmailSettings](col, "gmail")
	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q", got.ClientID, want.ClientID)
	}
	if got.ClientSecret != want.ClientSecret {
		t.Errorf("ClientSecret = %q, want %q", got.ClientSecret, want.ClientSecret)
	}
	if got.Token == nil {
		t.Fatal("Token should not be nil")
	}
	if got.Token.AccessToken != want.Token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", got.Token.AccessToken, want.Token.AccessToken)
	}
	if got.Token.RefreshToken != want.Token.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", got.Token.RefreshToken, want.Token.RefreshToken)
	}
}

func TestTwilioSettingsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		t.Fatal(err)
	}

	want := TwilioSettings{
		AccountSID:         "sid1",
		AuthToken:          "token1",
		PreferredCountries: []string{"GB", "US"},
	}

	if err := saveConfig(col, "twilio", want); err != nil {
		t.Fatal(err)
	}

	got := loadConfig[TwilioSettings](col, "twilio")
	if got.AccountSID != want.AccountSID {
		t.Errorf("AccountSID = %q, want %q", got.AccountSID, want.AccountSID)
	}
	if got.AuthToken != want.AuthToken {
		t.Errorf("AuthToken = %q, want %q", got.AuthToken, want.AuthToken)
	}
	if len(got.PreferredCountries) != len(want.PreferredCountries) {
		t.Fatalf("PreferredCountries length = %d, want %d", len(got.PreferredCountries), len(want.PreferredCountries))
	}
}

func TestLoadConfigMissing(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		t.Fatal(err)
	}

	got := loadConfig[NamecheapSettings](col, "nonexistent")
	if got.Username != "" || got.APIKey != "" {
		t.Error("missing config should return zero value")
	}
}

func TestLoadConfigNilCollection(t *testing.T) {
	got := loadConfig[NamecheapSettings](nil, "namecheap")
	if got.Username != "" {
		t.Error("nil collection should return zero value")
	}
}

// feature gating tests

func TestNamecheapConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  NamecheapSettings
		want bool
	}{
		{"empty", NamecheapSettings{}, false},
		{"partial username", NamecheapSettings{Username: "u"}, false},
		{"partial key", NamecheapSettings{APIKey: "k"}, false},
		{"full", NamecheapSettings{Username: "u", APIKey: "k"}, true},
	}

	for _, tt := range tests {
		if got := tt.cfg.Configured(); got != tt.want {
			t.Errorf("%s: Configured() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestGmailConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  GmailSettings
		want bool
	}{
		{"empty", GmailSettings{}, false},
		{"no token", GmailSettings{ClientID: "id", ClientSecret: "s"}, false},
		{"token no refresh", GmailSettings{Token: &gmail.Token{AccessToken: "a"}}, false},
		{"token no email", GmailSettings{Token: &gmail.Token{RefreshToken: "r"}}, false},
		{"configured", GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}, true},
	}

	for _, tt := range tests {
		if got := tt.cfg.Configured(); got != tt.want {
			t.Errorf("%s: Configured() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTwilioConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  TwilioSettings
		want bool
	}{
		{"empty", TwilioSettings{}, false},
		{"partial", TwilioSettings{AccountSID: "s"}, false},
		{"full", TwilioSettings{AccountSID: "s", AuthToken: "t"}, true},
	}

	for _, tt := range tests {
		if got := tt.cfg.Configured(); got != tt.want {
			t.Errorf("%s: Configured() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// settings menu tests

func TestSettingsViewShowsItems(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	view := m.View()

	for _, item := range []string{"namecheap", "gmail", "twilio", "back"} {
		if !strings.Contains(view, item) {
			t.Errorf("settings should contain %q", item)
		}
	}
}

func TestSettingsViewShowsStatus(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	m := newSettingsModel(nc, GmailSettings{}, TwilioSettings{})
	view := m.View()

	if !strings.Contains(view, "configured") {
		t.Error("should show configured status for namecheap")
	}
	if !strings.Contains(view, "not configured") {
		t.Error("should show not configured for gmail/twilio")
	}
}

func TestSettingsNavigation(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})

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

func TestSettingsSelectNamecheap(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettingsNamecheap {
		t.Errorf("view = %d, want viewSettingsNamecheap", nav.view)
	}
}

func TestSettingsSelectGmail(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
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
	if nav.view != viewSettingsGmail {
		t.Errorf("view = %d, want viewSettingsGmail", nav.view)
	}
}

func TestSettingsSelectTwilio(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
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
	if nav.view != viewSettingsTwilio {
		t.Errorf("view = %d, want viewSettingsTwilio", nav.view)
	}
}

func TestSettingsSelectBack(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	m.cursor = 4 // back
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
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

func TestSettingsEscGoesBack(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
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

func TestSettingsQuit(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

// namecheap form tests

func TestNamecheapFormPopulatesFromConfig(t *testing.T) {
	cfg := NamecheapSettings{
		Username:      "myuser",
		APIKey:        "mykey",
		CachedDomains: []string{"a.com", "b.io"},
	}
	m := newNamecheapModel(cfg)

	if m.inputs[ncUsername].Value() != "myuser" {
		t.Errorf("username = %q, want %q", m.inputs[ncUsername].Value(), "myuser")
	}
	if m.inputs[ncAPIKey].Value() != "mykey" {
		t.Errorf("api key = %q, want %q", m.inputs[ncAPIKey].Value(), "mykey")
	}
}

func TestNamecheapFormShowsLabels(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	view := m.View()

	for _, l := range ncLabels {
		if !strings.Contains(view, l) {
			t.Errorf("view should contain label %q", l)
		}
	}
}

func TestNamecheapFormTabAdvances(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})

	if m.focus != 0 {
		t.Fatal("focus should start at 0")
	}

	m = m.nextField()
	if m.focus != 1 {
		t.Errorf("focus = %d, want 1", m.focus)
	}

	// wraps back to 0 (only 2 fields now)
	m = m.nextField()
	if m.focus != 0 {
		t.Errorf("focus = %d, want 0 (wrap)", m.focus)
	}
}

func TestNamecheapFormValidateAndSave(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	m.inputs[ncUsername].SetValue("u1")
	m.inputs[ncAPIKey].SetValue("k1")

	// inject a fake validator that returns domains
	m.validateFn = func(_ context.Context, cfg namecheap.Config) ([]string, error) {
		if cfg.Username != "u1" || cfg.APIKey != "k1" {
			t.Errorf("validate got cfg %+v", cfg)
		}
		return []string{"x.com", "y.io"}, nil
	}

	m, cmd := m.startValidate()
	if !m.saving {
		t.Error("should be in saving state")
	}
	if cmd == nil {
		t.Fatal("should produce command")
	}

	// execute the validation command
	msg := cmd()
	result, ok := msg.(ncValidateResultMsg)
	if !ok {
		t.Fatal("should produce ncValidateResultMsg")
	}

	// feed result back into model
	m, cmd = m.Update(result)
	if m.saving {
		t.Error("should not be saving after result")
	}
	if cmd == nil {
		t.Fatal("should produce save command")
	}
	if !strings.Contains(m.flash, "2 domains found") {
		t.Errorf("flash = %q, want contains '2 domains found'", m.flash)
	}

	// execute the save command
	saveMsg := cmd()
	save, ok := saveMsg.(saveNamecheapMsg)
	if !ok {
		t.Fatal("should emit saveNamecheapMsg")
	}

	if save.settings.Username != "u1" {
		t.Errorf("Username = %q, want %q", save.settings.Username, "u1")
	}
	if save.settings.APIKey != "k1" {
		t.Errorf("APIKey = %q, want %q", save.settings.APIKey, "k1")
	}
	if len(save.settings.CachedDomains) != 2 {
		t.Fatalf("CachedDomains len = %d, want 2", len(save.settings.CachedDomains))
	}
	if save.settings.CachedDomains[0] != "x.com" || save.settings.CachedDomains[1] != "y.io" {
		t.Errorf("CachedDomains = %v, want [x.com y.io]", save.settings.CachedDomains)
	}
}

func TestNamecheapFormValidateError(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	m.inputs[ncUsername].SetValue("u1")
	m.inputs[ncAPIKey].SetValue("k1")

	m.validateFn = func(_ context.Context, _ namecheap.Config) ([]string, error) {
		return nil, fmt.Errorf("invalid credentials")
	}

	m, cmd := m.startValidate()
	msg := cmd()
	result := msg.(ncValidateResultMsg)

	m, _ = m.Update(result)
	if m.saving {
		t.Error("should not be saving after error")
	}
	if !strings.Contains(m.flash, "invalid credentials") {
		t.Errorf("flash = %q, should contain error", m.flash)
	}
}

func TestNamecheapFormEmptyFieldsReject(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	// leave fields empty
	m, _ = m.startValidate()
	if m.saving {
		t.Error("should not be saving with empty fields")
	}
	if !strings.Contains(m.flash, "required") {
		t.Errorf("flash = %q, should mention required", m.flash)
	}
}

func TestNamecheapFormSavingBlocksKeys(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	m.saving = true
	m, cmd := m.Update(keyMsg('q'))
	if cmd != nil {
		t.Error("keys should be blocked during saving")
	}
}

func TestNamecheapEscGoesBack(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettings {
		t.Errorf("view = %d, want viewSettings", nav.view)
	}
}

func TestNamecheapFlashClears(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	m.flash = "saved"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

// gmail form tests

func TestGmailFormPopulatesFromConfig(t *testing.T) {
	cfg := GmailSettings{
		ClientID:     "myid",
		ClientSecret: "mysecret",
		Token:        &gmail.Token{RefreshToken: "rt"},
	}
	m := newGmailModel(cfg)

	if m.inputs[gmailClientID].Value() != "myid" {
		t.Errorf("client id = %q, want %q", m.inputs[gmailClientID].Value(), "myid")
	}
	if m.inputs[gmailClientSecret].Value() != "mysecret" {
		t.Errorf("client secret = %q, want %q", m.inputs[gmailClientSecret].Value(), "mysecret")
	}
}

func TestGmailFormShowsConnectedStatus(t *testing.T) {
	cfg := GmailSettings{Token: &gmail.Token{RefreshToken: "rt"}, Email: "user@gmail.com"}
	m := newGmailModel(cfg)
	view := m.View()

	if !strings.Contains(view, "connected as user@gmail.com") {
		t.Error("should show connected status with email")
	}
}

func TestGmailFormShowsNotConnectedStatus(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	view := m.View()

	if !strings.Contains(view, "not connected") {
		t.Error("should show not connected status")
	}
}

func TestGmailOAuthStartsWaiting(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	m.inputs[gmailClientID].SetValue("id")
	m.inputs[gmailClientSecret].SetValue("secret")

	// inject a test auth function that returns immediately
	m.authenticateFn = func(_ context.Context, _ gmail.OAuthConfig) (*gmail.Token, error) {
		return &gmail.Token{
			AccessToken:  "at",
			RefreshToken: "rt",
			Expiry:       time.Now().Add(time.Hour),
		}, nil
	}
	m.profileFn = func(_ context.Context, _ string) (string, error) {
		return "user@gmail.com", nil
	}

	m, cmd := m.startOAuth()
	if m.action != gmailActionWaiting {
		t.Error("should be in waiting state")
	}
	if cmd == nil {
		t.Fatal("should produce command")
	}

	// execute the oauth cmd
	msg := cmd()
	result, ok := msg.(gmailOAuthResultMsg)
	if !ok {
		t.Fatal("should produce gmailOAuthResultMsg")
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if result.token.AccessToken != "at" {
		t.Errorf("token = %q, want %q", result.token.AccessToken, "at")
	}
	if result.email != "user@gmail.com" {
		t.Errorf("email = %q, want user@gmail.com", result.email)
	}
}

func TestGmailOAuthError(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	m.inputs[gmailClientID].SetValue("id")
	m.inputs[gmailClientSecret].SetValue("secret")

	m.authenticateFn = func(_ context.Context, _ gmail.OAuthConfig) (*gmail.Token, error) {
		return nil, fmt.Errorf("auth failed")
	}

	m, cmd := m.startOAuth()
	msg := cmd()
	result := msg.(gmailOAuthResultMsg)

	// feed the error back
	m, _ = m.Update(result)
	if m.action != gmailActionForm {
		t.Error("should return to form state on error")
	}
	if !strings.Contains(m.flash, "auth failed") {
		t.Errorf("flash = %q, should contain error", m.flash)
	}
}

func TestGmailOAuthSuccess(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	m.inputs[gmailClientID].SetValue("id")
	m.inputs[gmailClientSecret].SetValue("secret")

	tok := &gmail.Token{AccessToken: "at", RefreshToken: "rt"}
	m, cmd := m.Update(gmailOAuthResultMsg{token: tok})
	if cmd == nil {
		t.Fatal("should produce save command")
	}
	msg := cmd()
	save, ok := msg.(saveGmailMsg)
	if !ok {
		t.Fatal("should emit saveGmailMsg")
	}
	if save.settings.Token.AccessToken != "at" {
		t.Errorf("token = %q, want %q", save.settings.Token.AccessToken, "at")
	}
}

func TestGmailEmptyFieldsReject(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	// leave fields empty
	m, _ = m.startOAuth()
	if m.action != gmailActionForm {
		t.Error("should remain in form state")
	}
	if !strings.Contains(m.flash, "required") {
		t.Errorf("flash = %q, should mention required", m.flash)
	}
}

func TestGmailDisconnect(t *testing.T) {
	cfg := GmailSettings{Token: &gmail.Token{RefreshToken: "rt"}, Email: "user@gmail.com"}
	m := newGmailModel(cfg)
	_, cmd := m.Update(specialKey(0)) // dummy — test handleKey directly

	// simulate ctrl+d via handleKey
	ctrlD := keyMsg('d')                       // this won't match ctrl+d
	_ = ctrlD                                  // we need to test the disconnect path differently
	_, cmd = m.handleKey(ctrlDKey())
	if cmd == nil {
		t.Fatal("ctrl+d should produce command when connected")
	}
	msg := cmd()
	if _, ok := msg.(disconnectGmailMsg); !ok {
		t.Fatal("should emit disconnectGmailMsg")
	}
}

func TestGmailDisconnectNotConnected(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	_, cmd := m.handleKey(ctrlDKey())
	if cmd != nil {
		t.Error("ctrl+d should do nothing when not connected")
	}
}

func TestGmailEscGoesBack(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettings {
		t.Errorf("view = %d, want viewSettings", nav.view)
	}
}

func TestGmailWaitingStateEscCancels(t *testing.T) {
	m := newGmailModel(GmailSettings{})
	m.action = gmailActionWaiting
	view := m.View()
	if !strings.Contains(view, "waiting for browser") {
		t.Error("should show waiting message")
	}

	m, _ = m.Update(escKey())
	if m.action != gmailActionForm {
		t.Error("esc should cancel waiting")
	}
}

// twilio form tests

func TestTwilioFormPopulatesFromConfig(t *testing.T) {
	cfg := TwilioSettings{
		AccountSID:         "sid1",
		AuthToken:          "tok1",
		PreferredCountries: []string{"GB"},
	}
	m := newTwilioModel(cfg)

	if m.inputs[twAccountSID].Value() != "sid1" {
		t.Errorf("account sid = %q, want %q", m.inputs[twAccountSID].Value(), "sid1")
	}
	if m.inputs[twAuthToken].Value() != "tok1" {
		t.Errorf("auth token = %q, want %q", m.inputs[twAuthToken].Value(), "tok1")
	}
	if !m.countries["GB"] {
		t.Error("GB should be selected")
	}
	if m.countries["US"] {
		t.Error("US should not be selected when only GB specified")
	}
}

func TestTwilioDefaultCountries(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})
	if !m.countries["GB"] || !m.countries["US"] {
		t.Error("both countries should default to selected")
	}
}

func TestTwilioCountryToggle(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})
	// navigate to first country option
	m.focus = int(twFieldCount) // first country

	m, _ = m.Update(enterKey())
	// GB should be toggled off (was true by default)
	if m.countries["GB"] {
		t.Error("GB should be toggled off")
	}

	m, _ = m.Update(enterKey())
	// GB should be toggled back on
	if !m.countries["GB"] {
		t.Error("GB should be toggled back on")
	}
}

func TestTwilioFormSave(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})
	m.inputs[twAccountSID].SetValue("mysid")
	m.inputs[twAuthToken].SetValue("mytoken")

	cmd := m.save()
	if cmd == nil {
		t.Fatal("save should produce command")
	}
	msg := cmd()
	save, ok := msg.(saveTwilioMsg)
	if !ok {
		t.Fatal("should emit saveTwilioMsg")
	}

	if save.settings.AccountSID != "mysid" {
		t.Errorf("AccountSID = %q, want %q", save.settings.AccountSID, "mysid")
	}
	if save.settings.AuthToken != "mytoken" {
		t.Errorf("AuthToken = %q, want %q", save.settings.AuthToken, "mytoken")
	}
	if len(save.settings.PreferredCountries) == 0 {
		t.Error("should have preferred countries")
	}
}

func TestTwilioEscGoesBack(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettings {
		t.Errorf("view = %d, want viewSettings", nav.view)
	}
}

func TestTwilioFlashClears(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})
	m.flash = "saved"
	m, _ = m.Update(flashMsg{})
	if m.flash != "" {
		t.Errorf("flash should be empty, got %q", m.flash)
	}
}

func TestTwilioNavigation(t *testing.T) {
	m := newTwilioModel(TwilioSettings{})

	// start at first text input
	if m.focus != 0 {
		t.Fatal("focus should start at 0")
	}

	// tab through all fields
	total := m.totalFields()
	for i := 1; i < total; i++ {
		m = m.nextField()
		if m.focus != i {
			t.Errorf("focus = %d, want %d", m.focus, i)
		}
	}

	// wraps around
	m = m.nextField()
	if m.focus != 0 {
		t.Errorf("focus = %d, want 0 (wrap)", m.focus)
	}
}

// menu settings option tests

func TestMenuSelectSettings(t *testing.T) {
	m := newMenuModel("1.0")
	m.cursor = 2 // settings
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettings {
		t.Errorf("view = %d, want viewSettings", nav.view)
	}
}

func TestMenuShowsSettingsItem(t *testing.T) {
	m := newMenuModel("1.0")
	view := m.View()
	if !strings.Contains(view, "settings") {
		t.Error("menu should contain settings item")
	}
}

// root model settings navigation tests

func TestRootNavigateToSettings(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewMenu

	result, _ := m.Update(navigateMsg{view: viewSettings})
	rm := result.(Model)
	if rm.active != viewSettings {
		t.Errorf("active = %d, want viewSettings", rm.active)
	}
}

func TestRootNavigateToSettingsNamecheap(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings

	result, _ := m.Update(navigateMsg{view: viewSettingsNamecheap})
	rm := result.(Model)
	if rm.active != viewSettingsNamecheap {
		t.Errorf("active = %d, want viewSettingsNamecheap", rm.active)
	}
}

func TestRootNavigateToSettingsGmail(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings

	result, _ := m.Update(navigateMsg{view: viewSettingsGmail})
	rm := result.(Model)
	if rm.active != viewSettingsGmail {
		t.Errorf("active = %d, want viewSettingsGmail", rm.active)
	}
}

func TestRootNavigateToSettingsTwilio(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings

	result, _ := m.Update(navigateMsg{view: viewSettingsTwilio})
	rm := result.(Model)
	if rm.active != viewSettingsTwilio {
		t.Errorf("active = %d, want viewSettingsTwilio", rm.active)
	}
}

func TestRootQuitFromSettings(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings
	m.settings = newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})

	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit from settings")
	}
}

// conversion tests

func TestNamecheapConfigConversion(t *testing.T) {
	s := NamecheapSettings{Username: "u", APIKey: "k"}
	cfg := s.NamecheapConfig()
	if cfg.Username != "u" || cfg.APIKey != "k" {
		t.Error("conversion should preserve all fields")
	}
}

func TestGmailOAuthConfigConversion(t *testing.T) {
	s := GmailSettings{ClientID: "id", ClientSecret: "sec"}
	cfg := s.OAuthConfig()
	if cfg.ClientID != "id" || cfg.ClientSecret != "sec" {
		t.Error("conversion should preserve all fields")
	}
}

func TestTwilioConfigConversion(t *testing.T) {
	s := TwilioSettings{AccountSID: "sid", AuthToken: "tok"}
	cfg := s.TwilioConfig()
	if cfg.AccountSID != "sid" || cfg.AuthToken != "tok" {
		t.Error("conversion should preserve all fields")
	}
}

// configEnvelope json tests

func TestConfigEnvelopeJSON(t *testing.T) {
	want := NamecheapSettings{Username: "u", APIKey: "k"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	env := configEnvelope{Data: data}
	envJSON, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}

	var env2 configEnvelope
	if err := json.Unmarshal(envJSON, &env2); err != nil {
		t.Fatal(err)
	}

	var got NamecheapSettings
	if err := json.Unmarshal(env2.Data, &got); err != nil {
		t.Fatal(err)
	}

	if got.Username != want.Username || got.APIKey != want.APIKey {
		t.Errorf("round-trip failed: got %+v, want %+v", got, want)
	}
}

// settings menu forwarding item tests

func TestSettingsSelectForwarding(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	m.cursor = 3 // forwarding
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewForwarding {
		t.Errorf("view = %d, want viewForwarding", nav.view)
	}
}

func TestSettingsViewShowsForwarding(t *testing.T) {
	m := newSettingsModel(NamecheapSettings{}, GmailSettings{}, TwilioSettings{})
	view := m.View()
	if !strings.Contains(view, "forwarding") {
		t.Error("settings should contain forwarding item")
	}
}

// forwarding view tests

func TestForwardingWarningBothUnconfigured(t *testing.T) {
	m := newForwardingModel(NamecheapSettings{}, GmailSettings{})
	if m.warning != "configure namecheap and gmail to enable forwarding" {
		t.Errorf("warning = %q, want both-unconfigured message", m.warning)
	}
	view := m.View()
	if !strings.Contains(view, "configure namecheap and gmail") {
		t.Error("view should show both-unconfigured warning")
	}
}

func TestForwardingWarningGmailMissing(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	m := newForwardingModel(nc, GmailSettings{})
	if m.warning != "gmail not connected — forwarding inactive" {
		t.Errorf("warning = %q, want gmail-missing message", m.warning)
	}
}

func TestForwardingWarningNamecheapMissing(t *testing.T) {
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(NamecheapSettings{}, gm)
	if m.warning != "namecheap not connected — no domains" {
		t.Errorf("warning = %q, want namecheap-missing message", m.warning)
	}
}

func TestForwardingNoWarningWhenBothConfigured(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	if m.warning != "" {
		t.Errorf("warning = %q, want empty when both configured", m.warning)
	}
}

func TestForwardingLoadingState(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	m.loading = true
	view := m.View()
	if !strings.Contains(view, "loading...") {
		t.Error("should show loading state")
	}
}

func TestForwardingStatusMsgPopulates(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	m.loading = true

	statuses := []domainForwardingStatus{
		{domain: "alpha.com", rules: []namecheap.ForwardingRule{{Mailbox: "*", ForwardTo: "u@gmail.com"}}},
		{domain: "zarlcorp.com", excluded: true},
	}
	m, _ = m.Update(forwardingStatusMsg{statuses: statuses})
	if m.loading {
		t.Error("should not be loading after status msg")
	}
	if len(m.statuses) != 2 {
		t.Fatalf("statuses = %d, want 2", len(m.statuses))
	}
}

func TestForwardingViewShowsStatuses(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	m.statuses = []domainForwardingStatus{
		{domain: "alpha.com", rules: []namecheap.ForwardingRule{{Mailbox: "*", ForwardTo: "u@gmail.com"}}},
		{domain: "bravo.io", rules: nil},
		{domain: "zarlcorp.com", excluded: true},
		{domain: "fail.com", err: fmt.Errorf("api error")},
	}
	view := m.View()

	if !strings.Contains(view, "alpha.com") {
		t.Error("should show alpha.com")
	}
	if !strings.Contains(view, "u@gmail.com") {
		t.Error("should show forwarding target")
	}
	if !strings.Contains(view, "excluded") {
		t.Error("should show excluded for org domain")
	}
	if !strings.Contains(view, "not configured") {
		t.Error("should show not configured for domain without rules")
	}
	if !strings.Contains(view, "error") {
		t.Error("should show error for failed domain")
	}
}

func TestForwardingEscGoesBack(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should produce command")
	}
	msg := cmd()
	nav, ok := msg.(navigateMsg)
	if !ok {
		t.Fatal("should emit navigateMsg")
	}
	if nav.view != viewSettings {
		t.Errorf("view = %d, want viewSettings", nav.view)
	}
}

func TestForwardingQuit(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	_, cmd := m.Update(keyMsg('q'))
	if cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestForwardingNoDomains(t *testing.T) {
	nc := NamecheapSettings{Username: "u", APIKey: "k"}
	gm := GmailSettings{Token: &gmail.Token{RefreshToken: "r"}, Email: "u@gmail.com"}
	m := newForwardingModel(nc, gm)
	view := m.View()
	if !strings.Contains(view, "no domains") {
		t.Error("should show no domains when statuses empty")
	}
}

func TestFetchForwardingStatus(t *testing.T) {
	getter := &fakeForwardingGetter{
		rules: map[string][]namecheap.ForwardingRule{
			"alpha.com": {{Mailbox: "*", ForwardTo: "u@gmail.com"}},
			"bravo.io":  {},
		},
	}
	domains := []string{"alpha.com", "bravo.io", "zarlcorp.com"}
	msg := fetchForwardingStatus(context.Background(), getter, domains)

	if len(msg.statuses) != 3 {
		t.Fatalf("statuses = %d, want 3", len(msg.statuses))
	}

	// alpha.com has catch-all
	if catchAllTarget(msg.statuses[0].rules) != "u@gmail.com" {
		t.Errorf("alpha.com target = %q, want u@gmail.com", catchAllTarget(msg.statuses[0].rules))
	}

	// bravo.io has no rules
	if catchAllTarget(msg.statuses[1].rules) != "" {
		t.Error("bravo.io should have no catch-all target")
	}

	// zarlcorp.com is excluded
	if !msg.statuses[2].excluded {
		t.Error("zarlcorp.com should be excluded")
	}
}

func TestFetchForwardingStatusError(t *testing.T) {
	getter := &fakeForwardingGetter{
		err: fmt.Errorf("api error"),
	}
	domains := []string{"fail.com"}
	msg := fetchForwardingStatus(context.Background(), getter, domains)

	if msg.statuses[0].err == nil {
		t.Error("should propagate error")
	}
}

func TestCatchAllTarget(t *testing.T) {
	tests := []struct {
		name  string
		rules []namecheap.ForwardingRule
		want  string
	}{
		{"empty", nil, ""},
		{"no wildcard", []namecheap.ForwardingRule{{Mailbox: "info", ForwardTo: "a@b.com"}}, ""},
		{"wildcard", []namecheap.ForwardingRule{{Mailbox: "*", ForwardTo: "u@gmail.com"}}, "u@gmail.com"},
		{"mixed", []namecheap.ForwardingRule{
			{Mailbox: "info", ForwardTo: "a@b.com"},
			{Mailbox: "*", ForwardTo: "u@gmail.com"},
		}, "u@gmail.com"},
	}

	for _, tt := range tests {
		got := catchAllTarget(tt.rules)
		if got != tt.want {
			t.Errorf("%s: catchAllTarget = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// root model forwarding navigation tests

func TestRootNavigateToForwarding(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings

	result, _ := m.Update(navigateMsg{view: viewForwarding})
	rm := result.(Model)
	if rm.active != viewForwarding {
		t.Errorf("active = %d, want viewForwarding", rm.active)
	}
}

func TestRootNavigateToForwardingWithWarning(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings
	// no configs set

	result, _ := m.Update(navigateMsg{view: viewForwarding})
	rm := result.(Model)
	if rm.forwarding.warning == "" {
		t.Error("should have warning when no configs")
	}
}

func TestRootNavigateToForwardingFetchesWhenConfigured(t *testing.T) {
	m := New("1.0", t.TempDir(), identity.New(), false)
	m.active = viewSettings
	m.ncConfig = NamecheapSettings{
		Username:      "u",
		APIKey:        "k",
		CachedDomains: []string{"a.com"},
	}
	m.gmConfig = GmailSettings{
		Token: &gmail.Token{RefreshToken: "r"},
		Email: "u@gmail.com",
	}

	result, cmd := m.Update(navigateMsg{view: viewForwarding})
	rm := result.(Model)
	if !rm.forwarding.loading {
		t.Error("should be in loading state when both configured")
	}
	if cmd == nil {
		t.Error("should produce fetch command")
	}
}

// fakeForwardingGetter is a test double for the forwardingGetter interface.
type fakeForwardingGetter struct {
	rules map[string][]namecheap.ForwardingRule
	err   error
}

func (f *fakeForwardingGetter) GetForwarding(_ context.Context, domain string) ([]namecheap.ForwardingRule, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rules[domain], nil
}

// helper for ctrl+d key
func ctrlDKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlD}
}
