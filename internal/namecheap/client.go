// Package namecheap provides a client for managing email forwarding rules
// via the Namecheap API.
package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultBaseURL = "https://api.namecheap.com/xml.response"

// ForwardingRule maps a mailbox to a forwarding address.
type ForwardingRule struct {
	Mailbox   string // e.g. "john.doe" (the part before @)
	ForwardTo string // e.g. "shared@gmail.com"
}

// Config holds credentials for the Namecheap API.
type Config struct {
	Username string
	APIKey   string
}

// Client communicates with the Namecheap API.
type Client struct {
	cfg      Config
	baseURL  string
	http     *http.Client
	clientIP string // cached after first detection
}

// NewClient creates a Namecheap API client.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		baseURL: defaultBaseURL,
		http:    http.DefaultClient,
	}
}

// GetForwarding returns the current email forwarding rules for a domain.
func (c *Client) GetForwarding(ctx context.Context, domain string) ([]ForwardingRule, error) {
	sld, tld, err := splitDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("get forwarding: %w", err)
	}

	params, err := c.baseParams(ctx, "namecheap.domains.dns.getEmailForwarding")
	if err != nil {
		return nil, fmt.Errorf("get forwarding: %w", err)
	}
	params.Set("DomainName", domain)
	params.Set("SLD", sld)
	params.Set("TLD", tld)

	body, err := c.do(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get forwarding: %w", err)
	}

	var resp apiResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("get forwarding: parse response: %w", err)
	}

	if err := resp.err(); err != nil {
		return nil, fmt.Errorf("get forwarding: %w", err)
	}

	var result getForwardingResult
	if err := xml.Unmarshal(resp.CommandResponse.Raw, &result); err != nil {
		return nil, fmt.Errorf("get forwarding: parse result: %w", err)
	}

	rules := make([]ForwardingRule, len(result.Forwards))
	for i, f := range result.Forwards {
		rules[i] = ForwardingRule{
			Mailbox:   f.MailBox,
			ForwardTo: f.ForwardTo,
		}
	}

	return rules, nil
}

// SetForwarding replaces all email forwarding rules for a domain.
// This is a destructive operation — any existing rules not included will be removed.
func (c *Client) SetForwarding(ctx context.Context, domain string, rules []ForwardingRule) error {
	sld, tld, err := splitDomain(domain)
	if err != nil {
		return fmt.Errorf("set forwarding: %w", err)
	}

	params, err := c.baseParams(ctx, "namecheap.domains.dns.setEmailForwarding")
	if err != nil {
		return fmt.Errorf("set forwarding: %w", err)
	}
	params.Set("DomainName", domain)
	params.Set("SLD", sld)
	params.Set("TLD", tld)

	for i, r := range rules {
		n := strconv.Itoa(i + 1)
		params.Set("MailBox"+n, r.Mailbox)
		params.Set("ForwardTo"+n, r.ForwardTo)
	}

	body, err := c.do(ctx, params)
	if err != nil {
		return fmt.Errorf("set forwarding: %w", err)
	}

	var resp apiResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("set forwarding: parse response: %w", err)
	}

	if err := resp.err(); err != nil {
		return fmt.Errorf("set forwarding: %w", err)
	}

	return nil
}

// AddForwarding adds a forwarding rule without removing existing ones.
// It fetches current rules, appends the new one, and sets all rules.
func (c *Client) AddForwarding(ctx context.Context, domain, mailbox, forwardTo string) error {
	rules, err := c.GetForwarding(ctx, domain)
	if err != nil {
		return fmt.Errorf("add forwarding: %w", err)
	}

	rules = append(rules, ForwardingRule{
		Mailbox:   mailbox,
		ForwardTo: forwardTo,
	})

	if err := c.SetForwarding(ctx, domain, rules); err != nil {
		return fmt.Errorf("add forwarding: %w", err)
	}

	return nil
}

