package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
	"github.com/zarlcorp/zburn/internal/burn"
	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/gmail"
	"github.com/zarlcorp/zburn/internal/identity"
)

// openIntegrationStore opens a real zstore backed by OSFileSystem in a temp dir.
func openIntegrationStore(t *testing.T, password string) *zstore.Store {
	t.Helper()
	fs := zfilesystem.NewOSFileSystem(t.TempDir())
	s, err := zstore.Open(fs, []byte(password))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// setupModel creates a root Model with a real zstore, bypassing the password flow.
// Returns the model with open store and initialized collections.
func setupModel(t *testing.T) Model {
	t.Helper()
	s := openIntegrationStore(t, "testpass")

	idCol, err := zstore.NewCollection[identity.Identity](s, "identities")
	if err != nil {
		t.Fatal(err)
	}

	credCol, err := zstore.NewCollection[credential.Credential](s, "credentials")
	if err != nil {
		t.Fatal(err)
	}

	cfgCol, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		t.Fatal(err)
	}

	m := New("1.0", t.TempDir(), identity.New(), false)
	m.store = s
	m.identities = idCol
	m.credentials = credCol
	m.configs = cfgCol
	m.active = viewMenu
	return m
}

// saveIdentity puts an identity into the model's store and returns the model.
func saveIdentity(t *testing.T, m Model, id identity.Identity) Model {
	t.Helper()
	if err := m.identities.Put(id.ID, id); err != nil {
		t.Fatal(err)
	}
	return m
}

// saveCredential puts a credential into the model's store and returns the model.
func saveCredential(t *testing.T, m Model, c credential.Credential) Model {
	t.Helper()
	if err := m.credentials.Put(c.ID, c); err != nil {
		t.Fatal(err)
	}
	return m
}

// processMsg sends a message through the model and returns the updated model.
func processMsg(t *testing.T, m Model, msg interface{}) Model {
	t.Helper()
	result, _ := m.Update(msg)
	return result.(Model)
}

// credential lifecycle tests

