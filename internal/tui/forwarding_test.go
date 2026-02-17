package tui

import (
	"context"
	"fmt"
	"testing"

	"github.com/zarlcorp/zburn/internal/namecheap"
)

// fakeForwardingSetter records calls and can return per-domain errors.
type fakeForwardingSetter struct {
	calls  []fakeForwardingCall
	errors map[string]error
}

type fakeForwardingCall struct {
	domain string
	rules  []namecheap.ForwardingRule
}

func (f *fakeForwardingSetter) SetForwarding(_ context.Context, domain string, rules []namecheap.ForwardingRule) error {
	f.calls = append(f.calls, fakeForwardingCall{domain: domain, rules: rules})
	if f.errors != nil {
		return f.errors[domain]
	}
	return nil
}

func TestSetupCatchAllForwarding(t *testing.T) {
	tests := []struct {
		name         string
		domains      []string
		errors       map[string]error
		wantOK       int
		wantFail     int
		wantCalls    int
		wantSkipped  []string // domains that should NOT appear in calls
		wantCalled   []string // domains that SHOULD appear in calls
	}{
		{
			name:       "all succeed",
			domains:    []string{"alpha.com", "bravo.io", "charlie.net"},
			wantOK:     3,
			wantFail:   0,
			wantCalls:  3,
			wantCalled: []string{"alpha.com", "bravo.io", "charlie.net"},
		},
		{
			name:    "one fails",
			domains: []string{"alpha.com", "bravo.io", "charlie.net"},
			errors:  map[string]error{"bravo.io": fmt.Errorf("api error")},
			wantOK:  2,
			wantFail: 1,
			wantCalls: 3,
		},
		{
			name:    "all fail",
			domains: []string{"alpha.com", "bravo.io"},
			errors: map[string]error{
				"alpha.com": fmt.Errorf("err1"),
				"bravo.io":  fmt.Errorf("err2"),
			},
			wantOK:    0,
			wantFail:  2,
			wantCalls: 2,
		},
		{
			name:        "excluded domains skipped",
			domains:     []string{"zarlcorp.com", "zarl.dev", "burner.com"},
			wantOK:      1,
			wantFail:    0,
			wantCalls:   1,
			wantSkipped: []string{"zarlcorp.com", "zarl.dev"},
			wantCalled:  []string{"burner.com"},
		},
		{
			name:      "empty domain list",
			domains:   nil,
			wantOK:    0,
			wantFail:  0,
			wantCalls: 0,
		},
		{
			name:        "only excluded domains",
			domains:     []string{"zarlcorp.com", "zarl.dev"},
			wantOK:      0,
			wantFail:    0,
			wantCalls:   0,
			wantSkipped: []string{"zarlcorp.com", "zarl.dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeForwardingSetter{errors: tt.errors}

			ok, fail := setupCatchAllForwarding(context.Background(), f, tt.domains, "user@gmail.com")

			if ok != tt.wantOK {
				t.Errorf("successes = %d, want %d", ok, tt.wantOK)
			}
			if fail != tt.wantFail {
				t.Errorf("failures = %d, want %d", fail, tt.wantFail)
			}
			if len(f.calls) != tt.wantCalls {
				t.Errorf("calls = %d, want %d", len(f.calls), tt.wantCalls)
			}

			// verify skipped domains
			called := make(map[string]bool)
			for _, c := range f.calls {
				called[c.domain] = true
			}
			for _, d := range tt.wantSkipped {
				if called[d] {
					t.Errorf("domain %q should have been skipped", d)
				}
			}
			for _, d := range tt.wantCalled {
				if !called[d] {
					t.Errorf("domain %q should have been called", d)
				}
			}

			// verify rule content on every call
			for _, c := range f.calls {
				if len(c.rules) != 1 {
					t.Errorf("rules for %q: len = %d, want 1", c.domain, len(c.rules))
					continue
				}
				if c.rules[0].Mailbox != "*" {
					t.Errorf("rules[0].Mailbox = %q, want %q", c.rules[0].Mailbox, "*")
				}
				if c.rules[0].ForwardTo != "user@gmail.com" {
					t.Errorf("rules[0].ForwardTo = %q, want %q", c.rules[0].ForwardTo, "user@gmail.com")
				}
			}
		})
	}
}

func TestForwardingFlash(t *testing.T) {
	tests := []struct {
		name string
		msg  forwardingResultMsg
		want string
	}{
		{
			name: "all success",
			msg:  forwardingResultMsg{successes: 8, failures: 0},
			want: "forwarding set up on 8 domains",
		},
		{
			name: "mixed",
			msg:  forwardingResultMsg{successes: 6, failures: 2},
			want: "forwarding: 6 ok, 2 failed",
		},
		{
			name: "all failed",
			msg:  forwardingResultMsg{successes: 0, failures: 3},
			want: "forwarding: 3 failed",
		},
		{
			name: "zero domains",
			msg:  forwardingResultMsg{successes: 0, failures: 0},
			want: "forwarding set up on 0 domains",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forwardingFlash(tt.msg)
			if got != tt.want {
				t.Errorf("forwardingFlash() = %q, want %q", got, tt.want)
			}
		})
	}
}
