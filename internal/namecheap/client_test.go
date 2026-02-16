package namecheap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// canned XML responses

const getForwardingOK = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.getEmailForwarding</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.getEmailForwarding">
    <DomainDNSGetEmailForwardingResult Domain="example.com" IsOwner="true">
      <Forward mailbox="alice">shared@gmail.com</Forward>
      <Forward mailbox="bob">shared@gmail.com</Forward>
    </DomainDNSGetEmailForwardingResult>
  </CommandResponse>
  <Server>SERVER</Server>
  <GMTTimeDifference>--5:00</GMTTimeDifference>
  <ExecutionTime>0.5</ExecutionTime>
</ApiResponse>`

const getForwardingEmpty = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.dns.getEmailForwarding">
    <DomainDNSGetEmailForwardingResult Domain="example.com" IsOwner="true">
    </DomainDNSGetEmailForwardingResult>
  </CommandResponse>
</ApiResponse>`

const setForwardingOK = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.dns.setEmailForwarding">
    <DomainDNSSetEmailForwardingResult Domain="example.com" IsSuccess="true" />
  </CommandResponse>
</ApiResponse>`

const apiErrorResponse = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors>
    <Error Number="2019166">Domain not found</Error>
  </Errors>
  <CommandResponse />
</ApiResponse>`

const apiMultiErrorResponse = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors>
    <Error Number="1010101">Invalid API key</Error>
    <Error Number="1010102">IP not whitelisted</Error>
  </Errors>
  <CommandResponse />
</ApiResponse>`

const listDomainsOK = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <RequestedCommand>namecheap.domains.getList</RequestedCommand>
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="123" Name="example.com" User="testuser" Created="01/01/2020" Expires="01/01/2030" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="ENABLED" />
      <Domain ID="456" Name="other.io" User="testuser" Created="06/15/2021" Expires="06/15/2031" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="ENABLED" />
    </DomainGetListResult>
  </CommandResponse>
  <Server>SERVER</Server>
  <GMTTimeDifference>--5:00</GMTTimeDifference>
  <ExecutionTime>0.3</ExecutionTime>
</ApiResponse>`

const listDomainsEmpty = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
    </DomainGetListResult>
  </CommandResponse>
