// Package burn implements best-effort cascading deletion of persona resources.
package burn

import (
	"context"
	"fmt"
	"strings"

	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

// CredentialStore lists and deletes credentials.
type CredentialStore interface {
	List() ([]credential.Credential, error)
	Delete(id string) error
}

// IdentityStore deletes identities.
type IdentityStore interface {
	Delete(id string) error
}

// EmailForwarder removes email forwarding rules.
type EmailForwarder interface {
	RemoveForwarding(ctx context.Context, domain, mailbox string) error
}

// PhoneReleaser releases provisioned phone numbers.
type PhoneReleaser interface {
	ReleaseNumber(ctx context.Context, numberSID string) error
}

// EmailConfig holds forwarding details for an identity.
type EmailConfig struct {
	Domain  string // e.g. "zburn.id"
	Mailbox string // e.g. "swiftwolf1234"
}

// PhoneConfig holds provisioned phone details for an identity.
type PhoneConfig struct {
	NumberSID   string // Twilio SID for the provisioned number
	PhoneNumber string // display number e.g. "+447123456789"
}

// Request describes what to burn.
type Request struct {
	Identity    identity.Identity
	Credentials CredentialStore
	Identities  IdentityStore
	Email       *EmailConfig    // nil if no email forwarding configured
	Forwarder   EmailForwarder  // nil if namecheap not configured
	Phone       *PhoneConfig    // nil if no provisioned phone
	Releaser    PhoneReleaser   // nil if twilio not configured
}

// StepStatus records the outcome of one cascade step.
type StepStatus struct {
	Description string
	Err         error
}

// Result summarizes a completed burn.
type Result struct {
	Name             string
	CredentialsCount int
	Steps            []StepStatus
}

// HasErrors returns true if any step failed.
func (r Result) HasErrors() bool {
	for _, s := range r.Steps {
		if s.Err != nil {
			return true
		}
	}
	return false
}

// Summary returns a human-readable summary of the burn result.
func (r Result) Summary() string {
	var b strings.Builder

	if r.HasErrors() {
		fmt.Fprintf(&b, "burned %s (with errors)", r.Name)
	} else {
		fmt.Fprintf(&b, "burned %s", r.Name)
	}

	for _, s := range r.Steps {
		if s.Err != nil {
			fmt.Fprintf(&b, "\n- %s: %v", s.Description, s.Err)
		} else {
			fmt.Fprintf(&b, "\n- %s", s.Description)
		}
	}

	return b.String()
}

// Plan returns a list of human-readable descriptions of what will happen.
// Used to populate the confirmation dialog.
func Plan(req Request) []string {
	var steps []string

	// always list credentials
	creds, err := countCredentials(req)
	if err == nil && creds > 0 {
		steps = append(steps, fmt.Sprintf("delete all credentials (%d)", creds))
	} else if err == nil {
		steps = append(steps, "delete all credentials (0)")
	} else {
		steps = append(steps, "delete all credentials")
	}

	if req.Forwarder != nil && req.Email != nil {
		steps = append(steps, fmt.Sprintf("remove email forwarding for %s@%s", req.Email.Mailbox, req.Email.Domain))
	}

	if req.Releaser != nil && req.Phone != nil {
		steps = append(steps, fmt.Sprintf("release phone number %s", req.Phone.PhoneNumber))
	}

	return steps
}

// Execute runs the burn cascade. It is best-effort: each step is attempted
// regardless of whether previous steps failed. The identity itself is always
// deleted last.
func Execute(ctx context.Context, req Request) Result {
	name := req.Identity.FirstName + " " + req.Identity.LastName
	result := Result{Name: name}

	// 1. delete credentials
	result.deleteCredentials(req)

	// 2. remove email forwarding
	if req.Forwarder != nil && req.Email != nil {
		result.removeEmail(ctx, req)
	}

	// 3. release phone number
	if req.Releaser != nil && req.Phone != nil {
		result.releasePhone(ctx, req)
	}

	// 4. delete identity
	result.deleteIdentity(req)

	return result
}

func (r *Result) deleteCredentials(req Request) {
	creds, err := req.Credentials.List()
	if err != nil {
		r.Steps = append(r.Steps, StepStatus{
			Description: "delete credentials",
			Err:         fmt.Errorf("list credentials: %w", err),
		})
		return
	}

	var matching []credential.Credential
	for _, c := range creds {
		if c.IdentityID == req.Identity.ID {
			matching = append(matching, c)
		}
	}

	var errs []string
	deleted := 0
	for _, c := range matching {
		if err := req.Credentials.Delete(c.ID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", c.ID, err))
			continue
		}
		deleted++
	}

	r.CredentialsCount = deleted

	if len(errs) > 0 {
		r.Steps = append(r.Steps, StepStatus{
			Description: fmt.Sprintf("deleted %d/%d credentials", deleted, len(matching)),
			Err:         fmt.Errorf("%s", strings.Join(errs, "; ")),
		})
		return
	}

	r.Steps = append(r.Steps, StepStatus{
		Description: fmt.Sprintf("deleted %d credentials", deleted),
	})
}

func (r *Result) removeEmail(ctx context.Context, req Request) {
	addr := req.Email.Mailbox + "@" + req.Email.Domain
	err := req.Forwarder.RemoveForwarding(ctx, req.Email.Domain, req.Email.Mailbox)
	if err != nil {
		r.Steps = append(r.Steps, StepStatus{
			Description: fmt.Sprintf("email forwarding removal for %s", addr),
			Err:         err,
		})
		return
	}
	r.Steps = append(r.Steps, StepStatus{
		Description: fmt.Sprintf("removed email forwarding for %s", addr),
	})
}

func (r *Result) releasePhone(ctx context.Context, req Request) {
	err := req.Releaser.ReleaseNumber(ctx, req.Phone.NumberSID)
	if err != nil {
		r.Steps = append(r.Steps, StepStatus{
			Description: fmt.Sprintf("release phone number %s", req.Phone.PhoneNumber),
			Err:         err,
		})
		return
	}
	r.Steps = append(r.Steps, StepStatus{
		Description: fmt.Sprintf("released phone number %s", req.Phone.PhoneNumber),
	})
}

func (r *Result) deleteIdentity(req Request) {
	err := req.Identities.Delete(req.Identity.ID)
	if err != nil {
		r.Steps = append(r.Steps, StepStatus{
			Description: "delete identity",
			Err:         err,
		})
		return
	}
	r.Steps = append(r.Steps, StepStatus{
		Description: "deleted identity",
	})
}

func countCredentials(req Request) (int, error) {
	creds, err := req.Credentials.List()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, c := range creds {
		if c.IdentityID == req.Identity.ID {
			count++
		}
	}
	return count, nil
}
