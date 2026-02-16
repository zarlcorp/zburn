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
		APIUser:  "user1",
		APIKey:   "key1",
		Username: "name1",
		ClientIP: "1.2.3.4",
		Domains:  []string{"foo.com", "bar.io"},
	}

	if err := saveConfig(col, "namecheap", want); err != nil {
		t.Fatal(err)
	}

	got := loadConfig[NamecheapSettings](col, "namecheap")
	if got.APIUser != want.APIUser {
		t.Errorf("APIUser = %q, want %q", got.APIUser, want.APIUser)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.ClientIP != want.ClientIP {
		t.Errorf("ClientIP = %q, want %q", got.ClientIP, want.ClientIP)
	}
	if len(got.Domains) != len(want.Domains) {
		t.Fatalf("Domains length = %d, want %d", len(got.Domains), len(want.Domains))
	}
	for i := range want.Domains {
		if got.Domains[i] != want.Domains[i] {
			t.Errorf("Domains[%d] = %q, want %q", i, got.Domains[i], want.Domains[i])
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
	if got.APIUser != "" || got.APIKey != "" {
		t.Error("missing config should return zero value")
	}
}

func TestLoadConfigNilCollection(t *testing.T) {
	got := loadConfig[NamecheapSettings](nil, "namecheap")
	if got.APIUser != "" {
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
		{"partial", NamecheapSettings{APIUser: "u"}, false},
		{"full", NamecheapSettings{APIUser: "u", APIKey: "k"}, true},
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
		{"configured", GmailSettings{Token: &gmail.Token{RefreshToken: "r"}}, true},
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
	nc := NamecheapSettings{APIUser: "u", APIKey: "k"}
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
	m.cursor = 3 // back
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
		APIUser:  "myuser",
		APIKey:   "mykey",
		Username: "myname",
		ClientIP: "1.2.3.4",
		Domains:  []string{"a.com", "b.io"},
	}
	m := newNamecheapModel(cfg)

	if m.inputs[ncAPIUser].Value() != "myuser" {
		t.Errorf("api user = %q, want %q", m.inputs[ncAPIUser].Value(), "myuser")
	}
	if m.inputs[ncAPIKey].Value() != "mykey" {
		t.Errorf("api key = %q, want %q", m.inputs[ncAPIKey].Value(), "mykey")
	}
	if m.inputs[ncUsername].Value() != "myname" {
		t.Errorf("username = %q, want %q", m.inputs[ncUsername].Value(), "myname")
	}
	if m.inputs[ncClientIP].Value() != "1.2.3.4" {
		t.Errorf("client ip = %q, want %q", m.inputs[ncClientIP].Value(), "1.2.3.4")
	}
	if !strings.Contains(m.inputs[ncDomains].Value(), "a.com") {
		t.Error("domains should contain a.com")
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

	m = m.nextField()
	if m.focus != 2 {
		t.Errorf("focus = %d, want 2", m.focus)
	}
}

func TestNamecheapFormSave(t *testing.T) {
	m := newNamecheapModel(NamecheapSettings{})
	m.inputs[ncAPIUser].SetValue("u1")
	m.inputs[ncAPIKey].SetValue("k1")
	m.inputs[ncUsername].SetValue("n1")
	m.inputs[ncClientIP].SetValue("5.6.7.8")
	m.inputs[ncDomains].SetValue("x.com, y.io")

	cmd := m.save()
	if cmd == nil {
		t.Fatal("save should produce command")
	}
	msg := cmd()
	save, ok := msg.(saveNamecheapMsg)
	if !ok {
		t.Fatal("should emit saveNamecheapMsg")
	}

	if save.settings.APIUser != "u1" {
		t.Errorf("APIUser = %q, want %q", save.settings.APIUser, "u1")
	}
	if save.settings.APIKey != "k1" {
		t.Errorf("APIKey = %q, want %q", save.settings.APIKey, "k1")
	}
	if len(save.settings.Domains) != 2 {
		t.Fatalf("Domains len = %d, want 2", len(save.settings.Domains))
	}
	if save.settings.Domains[0] != "x.com" || save.settings.Domains[1] != "y.io" {
		t.Errorf("Domains = %v, want [x.com y.io]", save.settings.Domains)
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
	cfg := GmailSettings{Token: &gmail.Token{RefreshToken: "rt"}}
	m := newGmailModel(cfg)
	view := m.View()

	if !strings.Contains(view, "connected") {
		t.Error("should show connected status")
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
	cfg := GmailSettings{Token: &gmail.Token{RefreshToken: "rt"}}
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
	m.cursor = 3 // settings
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
	s := NamecheapSettings{APIUser: "u", APIKey: "k", Username: "n", ClientIP: "1.2.3.4"}
	cfg := s.NamecheapConfig()
	if cfg.APIUser != "u" || cfg.APIKey != "k" || cfg.Username != "n" || cfg.ClientIP != "1.2.3.4" {
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

// parseDomains tests

func TestParseDomains(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a.com", []string{"a.com"}},
		{"a.com, b.io", []string{"a.com", "b.io"}},
		{"a.com,b.io,c.org", []string{"a.com", "b.io", "c.org"}},
		{" a.com , b.io ", []string{"a.com", "b.io"}},
		{"a.com\nb.io", []string{"a.com", "b.io"}},
	}

	for _, tt := range tests {
		got := parseDomains(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseDomains(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("parseDomains(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// configEnvelope json tests

func TestConfigEnvelopeJSON(t *testing.T) {
	want := NamecheapSettings{APIUser: "u", APIKey: "k"}
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

	if got.APIUser != want.APIUser || got.APIKey != want.APIKey {
		t.Errorf("round-trip failed: got %+v, want %+v", got, want)
	}
}

// helper for ctrl+d key
func ctrlDKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlD}
}
