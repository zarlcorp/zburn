package credential

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
)

func openTestStore(t *testing.T) *zstore.Store {
	t.Helper()
	fs := zfilesystem.NewMemFS()
	s, err := zstore.Open(fs, []byte("testpass"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testCredential(id, identityID string) Credential {
	now := time.Now().Truncate(time.Second).UTC()
	return Credential{
		ID:         id,
		IdentityID: identityID,
		Label:      "test account",
		URL:        "https://example.com",
		Username:   "user@example.com",
		Password:   "s3cret!",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
		Notes:      "some notes",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func hexID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func TestCredentialRoundTrip(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[Credential](s, "credentials")
	if err != nil {
		t.Fatalf("new collection: %v", err)
	}

	want := testCredential("cred-001", "id-abc")
	if err := col.Put(want.ID, want); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := col.Get("cred-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	assertCredentialEqual(t, want, got)
}

func TestCredentialRoundTripOmitEmpty(t *testing.T) {
	s := openTestStore(t)
	col, err := zstore.NewCollection[Credential](s, "credentials")
	if err != nil {
		t.Fatalf("new collection: %v", err)
	}

	now := time.Now().Truncate(time.Second).UTC()
	want := Credential{
		ID:         "cred-002",
		IdentityID: "id-xyz",
		Label:      "minimal",
		URL:        "https://example.com",
		Username:   "user",
		Password:   "pass",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := col.Put(want.ID, want); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := col.Get("cred-002")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.TOTPSecret != "" {
		t.Errorf("TOTPSecret = %q, want empty", got.TOTPSecret)
	}
	if got.Notes != "" {
		t.Errorf("Notes = %q, want empty", got.Notes)
	}

	assertCredentialEqual(t, want, got)
}

type testIdentity struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	CreatedAt time.Time `json:"created_at"`
}

func TestCollectionIsolation(t *testing.T) {
	s := openTestStore(t)

	identities, err := zstore.NewCollection[testIdentity](s, "identities")
	if err != nil {
		t.Fatalf("new identities collection: %v", err)
	}

	credentials, err := zstore.NewCollection[Credential](s, "credentials")
	if err != nil {
		t.Fatalf("new credentials collection: %v", err)
	}

	now := time.Now().Truncate(time.Second).UTC()

	// save an identity
	id := testIdentity{ID: "id-001", FirstName: "Jane", CreatedAt: now}
	if err := identities.Put(id.ID, id); err != nil {
		t.Fatalf("put identity: %v", err)
	}

	// save a credential linked to that identity
	cred := testCredential("cred-001", "id-001")
	if err := credentials.Put(cred.ID, cred); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	// delete the identity
	if err := identities.Delete("id-001"); err != nil {
		t.Fatalf("delete identity: %v", err)
	}

	// identity should be gone
	_, err = identities.Get("id-001")
	if err == nil {
		t.Fatal("identity should be deleted")
	}

	// credential should still exist
	got, err := credentials.Get("cred-001")
	if err != nil {
		t.Fatalf("get credential after identity delete: %v", err)
	}
	if got.ID != "cred-001" {
		t.Errorf("credential ID = %q, want %q", got.ID, "cred-001")
	}
}

func TestCredentialIDGeneration(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		id := hexID()
		if len(id) != 8 {
			t.Fatalf("hex ID length = %d, want 8", len(id))
		}
		if seen[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

func assertCredentialEqual(t *testing.T, want, got Credential) {
	t.Helper()

	checks := []struct {
		field      string
		got, want  any
	}{
		{"ID", got.ID, want.ID},
		{"IdentityID", got.IdentityID, want.IdentityID},
		{"Label", got.Label, want.Label},
		{"URL", got.URL, want.URL},
		{"Username", got.Username, want.Username},
		{"Password", got.Password, want.Password},
		{"TOTPSecret", got.TOTPSecret, want.TOTPSecret},
		{"Notes", got.Notes, want.Notes},
		{"CreatedAt", got.CreatedAt.UTC(), want.CreatedAt.UTC()},
		{"UpdatedAt", got.UpdatedAt.UTC(), want.UpdatedAt.UTC()},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.field, c.got, c.want)
		}
	}
}
