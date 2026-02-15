package store

import (
	"errors"
	"testing"
	"time"

	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/zburn/internal/identity"
)

func newTestIdentity(id string, createdAt time.Time) identity.Identity {
	return identity.Identity{
		ID:        id,
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane@example.com",
		Phone:     "555-0100",
		Street:    "123 Main St",
		City:      "Portland",
		State:     "OR",
		Zip:       "97201",
		DOB:       time.Date(1990, 6, 15, 0, 0, 0, 0, time.UTC),
		Password:  "hunter2",
		CreatedAt: createdAt,
	}
}

func openTestStore(t *testing.T) (*Store, *zfilesystem.MemFS) {
	t.Helper()
	fs := zfilesystem.NewMemFS()
	s, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, fs
}

func TestFirstRunCreatesSaltAndVerify(t *testing.T) {
	fs := zfilesystem.NewMemFS()
	s, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	salt, err := fs.ReadFile("salt")
	if err != nil {
		t.Fatal("salt file not created")
	}
	if len(salt) != 16 {
		t.Fatalf("salt length: got %d, want 16", len(salt))
	}

	verify, err := fs.ReadFile("verify")
	if err != nil {
		t.Fatal("verify file not created")
	}
	if len(verify) == 0 {
		t.Fatal("verify file is empty")
	}
}

func TestReopenWithCorrectPassword(t *testing.T) {
	fs := zfilesystem.NewMemFS()

	s1, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	s1.Close()

	s2, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	s2.Close()
}

func TestWrongPasswordFails(t *testing.T) {
	fs := zfilesystem.NewMemFS()

	s, err := Open(fs, "correct")
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	s.Close()

	_, err = Open(fs, "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestSaveAndGet(t *testing.T) {
	s, _ := openTestStore(t)

	now := time.Now().Truncate(time.Second).UTC()
	want := newTestIdentity("abc123", now)

	if err := s.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get("abc123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	assertIdentityEqual(t, want, got)
}

func TestGetNotFound(t *testing.T) {
	s, _ := openTestStore(t)

	_, err := s.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("get nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestListSortedByCreatedAtDesc(t *testing.T) {
	s, _ := openTestStore(t)

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	for _, id := range []struct {
		name string
		at   time.Time
	}{
		{"id-oldest", t1},
		{"id-newest", t2},
		{"id-middle", t3},
	} {
		identity := newTestIdentity(id.name, id.at)
		identity.ID = id.name
		if err := s.Save(identity); err != nil {
			t.Fatalf("save %s: %v", id.name, err)
		}
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("list length: got %d, want 3", len(list))
	}

	if list[0].ID != "id-newest" {
		t.Errorf("list[0].ID = %s, want id-newest", list[0].ID)
	}
	if list[1].ID != "id-middle" {
		t.Errorf("list[1].ID = %s, want id-middle", list[1].ID)
	}
	if list[2].ID != "id-oldest" {
		t.Errorf("list[2].ID = %s, want id-oldest", list[2].ID)
	}
}

func TestDelete(t *testing.T) {
	s, _ := openTestStore(t)

	now := time.Now().UTC()
	if err := s.Save(newTestIdentity("to-delete", now)); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := s.Delete("to-delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := s.Get("to-delete")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete: got %v, want ErrNotFound", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s, _ := openTestStore(t)

	err := s.Delete("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestCloseErasesKey(t *testing.T) {
	fs := zfilesystem.NewMemFS()
	s, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	s.Close()

	if s.key != nil {
		t.Fatal("key not nil after close")
	}
}

func TestRoundTripPreservesAllFields(t *testing.T) {
	s, _ := openTestStore(t)

	want := identity.Identity{
		ID:        "full-fields",
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     "alice@example.com",
		Phone:     "555-0199",
		Street:    "456 Oak Ave",
		City:      "Seattle",
		State:     "WA",
		Zip:       "98101",
		DOB:       time.Date(1985, 3, 22, 0, 0, 0, 0, time.UTC),
		Password:  "s3cret!@#",
		CreatedAt: time.Date(2025, 12, 1, 10, 30, 0, 0, time.UTC),
	}

	if err := s.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get("full-fields")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	assertIdentityEqual(t, want, got)
}

func TestListEmptyStore(t *testing.T) {
	s, _ := openTestStore(t)

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 0 {
		t.Fatalf("list length: got %d, want 0", len(list))
	}
}

func TestDataPersistsAcrossReopen(t *testing.T) {
	fs := zfilesystem.NewMemFS()

	s1, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("first open: %v", err)
	}

	now := time.Date(2025, 7, 4, 12, 0, 0, 0, time.UTC)
	want := newTestIdentity("persist-id", now)
	if err := s1.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	s1.Close()

	s2, err := Open(fs, "testpass")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	got, err := s2.Get("persist-id")
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}

	assertIdentityEqual(t, want, got)
}

func assertIdentityEqual(t *testing.T, want, got identity.Identity) {
	t.Helper()

	checks := []struct {
		field    string
		got, want any
	}{
		{"ID", got.ID, want.ID},
		{"FirstName", got.FirstName, want.FirstName},
		{"LastName", got.LastName, want.LastName},
		{"Email", got.Email, want.Email},
		{"Phone", got.Phone, want.Phone},
		{"Street", got.Street, want.Street},
		{"City", got.City, want.City},
		{"State", got.State, want.State},
		{"Zip", got.Zip, want.Zip},
		{"DOB", got.DOB.UTC(), want.DOB.UTC()},
		{"Password", got.Password, want.Password},
		{"CreatedAt", got.CreatedAt.UTC(), want.CreatedAt.UTC()},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.field, c.got, c.want)
		}
	}
}
