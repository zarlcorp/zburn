package burn

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

// fakes

type fakeCredentialStore struct {
	creds   []credential.Credential
	listErr error
	delErr  map[string]error // per-ID delete errors
	deleted []string
}

func (f *fakeCredentialStore) List() ([]credential.Credential, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.creds, nil
}

func (f *fakeCredentialStore) Delete(id string) error {
	if err, ok := f.delErr[id]; ok {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeIdentityStore struct {
	deleted []string
	delErr  error
}

func (f *fakeIdentityStore) Delete(id string) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeForwarder struct {
	calls []forwarderCall
	err   error
}

type forwarderCall struct {
	domain, mailbox string
}

func (f *fakeForwarder) RemoveForwarding(_ context.Context, domain, mailbox string) error {
	f.calls = append(f.calls, forwarderCall{domain, mailbox})
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

// helpers

func testIdentity() identity.Identity {
	return identity.Identity{
		ID:        "id-001",
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "swiftwolf1234@zburn.id",
		Phone:     "(555) 123-4567",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func testCreds(identityID string, n int) []credential.Credential {
	creds := make([]credential.Credential, n)
	for i := range n {
		creds[i] = credential.Credential{
			ID:         fmt.Sprintf("cred-%03d", i),
			IdentityID: identityID,
			Label:      fmt.Sprintf("account %d", i),
		}
	}
	return creds
}

// tests

func TestExecuteFullCascade(t *testing.T) {
	cs := &fakeCredentialStore{creds: testCreds("id-001", 3)}
	is := &fakeIdentityStore{}
	fwd := &fakeForwarder{}
	rel := &fakeReleaser{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
		Email:       &EmailConfig{Domain: "zburn.id", Mailbox: "swiftwolf1234"},
		Forwarder:   fwd,
		Phone:       &PhoneConfig{NumberSID: "PN_abc123", PhoneNumber: "+447123456789"},
		Releaser:    rel,
	}

	result := Execute(context.Background(), req)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %s", result.Summary())
	}

	if result.CredentialsCount != 3 {
		t.Errorf("credentials deleted = %d, want 3", result.CredentialsCount)
	}

	if len(cs.deleted) != 3 {
		t.Errorf("credential store deletes = %d, want 3", len(cs.deleted))
	}

	if len(fwd.calls) != 1 || fwd.calls[0].domain != "zburn.id" || fwd.calls[0].mailbox != "swiftwolf1234" {
		t.Errorf("forwarder calls = %v, want [{zburn.id swiftwolf1234}]", fwd.calls)
	}

	if len(rel.calls) != 1 || rel.calls[0] != "PN_abc123" {
		t.Errorf("releaser calls = %v, want [PN_abc123]", rel.calls)
	}

	if len(is.deleted) != 1 || is.deleted[0] != "id-001" {
		t.Errorf("identity deletes = %v, want [id-001]", is.deleted)
	}

	if len(result.Steps) != 4 {
		t.Errorf("steps = %d, want 4", len(result.Steps))
	}
}

func TestExecuteNoEmailNoPhone(t *testing.T) {
	cs := &fakeCredentialStore{creds: testCreds("id-001", 2)}
	is := &fakeIdentityStore{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
	}

	result := Execute(context.Background(), req)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %s", result.Summary())
	}

	// only credentials + identity steps
	if len(result.Steps) != 2 {
		t.Errorf("steps = %d, want 2", len(result.Steps))
	}

	if result.CredentialsCount != 2 {
		t.Errorf("credentials deleted = %d, want 2", result.CredentialsCount)
	}
}

func TestExecuteNoMatchingCredentials(t *testing.T) {
	// credentials belong to a different identity
	otherCreds := testCreds("id-999", 5)
	cs := &fakeCredentialStore{creds: otherCreds}
	is := &fakeIdentityStore{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
	}

	result := Execute(context.Background(), req)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %s", result.Summary())
	}

	if result.CredentialsCount != 0 {
		t.Errorf("credentials deleted = %d, want 0", result.CredentialsCount)
	}
}

func TestExecuteEmailFailureContinues(t *testing.T) {
	cs := &fakeCredentialStore{creds: testCreds("id-001", 1)}
	is := &fakeIdentityStore{}
	fwd := &fakeForwarder{err: fmt.Errorf("network timeout")}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
		Email:       &EmailConfig{Domain: "zburn.id", Mailbox: "swiftwolf1234"},
		Forwarder:   fwd,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors when email forwarding fails")
	}

	// credentials should still be deleted
	if result.CredentialsCount != 1 {
		t.Errorf("credentials deleted = %d, want 1", result.CredentialsCount)
	}

	// identity should still be deleted
	if len(is.deleted) != 1 {
		t.Errorf("identity deletes = %d, want 1", len(is.deleted))
	}

	// summary should mention the failure
	if !strings.Contains(result.Summary(), "network timeout") {
		t.Errorf("summary should contain error: %s", result.Summary())
	}
}

func TestExecutePhoneFailureContinues(t *testing.T) {
	cs := &fakeCredentialStore{}
	is := &fakeIdentityStore{}
	rel := &fakeReleaser{err: fmt.Errorf("twilio api error")}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
		Phone:       &PhoneConfig{NumberSID: "PN_abc", PhoneNumber: "+447123456789"},
		Releaser:    rel,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors when phone release fails")
	}

	// identity should still be deleted
	if len(is.deleted) != 1 {
		t.Errorf("identity deletes = %d, want 1", len(is.deleted))
	}

	if !strings.Contains(result.Summary(), "twilio api error") {
		t.Errorf("summary should contain error: %s", result.Summary())
	}
}

