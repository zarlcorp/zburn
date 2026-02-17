package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/namecheap"
)

// domainForwardingStatus holds the forwarding state for a single domain.
type domainForwardingStatus struct {
	domain   string
	excluded bool
	rules    []namecheap.ForwardingRule
	err      error
}

// forwardingStatusMsg carries fetched forwarding status for all domains.
type forwardingStatusMsg struct {
	statuses []domainForwardingStatus
}

// forwardingGetter abstracts the GetForwarding call for testing.
type forwardingGetter interface {
	GetForwarding(ctx context.Context, domain string) ([]namecheap.ForwardingRule, error)
}

// forwardingModel displays per-domain forwarding status.
type forwardingModel struct {
	statuses []domainForwardingStatus
	loading  bool
	warning  string // auth warning message
}

func newForwardingModel(nc NamecheapSettings, gm GmailSettings) forwardingModel {
	m := forwardingModel{}

	switch {
	case !nc.Configured() && !gm.Configured():
		m.warning = "configure namecheap and gmail to enable forwarding"
	case nc.Configured() && !gm.Configured():
		m.warning = "gmail not connected — forwarding inactive"
	case !nc.Configured() && gm.Configured():
		m.warning = "namecheap not connected — no domains"
	}

	return m
}

func (m forwardingModel) Init() tea.Cmd {
	return nil
}

func (m forwardingModel) Update(msg tea.Msg) (forwardingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, zstyle.KeyQuit) {
			return m, tea.Quit
		}

		if key.Matches(msg, zstyle.KeyBack) {
			return m, func() tea.Msg { return navigateMsg{view: viewSettings} }
		}

	case forwardingStatusMsg:
		m.loading = false
		m.statuses = msg.statuses
		return m, nil
	}

	return m, nil
}

func (m forwardingModel) View() string {
	title := zstyle.Title.Render("forwarding")
	s := fmt.Sprintf("\n  %s\n\n", title)

	if m.warning != "" {
		s += "  " + zstyle.StatusWarn.Render(m.warning) + "\n\n"
		s += "  " + zstyle.MutedText.Render("esc back  q quit") + "\n"
		return s
	}

	if m.loading {
		s += "  " + zstyle.MutedText.Render("loading...") + "\n\n"
		s += "  " + zstyle.MutedText.Render("esc back  q quit") + "\n"
		return s
	}

	if len(m.statuses) == 0 {
		s += "  " + zstyle.MutedText.Render("no domains") + "\n\n"
		s += "  " + zstyle.MutedText.Render("esc back  q quit") + "\n"
		return s
	}

	for _, st := range m.statuses {
		s += formatDomainStatus(st)
	}

	s += "\n  " + zstyle.MutedText.Render("esc back  q quit") + "\n"
	return s
}

func formatDomainStatus(st domainForwardingStatus) string {
	domain := fmt.Sprintf("  %-22s", st.domain)

	if st.excluded {
		return domain + zstyle.MutedText.Render("excluded") + "\n"
	}

	if st.err != nil {
		return domain + zstyle.StatusErr.Render("error") + "\n"
	}

	target := catchAllTarget(st.rules)
	if target != "" {
		return domain + zstyle.MutedText.Render("* → "+target) + "\n"
	}

	return domain + zstyle.StatusWarn.Render("not configured") + "\n"
}

// catchAllTarget returns the forwarding address for the wildcard mailbox, if any.
func catchAllTarget(rules []namecheap.ForwardingRule) string {
	for _, r := range rules {
		if r.Mailbox == "*" {
			return r.ForwardTo
		}
	}
	return ""
}

// fetchForwardingStatusCmd returns a tea.Cmd that fetches forwarding status for all domains.
func fetchForwardingStatusCmd(cfg namecheap.Config, domains []string) tea.Cmd {
	return func() tea.Msg {
		c := namecheap.NewClient(cfg)
		return fetchForwardingStatus(context.Background(), c, domains)
	}
}

// fetchForwardingStatus queries forwarding status for each domain.
func fetchForwardingStatus(ctx context.Context, getter forwardingGetter, domains []string) forwardingStatusMsg {
	excluded := buildExclusionSet(excludedDomains)
	statuses := make([]domainForwardingStatus, len(domains))

	for i, d := range domains {
		st := domainForwardingStatus{domain: d}
		if excluded[d] {
			st.excluded = true
		} else {
			rules, err := getter.GetForwarding(ctx, d)
			st.rules = rules
			st.err = err
		}
		statuses[i] = st
	}

	return forwardingStatusMsg{statuses: statuses}
}