// RemoveForwarding removes a forwarding rule by mailbox name.
// It fetches current rules, removes the matching one, and sets the remaining rules.
func (c *Client) RemoveForwarding(ctx context.Context, domain, mailbox string) error {
	rules, err := c.GetForwarding(ctx, domain)
	if err != nil {
		return fmt.Errorf("remove forwarding: %w", err)
	}

	filtered := rules[:0]
	for _, r := range rules {
		if r.Mailbox != mailbox {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == len(rules) {
		return fmt.Errorf("remove forwarding: mailbox %q not found", mailbox)
	}

	if err := c.SetForwarding(ctx, domain, filtered); err != nil {
		return fmt.Errorf("remove forwarding: %w", err)
	}

	return nil
}

// ListDomains returns domain names associated with the account.
func (c *Client) ListDomains(ctx context.Context) ([]string, error) {
	params, err := c.baseParams(ctx, "namecheap.domains.getList")
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}

	body, err := c.do(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}

	var resp apiResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("list domains: parse response: %w", err)
	}

	if err := resp.err(); err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}

	var result domainsGetListResult
	if err := xml.Unmarshal(resp.CommandResponse.Raw, &result); err != nil {
		return nil, fmt.Errorf("list domains: parse result: %w", err)
	}

	domains := make([]string, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = d.Name
	}

	return domains, nil
}

func (c *Client) baseParams(ctx context.Context, command string) (url.Values, error) {
	ip, err := c.resolveIP(ctx)
	if err != nil {
		return nil, err
	}

	return url.Values{
		"ApiUser":  {c.cfg.Username},
		"ApiKey":   {c.cfg.APIKey},
		"UserName": {c.cfg.Username},
		"ClientIp": {ip},
		"Command":  {command},
	}, nil
}

// resolveIP returns the cached client IP, detecting it on first call.
func (c *Client) resolveIP(ctx context.Context) (string, error) {
	if c.clientIP != "" {
		return c.clientIP, nil
	}

	ip, err := detectPublicIP(ctx)
	if err != nil {
		return "", err
	}

	c.clientIP = ip
	return ip, nil
}

func (c *Client) do(ctx context.Context, params url.Values) ([]byte, error) {
	u := c.baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return body, nil
}

func splitDomain(domain string) (sld, tld string, err error) {
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid domain %q", domain)
	}
	return parts[0], parts[1], nil
}

// xml response types

type apiResponse struct {
	XMLName         xml.Name        `xml:"ApiResponse"`
	Status          string          `xml:"Status,attr"`
	Errors          apiErrors       `xml:"Errors"`
	CommandResponse commandResponse `xml:"CommandResponse"`
}

func (r *apiResponse) err() error {
	if r.Status == "OK" {
		return nil
	}

	if len(r.Errors.Items) > 0 {
		msgs := make([]string, len(r.Errors.Items))
		for i, e := range r.Errors.Items {
			msgs[i] = e.Message
		}
		return fmt.Errorf("api: %s", strings.Join(msgs, "; "))
	}

	return fmt.Errorf("api: status %s", r.Status)
}

type apiErrors struct {
	Items []apiError `xml:"Error"`
}

type apiError struct {
	Number  string `xml:"Number,attr"`
	Message string `xml:",chardata"`
}

type commandResponse struct {
	Raw []byte `xml:",innerxml"`
}

type getForwardingResult struct {
	XMLName  xml.Name          `xml:"DomainDNSGetEmailForwardingResult"`
	Forwards []forwardingEntry `xml:"Forward"`
}

type forwardingEntry struct {
	MailBox   string `xml:"mailbox,attr"`
	ForwardTo string `xml:",chardata"`
}

type domainsGetListResult struct {
	XMLName xml.Name      `xml:"DomainGetListResult"`
	Domains []domainEntry `xml:"Domain"`
}

type domainEntry struct {
	Name string `xml:"Name,attr"`
}
