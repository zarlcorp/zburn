// Package gmail provides OAuth2 authentication and API access to Gmail.
// it handles the browser-based consent flow and token refresh for TUI apps.
package gmail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

const (
	authEndpoint        = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenEndpoint = "https://oauth2.googleapis.com/token"
	gmailScope          = "https://www.googleapis.com/auth/gmail.readonly"
)

// tokenEndpoint is a var so tests can point it at httptest servers.
var tokenEndpoint = defaultTokenEndpoint

// setTokenEndpoint overrides the token endpoint (for testing).
func setTokenEndpoint(u string) { tokenEndpoint = u }

// OAuthConfig holds Google OAuth2 client credentials.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// Token holds OAuth2 access and refresh tokens.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

// tokenResponse maps the JSON returned by Google's token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Authenticate runs the OAuth2 authorization code flow.
// it starts a temporary localhost server, opens the browser to Google's consent
// screen, captures the callback, and exchanges the code for tokens.
func Authenticate(ctx context.Context, cfg OAuthConfig) (*Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("authenticate: generate state: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("authenticate: listen: %w", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	authURL := buildAuthURL(cfg.ClientID, redirectURI, state)

	// channel receives the auth code or an error from the callback handler
	type result struct {
		code string
		err  error
	}
	ch := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if s := r.URL.Query().Get("state"); s != state {
			ch <- result{err: fmt.Errorf("state mismatch: got %q", s)}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			ch <- result{err: fmt.Errorf("oauth error: %s", errMsg)}
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			ch <- result{err: fmt.Errorf("no code in callback")}
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, "authorization complete — you can close this tab.")
		ch <- result{code: code}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Shutdown(context.Background())

	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("authenticate: open browser: %w", err)
	}

	var code string
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("authenticate: %w", ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("authenticate: %w", res.err)
		}
		code = res.code
	}

	return exchangeCode(ctx, cfg, code, redirectURI)
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(ctx context.Context, cfg OAuthConfig, refreshToken string) (*Token, error) {
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("refresh token: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token: status %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("refresh token: decode response: %w", err)
	}

	tok := &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: refreshToken, // Google may not return a new refresh token
		Expiry:       time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}
	if tr.RefreshToken != "" {
		tok.RefreshToken = tr.RefreshToken
	}

	return tok, nil
}

// buildAuthURL constructs the Google OAuth2 authorization URL.
func buildAuthURL(clientID, redirectURI, state string) string {
	v := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {gmailScope},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return authEndpoint + "?" + v.Encode()
}

func exchangeCode(ctx context.Context, cfg OAuthConfig, code, redirectURI string) (*Token, error) {
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("exchange code: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange code: status %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("exchange code: decode response: %w", err)
	}

	return &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	return exec.Command("open", url).Start()
}