func TestExecuteCredentialListError(t *testing.T) {
	cs := &fakeCredentialStore{listErr: fmt.Errorf("store corrupt")}
	is := &fakeIdentityStore{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors when credential list fails")
	}

	// identity should still be deleted despite credential list failure
	if len(is.deleted) != 1 {
		t.Errorf("identity deletes = %d, want 1", len(is.deleted))
	}
}

func TestExecutePartialCredentialDeleteError(t *testing.T) {
	cs := &fakeCredentialStore{
		creds:  testCreds("id-001", 3),
		delErr: map[string]error{"cred-001": fmt.Errorf("locked")},
	}
	is := &fakeIdentityStore{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors when some credential deletes fail")
	}

	// 2 of 3 should succeed
	if result.CredentialsCount != 2 {
		t.Errorf("credentials deleted = %d, want 2", result.CredentialsCount)
	}

	if !strings.Contains(result.Summary(), "locked") {
		t.Errorf("summary should mention locked error: %s", result.Summary())
	}
}

func TestExecuteIdentityDeleteError(t *testing.T) {
	cs := &fakeCredentialStore{}
	is := &fakeIdentityStore{delErr: fmt.Errorf("permission denied")}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors when identity delete fails")
	}

	if !strings.Contains(result.Summary(), "permission denied") {
		t.Errorf("summary should mention error: %s", result.Summary())
	}
}

func TestExecuteAllFailures(t *testing.T) {
	cs := &fakeCredentialStore{listErr: fmt.Errorf("store error")}
	is := &fakeIdentityStore{delErr: fmt.Errorf("identity error")}
	fwd := &fakeForwarder{err: fmt.Errorf("email error")}
	rel := &fakeReleaser{err: fmt.Errorf("phone error")}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Identities:  is,
		Email:       &EmailConfig{Domain: "zburn.id", Mailbox: "test"},
		Forwarder:   fwd,
		Phone:       &PhoneConfig{NumberSID: "PN_x", PhoneNumber: "+441234"},
		Releaser:    rel,
	}

	result := Execute(context.Background(), req)

	if !result.HasErrors() {
		t.Error("should have errors")
	}

	// all 4 steps should be attempted
	if len(result.Steps) != 4 {
		t.Errorf("steps = %d, want 4", len(result.Steps))
	}

	// every step should have an error
	for i, s := range result.Steps {
		if s.Err == nil {
			t.Errorf("step %d (%s) should have error", i, s.Description)
		}
	}

	summary := result.Summary()
	if !strings.Contains(summary, "with errors") {
		t.Errorf("summary should say 'with errors': %s", summary)
	}
}

