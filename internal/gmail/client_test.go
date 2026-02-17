package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func b64(s string) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(s))
}

func TestListMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth header = %q, want Bearer test-token", got)
		}

		if got := r.URL.Query().Get("q"); got != "to:alice@example.com" {
			t.Errorf("query = %q, want to:alice@example.com", got)
		}

		json.NewEncoder(w).Encode(listResponse{
			Messages: []struct {
				ID string `json:"id"`
			}{
				{ID: "msg-1"},
				{ID: "msg-2"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	// override apiBase by using a custom URL in the request
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	msgs, err := c.ListMessages(context.Background(), "to:alice@example.com", 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
	if msgs[0].ID != "msg-1" {
		t.Errorf("msgs[0].ID = %q, want msg-1", msgs[0].ID)
	}
	if msgs[1].ID != "msg-2" {
		t.Errorf("msgs[1].ID = %q, want msg-2", msgs[1].ID)
	}
}

func TestListMessagesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listResponse{})
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	msgs, err := c.ListMessages(context.Background(), "to:nobody@example.com", 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	if len(msgs) != 0 {
		t.Errorf("got %d messages, want 0", len(msgs))
	}
}

func TestGetMessagePlainText(t *testing.T) {
	bodyText := "Your verification code is 123456"

	am := apiMessage{
		ID: "msg-abc",
	}
	am.Payload.MimeType = "text/plain"
	am.Payload.Headers = []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{
		{Name: "From", Value: "noreply@example.com"},
		{Name: "To", Value: "alice@example.com"},
		{Name: "Subject", Value: "Your 2FA Code"},
		{Name: "Date", Value: "Mon, 16 Feb 2026 10:00:00 -0500"},
	}
	am.Payload.Body.Data = b64(bodyText)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(am)
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	msg, err := c.GetMessage(context.Background(), "msg-abc")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	if msg.ID != "msg-abc" {
		t.Errorf("ID = %q, want msg-abc", msg.ID)
	}
	if msg.From != "noreply@example.com" {
		t.Errorf("From = %q, want noreply@example.com", msg.From)
	}
	if msg.To != "alice@example.com" {
		t.Errorf("To = %q, want alice@example.com", msg.To)
	}
	if msg.Subject != "Your 2FA Code" {
		t.Errorf("Subject = %q, want Your 2FA Code", msg.Subject)
	}
	if msg.Body != bodyText {
		t.Errorf("Body = %q, want %q", msg.Body, bodyText)
	}
	if msg.Date.IsZero() {
		t.Error("Date is zero")
	}
}

func TestGetMessageMultipart(t *testing.T) {
	plainBody := "plain text part"
	htmlBody := "<p>html part</p>"

	am := apiMessage{
		ID: "msg-multi",
	}
	am.Payload.MimeType = "multipart/alternative"
	am.Payload.Headers = []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{
		{Name: "Subject", Value: "Multipart Test"},
	}
	am.Payload.Parts = []apiPart{
		{
			MimeType: "text/plain",
			Body: struct {
				Data string `json:"data"`
			}{Data: b64(plainBody)},
		},
		{
			MimeType: "text/html",
			Body: struct {
				Data string `json:"data"`
			}{Data: b64(htmlBody)},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(am)
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	msg, err := c.GetMessage(context.Background(), "msg-multi")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	if msg.Body != plainBody {
		t.Errorf("Body = %q, want %q", msg.Body, plainBody)
	}
}

func TestGetMessageNestedMultipart(t *testing.T) {
	plainBody := "nested plain text"

	am := apiMessage{
		ID: "msg-nested",
	}
	am.Payload.MimeType = "multipart/mixed"
	am.Payload.Parts = []apiPart{
		{
			MimeType: "multipart/alternative",
			Parts: []apiPart{
				{
					MimeType: "text/plain",
					Body: struct {
						Data string `json:"data"`
					}{Data: b64(plainBody)},
				},
				{
					MimeType: "text/html",
					Body: struct {
						Data string `json:"data"`
					}{Data: b64("<p>nested html</p>")},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(am)
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	msg, err := c.GetMessage(context.Background(), "msg-nested")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	if msg.Body != plainBody {
		t.Errorf("Body = %q, want %q", msg.Body, plainBody)
	}
}

func TestGetMessageErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	_, err := c.GetMessage(context.Background(), "not-found")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestParseMIMEBody(t *testing.T) {
	boundary := "boundary123"
	body := "--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Hello from plain text\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>Hello from HTML</p>\r\n" +
		"--" + boundary + "--\r\n"

	contentType := "multipart/alternative; boundary=" + boundary

	result := ParseMIMEBody(contentType, body)
	if result != "Hello from plain text" {
		t.Errorf("ParseMIMEBody = %q, want %q", result, "Hello from plain text")
	}
}

func TestParseMIMEBodyNoPlainText(t *testing.T) {
	boundary := "boundary456"
	body := "--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>Only HTML</p>\r\n" +
		"--" + boundary + "--\r\n"

	contentType := "multipart/alternative; boundary=" + boundary

	result := ParseMIMEBody(contentType, body)
	if result != "" {
		t.Errorf("ParseMIMEBody = %q, want empty string", result)
	}
}

func TestParseMIMEBodyInvalidContentType(t *testing.T) {
	result := ParseMIMEBody("text/plain", "just text")
	if result != "" {
		t.Errorf("ParseMIMEBody = %q, want empty for non-multipart", result)
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		year  int
		month int
		day   int
	}{
		{"Mon, 16 Feb 2026 10:00:00 -0500", 2026, 2, 16},
		{"16 Feb 2026 10:00:00 -0500", 2026, 2, 16},
		{"2026-02-16T10:00:00Z", 2026, 2, 16},
	}

	for _, tt := range tests {
		d := parseDate(tt.input)
		if d.IsZero() {
			t.Errorf("parseDate(%q) returned zero time", tt.input)
			continue
		}
		if d.Year() != tt.year || int(d.Month()) != tt.month || d.Day() != tt.day {
			t.Errorf("parseDate(%q) = %v, want %d-%d-%d", tt.input, d, tt.year, tt.month, tt.day)
		}
	}
}

func TestParseDateUnknownFormat(t *testing.T) {
	d := parseDate("not a date")
	if !d.IsZero() {
		t.Errorf("parseDate(unknown) = %v, want zero", d)
	}
}

func TestGetProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth header = %q, want Bearer test-token", got)
		}

		if r.URL.Path != "/profile" {
			t.Errorf("path = %q, want /profile", r.URL.Path)
		}

		json.NewEncoder(w).Encode(profileResponse{
			EmailAddress: "user@gmail.com",
		})
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	email, err := c.GetProfile(context.Background())
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	if email != "user@gmail.com" {
		t.Errorf("email = %q, want user@gmail.com", email)
	}
}

func TestGetProfileErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("bad-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	_, err := c.GetProfile(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestGetProfileEmptyEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(profileResponse{EmailAddress: ""})
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	_, err := c.GetProfile(context.Background())
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestGetProfileInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.httpClient = srv.Client()

	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(srv.URL)

	_, err := c.GetProfile(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