</ApiResponse>`

func testConfig() Config {
	return Config{
		Username: "testuser",
		APIKey:   "testkey",
	}
}

func newTestClient(url string) *Client {
	c := NewClient(testConfig())
	c.baseURL = url
	c.clientIP = "1.2.3.4" // pre-cache to avoid real DNS lookups
	return c
}

func TestGetForwarding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "Command", "namecheap.domains.dns.getEmailForwarding")
		assertQueryParam(t, r, "SLD", "example")
		assertQueryParam(t, r, "TLD", "com")
		assertQueryParam(t, r, "DomainName", "example.com")
		assertQueryParam(t, r, "ApiUser", "testuser")
		assertQueryParam(t, r, "ApiKey", "testkey")
		assertQueryParam(t, r, "ClientIp", "1.2.3.4")

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(getForwardingOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	rules, err := c.GetForwarding(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("get forwarding: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("rules count: got %d, want 2", len(rules))
	}

	want := []ForwardingRule{
		{Mailbox: "alice", ForwardTo: "shared@gmail.com"},
		{Mailbox: "bob", ForwardTo: "shared@gmail.com"},
	}
	for i, r := range rules {
		if r != want[i] {
			t.Errorf("rule[%d]: got %+v, want %+v", i, r, want[i])
		}
	}
}

func TestGetForwardingEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(getForwardingEmpty))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	rules, err := c.GetForwarding(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("get forwarding: %v", err)
	}

	if len(rules) != 0 {
		t.Fatalf("rules count: got %d, want 0", len(rules))
	}
}

func TestSetForwarding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "Command", "namecheap.domains.dns.setEmailForwarding")
		assertQueryParam(t, r, "SLD", "example")
		assertQueryParam(t, r, "TLD", "com")
		assertQueryParam(t, r, "MailBox1", "alice")
		assertQueryParam(t, r, "ForwardTo1", "shared@gmail.com")
		assertQueryParam(t, r, "MailBox2", "bob")
		assertQueryParam(t, r, "ForwardTo2", "other@gmail.com")

		w.Write([]byte(setForwardingOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.SetForwarding(context.Background(), "example.com", []ForwardingRule{
		{Mailbox: "alice", ForwardTo: "shared@gmail.com"},
		{Mailbox: "bob", ForwardTo: "other@gmail.com"},
	})
	if err != nil {
		t.Fatalf("set forwarding: %v", err)
	}
}

func TestAddForwardingPreservesExisting(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("Command")
		callCount++

		switch cmd {
		case "namecheap.domains.dns.getEmailForwarding":
			w.Write([]byte(getForwardingOK))
		case "namecheap.domains.dns.setEmailForwarding":
			// verify all three rules are present
			assertQueryParam(t, r, "MailBox1", "alice")
			assertQueryParam(t, r, "ForwardTo1", "shared@gmail.com")
			assertQueryParam(t, r, "MailBox2", "bob")
			assertQueryParam(t, r, "ForwardTo2", "shared@gmail.com")
			assertQueryParam(t, r, "MailBox3", "charlie")
			assertQueryParam(t, r, "ForwardTo3", "other@gmail.com")
			w.Write([]byte(setForwardingOK))
		default:
			t.Errorf("unexpected command: %s", cmd)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.AddForwarding(context.Background(), "example.com", "charlie", "other@gmail.com")
	if err != nil {
		t.Fatalf("add forwarding: %v", err)
	}

	if callCount != 2 {
		t.Errorf("call count: got %d, want 2 (get + set)", callCount)
	}
}

func TestAddForwardingToEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("Command")

		switch cmd {
		case "namecheap.domains.dns.getEmailForwarding":
			w.Write([]byte(getForwardingEmpty))
		case "namecheap.domains.dns.setEmailForwarding":
			assertQueryParam(t, r, "MailBox1", "first")
			assertQueryParam(t, r, "ForwardTo1", "dest@gmail.com")

			// verify no second rule
			if v := r.URL.Query().Get("MailBox2"); v != "" {
				t.Errorf("unexpected MailBox2: %s", v)
			}

			w.Write([]byte(setForwardingOK))
		default:
			t.Errorf("unexpected command: %s", cmd)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.AddForwarding(context.Background(), "example.com", "first", "dest@gmail.com")
	if err != nil {
		t.Fatalf("add forwarding: %v", err)
	}
}

func TestRemoveForwarding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("Command")

		switch cmd {
		case "namecheap.domains.dns.getEmailForwarding":
			w.Write([]byte(getForwardingOK))
		case "namecheap.domains.dns.setEmailForwarding":
			// alice removed, only bob remains
			assertQueryParam(t, r, "MailBox1", "bob")
			assertQueryParam(t, r, "ForwardTo1", "shared@gmail.com")

			if v := r.URL.Query().Get("MailBox2"); v != "" {
				t.Errorf("unexpected MailBox2: %s", v)
			}

			w.Write([]byte(setForwardingOK))
		default:
			t.Errorf("unexpected command: %s", cmd)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.RemoveForwarding(context.Background(), "example.com", "alice")
	if err != nil {
		t.Fatalf("remove forwarding: %v", err)
	}
}

func TestRemoveForwardingNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(getForwardingOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.RemoveForwarding(context.Background(), "example.com", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mailbox")
	}
	if !strings.Contains(err.Error(), `mailbox "nonexistent" not found`) {
		t.Errorf("error message: got %q, want contains mailbox not found", err.Error())
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(apiErrorResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetForwarding(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Domain not found") {
		t.Errorf("error: got %q, want contains 'Domain not found'", err.Error())
	}
}

func TestAPIMultipleErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(apiMultiErrorResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetForwarding(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("error: got %q, want contains 'Invalid API key'", err.Error())
	}
	if !strings.Contains(err.Error(), "IP not whitelisted") {
		t.Errorf("error: got %q, want contains 'IP not whitelisted'", err.Error())
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetForwarding(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error: got %q, want contains 'unexpected status 500'", err.Error())
	}
}

func TestNetworkError(t *testing.T) {
	// point at a closed server
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetForwarding(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestMalformedXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("this is not xml"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetForwarding(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error for malformed XML")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error: got %q, want contains 'parse response'", err.Error())
	}
}

func TestInvalidDomain(t *testing.T) {
	c := newTestClient("http://unused")

	tests := []string{
		"nodot",
		".leading",
		"trailing.",
		"",
	}

	for _, d := range tests {
		_, err := c.GetForwarding(context.Background(), d)
		if err == nil {
			t.Errorf("expected error for domain %q", d)
		}
	}
}

func TestSplitDomain(t *testing.T) {
	tests := []struct {
		domain  string
		sld     string
		tld     string
		wantErr bool
	}{
		{"example.com", "example", "com", false},
		{"my.co.uk", "my", "co.uk", false},
		{"sub.example.org", "sub", "example.org", false},
		{"nodot", "", "", true},
		{"", "", "", true},
		{".com", "", "", true},
		{"example.", "", "", true},
	}

	for _, tt := range tests {
		sld, tld, err := splitDomain(tt.domain)
		if tt.wantErr {
			if err == nil {
				t.Errorf("splitDomain(%q): want error, got nil", tt.domain)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitDomain(%q): %v", tt.domain, err)
			continue
		}
		if sld != tt.sld || tld != tt.tld {
			t.Errorf("splitDomain(%q): got (%q, %q), want (%q, %q)", tt.domain, sld, tld, tt.sld, tt.tld)
		}
	}
}

func TestSetForwardingAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(apiErrorResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	err := c.SetForwarding(context.Background(), "example.com", []ForwardingRule{
		{Mailbox: "test", ForwardTo: "dest@gmail.com"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Domain not found") {
		t.Errorf("error: got %q, want contains 'Domain not found'", err.Error())
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(getForwardingOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.GetForwarding(ctx, "example.com")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestListDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "Command", "namecheap.domains.getList")
		assertQueryParam(t, r, "ApiUser", "testuser")
		assertQueryParam(t, r, "UserName", "testuser")
		assertQueryParam(t, r, "ApiKey", "testkey")
		assertQueryParam(t, r, "ClientIp", "1.2.3.4")

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(listDomainsOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	domains, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}

	if len(domains) != 2 {
		t.Fatalf("domains count: got %d, want 2", len(domains))
	}

	want := []string{"example.com", "other.io"}
	for i, d := range domains {
		if d != want[i] {
			t.Errorf("domains[%d]: got %q, want %q", i, d, want[i])
		}
	}
}

func TestListDomainsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(listDomainsEmpty))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	domains, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}

	if len(domains) != 0 {
		t.Fatalf("domains count: got %d, want 0", len(domains))
	}
}

func TestListDomainsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(apiErrorResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListDomains(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Domain not found") {
		t.Errorf("error: got %q, want contains 'Domain not found'", err.Error())
	}
}

func TestClientIPCaching(t *testing.T) {
	c := NewClient(testConfig())
	c.clientIP = "10.20.30.40"

	// resolveIP should return the cached value
	ip, err := c.resolveIP(context.Background())
	if err != nil {
		t.Fatalf("resolve ip: %v", err)
	}
	if ip != "10.20.30.40" {
		t.Errorf("ip: got %q, want %q", ip, "10.20.30.40")
	}
}

func TestBaseParamsSendsUsernameAsBoth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "ApiUser", "testuser")
		assertQueryParam(t, r, "UserName", "testuser")
		w.Write([]byte(listDomainsOK))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
}

func assertQueryParam(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	got := r.URL.Query().Get(key)
	if got != want {
		t.Errorf("query param %s: got %q, want %q", key, got, want)
	}
}