func TestPlanFullConfig(t *testing.T) {
	cs := &fakeCredentialStore{creds: testCreds("id-001", 3)}
	fwd := &fakeForwarder{}
	rel := &fakeReleaser{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
		Email:       &EmailConfig{Domain: "zburn.id", Mailbox: "swiftwolf1234"},
		Forwarder:   fwd,
		Phone:       &PhoneConfig{NumberSID: "PN_abc", PhoneNumber: "+447123456789"},
		Releaser:    rel,
	}

	steps := Plan(req)

	if len(steps) != 3 {
		t.Fatalf("plan steps = %d, want 3", len(steps))
	}

	if !strings.Contains(steps[0], "credentials (3)") {
		t.Errorf("step 0 = %q, want credentials count", steps[0])
	}

	if !strings.Contains(steps[1], "swiftwolf1234@zburn.id") {
		t.Errorf("step 1 = %q, want email address", steps[1])
	}

	if !strings.Contains(steps[2], "+447123456789") {
		t.Errorf("step 2 = %q, want phone number", steps[2])
	}
}

func TestPlanNoExternal(t *testing.T) {
	cs := &fakeCredentialStore{}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
	}

	steps := Plan(req)

	if len(steps) != 1 {
		t.Fatalf("plan steps = %d, want 1", len(steps))
	}

	if !strings.Contains(steps[0], "credentials") {
		t.Errorf("step 0 = %q, want credentials", steps[0])
	}
}

func TestPlanCredentialListError(t *testing.T) {
	cs := &fakeCredentialStore{listErr: fmt.Errorf("oops")}

	req := Request{
		Identity:    testIdentity(),
		Credentials: cs,
	}

	steps := Plan(req)

	// should still include credentials step, just without count
	if len(steps) != 1 {
		t.Fatalf("plan steps = %d, want 1", len(steps))
	}

	if !strings.Contains(steps[0], "credentials") {
		t.Errorf("step 0 = %q, want credentials mention", steps[0])
	}
}

func TestResultSummaryNoErrors(t *testing.T) {
	r := Result{
		Name:             "Jane Doe",
		CredentialsCount: 3,
		Steps: []StepStatus{
			{Description: "deleted 3 credentials"},
			{Description: "removed email forwarding for test@zburn.id"},
			{Description: "deleted identity"},
		},
	}

	s := r.Summary()
	if strings.Contains(s, "with errors") {
		t.Errorf("summary should not say 'with errors': %s", s)
	}
	if !strings.Contains(s, "burned Jane Doe") {
		t.Errorf("summary should contain name: %s", s)
	}
}

func TestResultSummaryWithErrors(t *testing.T) {
	r := Result{
		Name: "Jane Doe",
		Steps: []StepStatus{
			{Description: "deleted 0 credentials"},
			{Description: "email forwarding removal for test@zburn.id", Err: fmt.Errorf("timeout")},
			{Description: "deleted identity"},
		},
	}

	s := r.Summary()
	if !strings.Contains(s, "with errors") {
		t.Errorf("summary should say 'with errors': %s", s)
	}
	if !strings.Contains(s, "timeout") {
		t.Errorf("summary should contain error: %s", s)
	}
}

func TestResultHasErrors(t *testing.T) {
	tests := []struct {
		name  string
		steps []StepStatus
		want  bool
	}{
		{
			name: "no errors",
			steps: []StepStatus{
				{Description: "ok"},
			},
			want: false,
		},
		{
			name: "with error",
			steps: []StepStatus{
				{Description: "ok"},
				{Description: "bad", Err: fmt.Errorf("fail")},
			},
			want: true,
		},
		{
			name:  "empty",
			steps: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{Steps: tt.steps}
			if got := r.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}
