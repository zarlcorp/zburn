package gmail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestBuildAuthURL(t *testing.T) {
	u := buildAuthURL("my-client-id", "http://localhost:9999/callback", "test-state")

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != authEndpoint {
		t.Errorf("endpoint = %q, want %q", got, authEndpoint)
	}

	tests := []struct {
		param string
		want  string
	}{
		{"client_id", "my-client-id"},
		{"redirect_uri", "http://localhost:9999/callback"},
		{"response_type", "code"},
		{"scope", gmailScope},
		{"access_type", "offline"},
		{"prompt", "consent"},
		{"state", "test-state"},
	}

	q := parsed.Query()
	for _, tt := range tests {
		if got := q.Get(tt.param); got != tt.want {
			t.Errorf("param %q = %q, want %q", tt.param, got, tt.want)
		}
	}
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", got)
		}
		if got := r.Form.Get("code"); got != "test-code" {
			t.Errorf("code = %q, want test-code", got)
		}
		if got := r.Form.Get("client_id"); got != "cid" {
			t.Errorf("client_id = %q, want cid", got)
		}
		if got := r.Form.Get("client_secret"); got != "csecret" {
			t.Errorf("client_secret = %q, want csecret", got)
		}

		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "access-123",
			RefreshToken: "refresh-456",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer srv.Close()

	// temporarily override the token endpoint
	origEndpoint := tokenEndpoint
	defer func() { setTokenEndpoint(origEndpoint) }()
	setTokenEndpoint(srv.URL)

	tok, err := exchangeCode(context.Background(), OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csecret",
	}, "test-code", "http://localhost:9999/callback")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}

	if tok.AccessToken != "access-123" {
		t.Errorf("access token = %q, want access-123", tok.AccessToken)
	}
	if tok.RefreshToken != "refresh-456" {
		t.Errorf("refresh token = %q, want refresh-456", tok.RefreshToken)
	}
	if tok.Expiry.Before(time.Now()) {
		t.Errorf("expiry %v is in the past", tok.Expiry)
	}
}

func TestRefreshToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", got)
		}
		if got := r.Form.Get("refresh_token"); got != "old-refresh" {
			t.Errorf("refresh_token = %q, want old-refresh", got)
		}

		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "new-access",
			ExpiresIn:   7200,
		})
	}))
	defer srv.Close()

	origEndpoint := tokenEndpoint
	defer func() { setTokenEndpoint(origEndpoint) }()
	setTokenEndpoint(srv.URL)

	tok, err := RefreshToken(context.Background(), OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csecret",
	}, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if tok.AccessToken != "new-access" {
		t.Errorf("access token = %q, want new-access", tok.AccessToken)
	}
	// refresh token preserved when Google doesn't return a new one
	if tok.RefreshToken != "old-refresh" {
		t.Errorf("refresh token = %q, want old-refresh", tok.RefreshToken)
	}
}

func TestRefreshTokenReturnsNewRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    7200,
		})
	}))
	defer srv.Close()

	origEndpoint := tokenEndpoint
	defer func() { setTokenEndpoint(origEndpoint) }()
	setTokenEndpoint(srv.URL)

	tok, err := RefreshToken(context.Background(), OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csecret",
	}, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if tok.RefreshToken != "new-refresh" {
		t.Errorf("refresh token = %q, want new-refresh", tok.RefreshToken)
	}
}

func TestRefreshTokenErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origEndpoint := tokenEndpoint
	defer func() { setTokenEndpoint(origEndpoint) }()
	setTokenEndpoint(srv.URL)

	_, err := RefreshToken(context.Background(), OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csecret",
	}, "bad-refresh")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
