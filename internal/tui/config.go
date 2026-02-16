package tui

import (
	"encoding/json"

	"github.com/zarlcorp/zburn/internal/gmail"
	"github.com/zarlcorp/zburn/internal/namecheap"
	"github.com/zarlcorp/zburn/internal/twilio"
)

// configEnvelope wraps a JSON-encoded config value so we can store
// heterogeneous config types in a single zstore collection.
type configEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// NamecheapSettings holds Namecheap credentials and cached domain list.
type NamecheapSettings struct {
	Username      string   `json:"username"`
	APIKey        string   `json:"api_key"`
	CachedDomains []string `json:"cached_domains"`
}

// GmailSettings holds Gmail OAuth2 credentials and tokens.
type GmailSettings struct {
	ClientID     string      `json:"client_id"`
	ClientSecret string      `json:"client_secret"`
	Token        *gmail.Token `json:"token,omitempty"`
}

// TwilioSettings holds Twilio credentials and preferred countries.
type TwilioSettings struct {
	AccountSID         string   `json:"account_sid"`
	AuthToken          string   `json:"auth_token"`
	PreferredCountries []string `json:"preferred_countries"`
}

func (s NamecheapSettings) Configured() bool {
	return s.Username != "" && s.APIKey != ""
}

func (s GmailSettings) Configured() bool {
	return s.Token != nil && s.Token.RefreshToken != ""
}

func (s TwilioSettings) Configured() bool {
	return s.AccountSID != "" && s.AuthToken != ""
}

// NamecheapConfig converts settings to a namecheap.Config for API use.
func (s NamecheapSettings) NamecheapConfig() namecheap.Config {
	return namecheap.Config{
		Username: s.Username,
		APIKey:   s.APIKey,
	}
}

// OAuthConfig converts settings to a gmail.OAuthConfig for API use.
func (s GmailSettings) OAuthConfig() gmail.OAuthConfig {
	return gmail.OAuthConfig{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
	}
}

// TwilioConfig converts settings to a twilio.Config for API use.
func (s TwilioSettings) TwilioConfig() twilio.Config {
	return twilio.Config{
		AccountSID: s.AccountSID,
		AuthToken:  s.AuthToken,
	}
}