func TestIntegrationCredentialLifecycle(t *testing.T) {
	m := setupModel(t)
	id := testIdentity()
	m = saveIdentity(t, m, id)

	// navigate to identity detail
	m = processMsg(t, m, viewIdentityMsg{identity: id})
	if m.active != viewDetail {
		t.Fatalf("active = %d, want viewDetail", m.active)
	}
	if m.detail.credentialCount != 0 {
		t.Fatalf("credential count = %d, want 0", m.detail.credentialCount)
	}

	// add a credential via the message flow
	now := time.Now()
	cred := credential.Credential{
		ID:         "cred-int-001",
		IdentityID: id.ID,
		Label:      "GitHub",
		URL:        "https://github.com",
		Username:   "janedoe",
		Password:   "pass1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m = processMsg(t, m, saveCredentialMsg{credential: cred})
	if m.active != viewCredentialDetail {
		t.Fatalf("active = %d, want viewCredentialDetail after save", m.active)
	}

	// go back and check the identity detail shows credential count = 1
	m = processMsg(t, m, viewIdentityMsg{identity: id})
	if m.detail.credentialCount != 1 {
		t.Errorf("credential count = %d, want 1", m.detail.credentialCount)
	}

	// add a second credential
	cred2 := credential.Credential{
		ID:         "cred-int-002",
		IdentityID: id.ID,
		Label:      "Netflix",
		URL:        "https://netflix.com",
		Username:   "jane",
		Password:   "pass2",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m = processMsg(t, m, saveCredentialMsg{credential: cred2})
	if m.active != viewCredentialDetail {
		t.Fatalf("active = %d, want viewCredentialDetail", m.active)
	}

	// check count is now 2
	m = processMsg(t, m, viewIdentityMsg{identity: id})
	if m.detail.credentialCount != 2 {
		t.Errorf("credential count = %d, want 2", m.detail.credentialCount)
	}
}

func TestIntegrationCredentialEditPersists(t *testing.T) {
	m := setupModel(t)
	id := testIdentity()
	m = saveIdentity(t, m, id)

	now := time.Now()
	cred := credential.Credential{
		ID:         "cred-edit-001",
		IdentityID: id.ID,
		Label:      "GitHub",
		Username:   "janedoe",
		Password:   "pass1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m = processMsg(t, m, saveCredentialMsg{credential: cred})

	// edit: change the label and password
	edited := cred
	edited.Label = "GitHub Enterprise"
	edited.Password = "newpass!"
	edited.UpdatedAt = time.Now()
	m = processMsg(t, m, saveCredentialMsg{credential: edited})

	// verify by loading credential list
	result, _ := m.Update(viewCredentialsMsg{identityID: id.ID})
	rm := result.(Model)
	if rm.active != viewCredentialList {
		t.Fatalf("active = %d, want viewCredentialList", rm.active)
	}

	// should still have 1 credential (edit, not add)
	if len(rm.credentialList.credentials) != 1 {
		t.Fatalf("credentials = %d, want 1", len(rm.credentialList.credentials))
	}
	if rm.credentialList.credentials[0].Label != "GitHub Enterprise" {
		t.Errorf("label = %q, want %q", rm.credentialList.credentials[0].Label, "GitHub Enterprise")
	}
	if rm.credentialList.credentials[0].Password != "newpass!" {
		t.Errorf("password = %q, want %q", rm.credentialList.credentials[0].Password, "newpass!")
	}
}

func TestIntegrationCredentialDeleteOthersRemain(t *testing.T) {
	m := setupModel(t)
	id := testIdentity()
	m = saveIdentity(t, m, id)

	now := time.Now()
	for i := range 3 {
		c := credential.Credential{
			ID:         fmt.Sprintf("cred-del-%03d", i),
			IdentityID: id.ID,
			Label:      fmt.Sprintf("Account %d", i),
			Username:   fmt.Sprintf("user%d", i),
			Password:   "pass",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		m = saveCredential(t, m, c)
	}

	// delete the middle credential
	m = processMsg(t, m, deleteCredentialMsg{id: "cred-del-001"})

	// load credential list — should have 2 remaining
	result, _ := m.Update(viewCredentialsMsg{identityID: id.ID})
	rm := result.(Model)
	if len(rm.credentialList.credentials) != 2 {
		t.Fatalf("credentials = %d, want 2", len(rm.credentialList.credentials))
	}

	// verify which ones remain
	labels := map[string]bool{}
	for _, c := range rm.credentialList.credentials {
		labels[c.Label] = true
	}
	if labels["Account 1"] {
		t.Error("Account 1 should have been deleted")
	}
	if !labels["Account 0"] || !labels["Account 2"] {
		t.Error("Account 0 and Account 2 should remain")
	}
}

func TestIntegrationCredentialIsolation(t *testing.T) {
	m := setupModel(t)

	idA := testIdentity()
	idA.ID = "id-A"
	idB := testIdentity()
	idB.ID = "id-B"
	idB.Email = "bob@zburn.id"

	m = saveIdentity(t, m, idA)
	m = saveIdentity(t, m, idB)

	now := time.Now()
	// save 2 credentials for identity A
	for i := range 2 {
		c := credential.Credential{
			ID:         fmt.Sprintf("cred-A-%d", i),
			IdentityID: "id-A",
			Label:      fmt.Sprintf("A-Account %d", i),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		m = saveCredential(t, m, c)
	}

	// save 1 credential for identity B
	m = saveCredential(t, m, credential.Credential{
		ID:         "cred-B-0",
		IdentityID: "id-B",
		Label:      "B-Account",
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// load credentials for identity A — should see 2
	result, _ := m.Update(viewCredentialsMsg{identityID: "id-A"})
	rm := result.(Model)
	if len(rm.credentialList.credentials) != 2 {
		t.Errorf("identity A credentials = %d, want 2", len(rm.credentialList.credentials))
	}
	for _, c := range rm.credentialList.credentials {
		if c.IdentityID != "id-A" {
			t.Errorf("credential %s belongs to %s, want id-A", c.ID, c.IdentityID)
		}
	}

	// load credentials for identity B — should see 1
	result, _ = rm.Update(viewCredentialsMsg{identityID: "id-B"})
	rm = result.(Model)
	if len(rm.credentialList.credentials) != 1 {
		t.Errorf("identity B credentials = %d, want 1", len(rm.credentialList.credentials))
	}
	if rm.credentialList.credentials[0].IdentityID != "id-B" {
		t.Errorf("credential belongs to %s, want id-B", rm.credentialList.credentials[0].IdentityID)
	}
}

// settings persistence tests

func TestIntegrationNamecheapSettingsPersist(t *testing.T) {
	m := setupModel(t)

	nc := NamecheapSettings{
		Username:      "myuser",
		APIKey:        "mykey",
		CachedDomains: []string{"foo.com", "bar.io"},
	}

	// save namecheap settings
	m = processMsg(t, m, saveNamecheapMsg{settings: nc})

	if !m.NamecheapConfigured() {
		t.Error("namecheap should be configured after save")
	}

	// simulate reload: re-read configs from store
	m.loadConfigs()

	if !m.NamecheapConfigured() {
		t.Error("namecheap should be configured after reload")
	}
	if m.ncConfig.Username != "myuser" {
		t.Errorf("Username = %q, want %q", m.ncConfig.Username, "myuser")
	}
	if m.ncConfig.APIKey != "mykey" {
		t.Errorf("APIKey = %q, want %q", m.ncConfig.APIKey, "mykey")
	}
	if len(m.ncConfig.CachedDomains) != 2 {
		t.Fatalf("CachedDomains = %d, want 2", len(m.ncConfig.CachedDomains))
	}
	if m.ncConfig.CachedDomains[0] != "foo.com" || m.ncConfig.CachedDomains[1] != "bar.io" {
		t.Errorf("CachedDomains = %v, want [foo.com bar.io]", m.ncConfig.CachedDomains)
	}
}

func TestIntegrationGmailSettingsPersist(t *testing.T) {
	m := setupModel(t)

	gm := GmailSettings{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Token: &gmail.Token{
			AccessToken:  "access",
			RefreshToken: "refresh",
			Expiry:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	m = processMsg(t, m, saveGmailMsg{settings: gm})

	if !m.GmailConfigured() {
		t.Error("gmail should be configured after save")
	}

	// reload from store
	m.loadConfigs()

	if !m.GmailConfigured() {
		t.Error("gmail should be configured after reload")
	}
	if m.gmConfig.ClientID != "client-id" {
		t.Errorf("ClientID = %q, want %q", m.gmConfig.ClientID, "client-id")
	}
	if m.gmConfig.Token == nil || m.gmConfig.Token.RefreshToken != "refresh" {
		t.Errorf("Token.RefreshToken = %v, want %q", m.gmConfig.Token, "refresh")
	}
}

func TestIntegrationTwilioSettingsPersist(t *testing.T) {
	m := setupModel(t)

	tw := TwilioSettings{
		AccountSID:         "sid-123",
		AuthToken:          "tok-abc",
		PreferredCountries: []string{"GB", "US"},
	}

	m = processMsg(t, m, saveTwilioMsg{settings: tw})

	if !m.TwilioConfigured() {
		t.Error("twilio should be configured after save")
	}

	m.loadConfigs()

	if !m.TwilioConfigured() {
		t.Error("twilio should be configured after reload")
	}
	if m.twConfig.AccountSID != "sid-123" {
		t.Errorf("AccountSID = %q, want %q", m.twConfig.AccountSID, "sid-123")
	}
	if m.twConfig.AuthToken != "tok-abc" {
		t.Errorf("AuthToken = %q, want %q", m.twConfig.AuthToken, "tok-abc")
	}
	if len(m.twConfig.PreferredCountries) != 2 {
		t.Fatalf("PreferredCountries = %d, want 2", len(m.twConfig.PreferredCountries))
	}
}

func TestIntegrationGmailDisconnect(t *testing.T) {
	m := setupModel(t)

	gm := GmailSettings{
		ClientID:     "id",
		ClientSecret: "secret",
		Token:        &gmail.Token{RefreshToken: "rt"},
	}
	m = processMsg(t, m, saveGmailMsg{settings: gm})

	if !m.GmailConfigured() {
		t.Fatal("gmail should be configured after save")
	}

	// disconnect
	m = processMsg(t, m, disconnectGmailMsg{})

	if m.GmailConfigured() {
		t.Error("gmail should not be configured after disconnect")
	}

	// reload from store to confirm persistence
	m.loadConfigs()

	if m.GmailConfigured() {
		t.Error("gmail should not be configured after reload")
	}
}

func TestIntegrationFeatureGatingMatchesStoredState(t *testing.T) {
	m := setupModel(t)

	// initially nothing configured
	if m.NamecheapConfigured() {
		t.Error("namecheap should not be configured initially")
	}
	if m.GmailConfigured() {
		t.Error("gmail should not be configured initially")
	}
	if m.TwilioConfigured() {
		t.Error("twilio should not be configured initially")
	}

	// configure namecheap only
	m = processMsg(t, m, saveNamecheapMsg{settings: NamecheapSettings{
		Username: "u", APIKey: "k",
	}})

	if !m.NamecheapConfigured() {
		t.Error("namecheap should be configured")
	}
	if m.GmailConfigured() {
		t.Error("gmail should still not be configured")
	}
	if m.TwilioConfigured() {
		t.Error("twilio should still not be configured")
	}
}

// burn cascade integration tests

func TestIntegrationBurnWithCredentials(t *testing.T) {
	m := setupModel(t)
	id := testIdentity()
	m = saveIdentity(t, m, id)

	now := time.Now()
	for i := range 3 {
		c := credential.Credential{
			ID:         fmt.Sprintf("cred-burn-%03d", i),
			IdentityID: id.ID,
			Label:      fmt.Sprintf("Account %d", i),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		m = saveCredential(t, m, c)
	}

	// start burn flow
	result, _ := m.Update(burnStartMsg{identity: id})
	rm := result.(Model)
	if rm.active != viewBurn {
		t.Fatalf("active = %d, want viewBurn", rm.active)
	}
	if rm.burn.phase != burnConfirm {
		t.Fatalf("phase = %d, want burnConfirm", rm.burn.phase)
	}

	// execute the burn directly through the model
	req := rm.buildBurnRequest(id)
	burnResult := burn.Execute(context.Background(), req)

	if burnResult.HasErrors() {
		t.Errorf("unexpected burn errors: %s", burnResult.Summary())
	}
	if burnResult.CredentialsCount != 3 {
		t.Errorf("credentials deleted = %d, want 3", burnResult.CredentialsCount)
	}

	// verify identity is gone
	ids, err := m.identities.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ids {
		if i.ID == id.ID {
			t.Error("identity should have been deleted by burn")
		}
	}

	// verify all credentials are gone
	creds, err := m.credentials.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range creds {
		if c.IdentityID == id.ID {
			t.Errorf("credential %s should have been deleted by burn", c.ID)
		}
	}
}

func TestIntegrationBurnNoCredentials(t *testing.T) {
	m := setupModel(t)
	id := testIdentity()
	m = saveIdentity(t, m, id)

	req := m.buildBurnRequest(id)
	result := burn.Execute(context.Background(), req)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %s", result.Summary())
	}
	if result.CredentialsCount != 0 {
		t.Errorf("credentials deleted = %d, want 0", result.CredentialsCount)
	}

	// identity should be deleted
	ids, err := m.identities.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("identities count = %d, want 0", len(ids))
	}
}

type fakeForwarder struct {
	calls []struct{ domain, mailbox string }
	err   error
}

func (f *fakeForwarder) RemoveForwarding(_ context.Context, domain, mailbox string) error {
	f.calls = append(f.calls, struct{ domain, mailbox string }{domain, mailbox})
	return f.err
}

type fakeReleaser struct {
	calls []string
	err   error
}

func (f *fakeReleaser) ReleaseNumber(_ context.Context, numberSID string) error {
	f.calls = append(f.calls, numberSID)
	return f.err
}

func TestIntegrationBurnWithExternalServices(t *testing.T) {
	m := setupModel(t)

	// generate an identity with an email on the configured domain
	gen := identity.New()
	id := gen.Generate()
	m = saveIdentity(t, m, id)

	now := time.Now()
	m = saveCredential(t, m, credential.Credential{
		ID:         "cred-ext-001",
		IdentityID: id.ID,
		Label:      "TestSvc",
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	fwd := &fakeForwarder{}
	rel := &fakeReleaser{}
	m.SetExternalServices(ExternalServices{
		Forwarder:   fwd,
		Releaser:    rel,
		EmailDomain: "zburn.id",
		PhoneForIdentity: func(identityID string) *burn.PhoneConfig {
			if identityID == id.ID {
				return &burn.PhoneConfig{NumberSID: "PN_test", PhoneNumber: "+441234567890"}
			}
			return nil
		},
	})

	req := m.buildBurnRequest(id)
	result := burn.Execute(context.Background(), req)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %s", result.Summary())
	}

	// forwarder should have been called
	if len(fwd.calls) != 1 {
		t.Fatalf("forwarder calls = %d, want 1", len(fwd.calls))
	}
	if fwd.calls[0].domain != "zburn.id" {
		t.Errorf("forwarder domain = %q, want %q", fwd.calls[0].domain, "zburn.id")
	}

	// releaser should have been called
	if len(rel.calls) != 1 {
		t.Fatalf("releaser calls = %d, want 1", len(rel.calls))
	}
	if rel.calls[0] != "PN_test" {
		t.Errorf("releaser SID = %q, want %q", rel.calls[0], "PN_test")
	}

	// credential should be deleted
	if result.CredentialsCount != 1 {
		t.Errorf("credentials deleted = %d, want 1", result.CredentialsCount)
	}
}

func TestIntegrationBurnWithFailingExternalService(t *testing.T) {
	m := setupModel(t)

	gen := identity.New()
	id := gen.Generate()
	m = saveIdentity(t, m, id)

	fwd := &fakeForwarder{err: fmt.Errorf("network timeout")}
	rel := &fakeReleaser{err: fmt.Errorf("twilio api error")}
	m.SetExternalServices(ExternalServices{
		Forwarder:   fwd,
		Releaser:    rel,
		EmailDomain: "zburn.id",
		PhoneForIdentity: func(identityID string) *burn.PhoneConfig {
			if identityID == id.ID {
				return &burn.PhoneConfig{NumberSID: "PN_fail", PhoneNumber: "+441234"}
			}
			return nil
		},
	})

	req := m.buildBurnRequest(id)
	result := burn.Execute(context.Background(), req)

	// should have errors from external services
	if !result.HasErrors() {
		t.Error("should have errors when external services fail")
	}

	// identity should still be deleted (best-effort)
	ids, err := m.identities.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ids {
		if i.ID == id.ID {
			t.Error("identity should still be deleted even when external services fail")
		}
	}
}

// store lifecycle tests

func TestIntegrationFreshStoreEmpty(t *testing.T) {
	m := setupModel(t)

	ids, err := m.identities.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("identities = %d, want 0", len(ids))
	}

	creds, err := m.credentials.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 0 {
		t.Errorf("credentials = %d, want 0", len(creds))
	}
}

func TestIntegrationStoreReopenPersists(t *testing.T) {
	password := "testpass"
	dir := t.TempDir()
	fs := zfilesystem.NewOSFileSystem(dir)

	// first session: open store, save data, close
	s1, err := zstore.Open(fs, []byte(password))
	if err != nil {
		t.Fatal(err)
	}

	idCol1, err := zstore.NewCollection[identity.Identity](s1, "identities")
	if err != nil {
		t.Fatal(err)
	}

	id := testIdentity()
	if err := idCol1.Put(id.ID, id); err != nil {
		t.Fatal(err)
	}

	credCol1, err := zstore.NewCollection[credential.Credential](s1, "credentials")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	cred := credential.Credential{
		ID:         "cred-reopen-001",
		IdentityID: id.ID,
		Label:      "TestService",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := credCol1.Put(cred.ID, cred); err != nil {
		t.Fatal(err)
	}

	s1.Close()

	// second session: reopen with same password
	s2, err := zstore.Open(fs, []byte(password))
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	idCol2, err := zstore.NewCollection[identity.Identity](s2, "identities")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := idCol2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("identities = %d, want 1", len(ids))
	}
	if ids[0].ID != id.ID {
		t.Errorf("identity ID = %q, want %q", ids[0].ID, id.ID)
	}
	if ids[0].FirstName != id.FirstName {
		t.Errorf("FirstName = %q, want %q", ids[0].FirstName, id.FirstName)
	}

	credCol2, err := zstore.NewCollection[credential.Credential](s2, "credentials")
	if err != nil {
		t.Fatal(err)
	}

	creds, err := credCol2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("credentials = %d, want 1", len(creds))
	}
	if creds[0].ID != cred.ID {
		t.Errorf("credential ID = %q, want %q", creds[0].ID, cred.ID)
	}
	if creds[0].Label != "TestService" {
		t.Errorf("label = %q, want %q", creds[0].Label, "TestService")
	}
}

func TestIntegrationStoreWrongPassword(t *testing.T) {
	dir := t.TempDir()
	fs := zfilesystem.NewOSFileSystem(dir)

	// create store with password
	s, err := zstore.Open(fs, []byte("correct"))
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	// try to reopen with wrong password
	_, err = zstore.Open(fs, []byte("wrong"))
	if err == nil {
		t.Fatal("should fail with wrong password")
	}
	if err != zstore.ErrWrongPassword {
		t.Errorf("error = %v, want ErrWrongPassword", err)
	}
}

// root model password → store → menu integration

func TestIntegrationPasswordToStoreFlow(t *testing.T) {
	dir := t.TempDir()
	m := New("1.0", dir, identity.New(), true)
	if m.active != viewPassword {
		t.Fatalf("active = %d, want viewPassword", m.active)
	}

	// submit password through the model
	result, _ := m.Update(passwordSubmitMsg{password: "mypassword"})
	rm := result.(Model)

	if rm.active != viewMenu {
		t.Errorf("active = %d, want viewMenu", rm.active)
	}
	if rm.store == nil {
		t.Error("store should be initialized")
	}
	if rm.identities == nil {
		t.Error("identities collection should be initialized")
	}
	if rm.credentials == nil {
		t.Error("credentials collection should be initialized")
	}
	if rm.configs == nil {
		t.Error("configs collection should be initialized")
	}
}

// full user flow: password → save identity → browse → view → burn

func TestIntegrationFullUserFlow(t *testing.T) {
	dir := t.TempDir()
	m := New("1.0", dir, identity.New(), true)

	// open store
	result, _ := m.Update(passwordSubmitMsg{password: "secret"})
	rm := result.(Model)
	if rm.active != viewMenu {
		t.Fatalf("active = %d, want viewMenu after password", rm.active)
	}

	// generate and save an identity
	result, _ = rm.Update(navigateMsg{view: viewGenerate})
	rm = result.(Model)
	id := rm.generate.identity

	result, _ = rm.Update(saveIdentityMsg{identity: id})
	rm = result.(Model)

	// browse list
	result, _ = rm.Update(navigateMsg{view: viewList})
	rm = result.(Model)
	if rm.active != viewList {
		t.Fatalf("active = %d, want viewList", rm.active)
	}
	if len(rm.list.identities) != 1 {
		t.Fatalf("identities = %d, want 1", len(rm.list.identities))
	}

	// view the identity
	result, _ = rm.Update(viewIdentityMsg{identity: rm.list.identities[0]})
	rm = result.(Model)
	if rm.active != viewDetail {
		t.Fatalf("active = %d, want viewDetail", rm.active)
	}

	// add a credential
	cred := credential.Credential{
		ID:         "cred-flow-001",
		IdentityID: id.ID,
		Label:      "FlowTest",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	result, _ = rm.Update(saveCredentialMsg{credential: cred})
	rm = result.(Model)

	// start burn
	result, _ = rm.Update(burnStartMsg{identity: id})
	rm = result.(Model)
	if rm.active != viewBurn {
		t.Fatalf("active = %d, want viewBurn", rm.active)
	}

	// execute burn
	req := rm.buildBurnRequest(id)
	burnResult := burn.Execute(context.Background(), req)
	if burnResult.HasErrors() {
		t.Errorf("burn errors: %s", burnResult.Summary())
	}

	// verify everything is cleaned up
	ids, err := rm.identities.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("identities after burn = %d, want 0", len(ids))
	}

	creds, err := rm.credentials.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 0 {
		t.Errorf("credentials after burn = %d, want 0", len(creds))
	}
}

// settings persistence combined with burn plan

func TestIntegrationSettingsAffectBurnPlan(t *testing.T) {
	m := setupModel(t)

	gen := identity.New()
	id := gen.Generate()
	m = saveIdentity(t, m, id)

	// burn plan without external services — only credentials step
	result, _ := m.Update(burnStartMsg{identity: id})
	rm := result.(Model)
	if len(rm.burn.plan) != 1 {
		t.Errorf("plan steps without external = %d, want 1", len(rm.burn.plan))
	}

	// configure external services
	fwd := &fakeForwarder{}
	rel := &fakeReleaser{}
	rm.SetExternalServices(ExternalServices{
		Forwarder:   fwd,
		Releaser:    rel,
		EmailDomain: "zburn.id",
		PhoneForIdentity: func(identityID string) *burn.PhoneConfig {
			if identityID == id.ID {
				return &burn.PhoneConfig{NumberSID: "PN_plan", PhoneNumber: "+441234"}
			}
			return nil
		},
	})

	// burn plan should now include email and phone steps
	result, _ = rm.Update(burnStartMsg{identity: id})
	rm = result.(Model)
	if len(rm.burn.plan) != 3 {
		t.Errorf("plan steps with external = %d, want 3", len(rm.burn.plan))
	}
}
