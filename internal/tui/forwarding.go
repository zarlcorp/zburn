package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/zburn/internal/namecheap"
)

// excludedDomains are org-owned domains that should never get catch-all forwarding.
var excludedDomains = []string{
	"zarlcorp.com",
	"zarl.dev",
}

// forwardingResultMsg carries the outcome of a catch-all forwarding batch.
type forwardingResultMsg struct {
	successes int
	failures  int
}

// forwardingSetter abstracts the SetForwarding call for testing.
type forwardingSetter interface {
	SetForwarding(ctx context.Context, domain string, rules []namecheap.ForwardingRule) error
}

// setupCatchAllForwarding sets wildcard forwarding on each domain, skipping excluded ones.
// It continues past individual failures and returns the count of successes and failures.
func setupCatchAllForwarding(ctx context.Context, setter forwardingSetter, domains []string, gmailAddress string) (successes, failures int) {
	excluded := buildExclusionSet(excludedDomains)
	rule := []namecheap.ForwardingRule{{Mailbox: "*", ForwardTo: gmailAddress}}

	for _, d := range domains {
		if excluded[d] {
			continue
		}
		if err := setter.SetForwarding(ctx, d, rule); err != nil {
			failures++
			continue
		}
		successes++
	}

	return successes, failures
}

func buildExclusionSet(domains []string) map[string]bool {
	m := make(map[string]bool, len(domains))
	for _, d := range domains {
		m[d] = true
	}
	return m
}

// forwardingCmd returns a tea.Cmd that runs catch-all forwarding setup in the background.
func forwardingCmd(cfg namecheap.Config, domains []string, gmailAddress string) tea.Cmd {
	return func() tea.Msg {
		c := namecheap.NewClient(cfg)
		ok, fail := setupCatchAllForwarding(context.Background(), c, domains, gmailAddress)
		return forwardingResultMsg{successes: ok, failures: fail}
	}
}

// forwardingFlash builds the flash text for a forwarding result.
func forwardingFlash(r forwardingResultMsg) string {
	if r.failures == 0 {
		return fmt.Sprintf("forwarding set up on %d domains", r.successes)
	}
	if r.successes == 0 {
		return fmt.Sprintf("forwarding: %d failed", r.failures)
	}
	return fmt.Sprintf("forwarding: %d ok, %d failed", r.successes, r.failures)
}
