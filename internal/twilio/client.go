// Package twilio provides a client for provisioning phone numbers and reading
// incoming SMS messages via the Twilio REST API.
package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds Twilio API credentials.
type Config struct {
	AccountSID string
	AuthToken  string
}

// AvailableNumber represents a phone number available for purchase.
type AvailableNumber struct {
	PhoneNumber  string
	FriendlyName string
	Capabilities struct {
		SMS   bool
		Voice bool
	}
}

// PhoneNumber represents a provisioned incoming phone number.
type PhoneNumber struct {
	SID          string
	PhoneNumber  string
	FriendlyName string
}

// SMSMessage represents an SMS message.
type SMSMessage struct {
	SID      string
	From     string
	To       string
	Body     string
	DateSent time.Time
	Status   string
}

// Client communicates with the Twilio REST API.
type Client struct {
	accountSID string
	authToken  string
	baseURL    string
	http       *http.Client
}

// NewClient creates a Twilio client with the given credentials.
func NewClient(cfg Config) *Client {
	return &Client{
		accountSID: cfg.AccountSID,
		authToken:  cfg.AuthToken,
		baseURL:    "https://api.twilio.com/2010-04-01/Accounts/" + cfg.AccountSID,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
}

// SearchNumbers returns available local phone numbers for a country.
// country is an ISO country code (e.g. "GB", "US").
func (c *Client) SearchNumbers(ctx context.Context, country string) ([]AvailableNumber, error) {
	u := c.baseURL + "/AvailablePhoneNumbers/" + country + "/Local.json"

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("search numbers: %w", err)
	}

	var resp availableNumbersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("search numbers: unmarshal: %w", err)
	}

	numbers := make([]AvailableNumber, len(resp.AvailablePhoneNumbers))
	for i, n := range resp.AvailablePhoneNumbers {
		numbers[i] = AvailableNumber{
			PhoneNumber:  n.PhoneNumber,
			FriendlyName: n.FriendlyName,
		}
		numbers[i].Capabilities.SMS = n.Capabilities.SMS
		numbers[i].Capabilities.Voice = n.Capabilities.Voice
	}

	return numbers, nil
}

// BuyNumber purchases a phone number and returns the provisioned details.
func (c *Client) BuyNumber(ctx context.Context, phoneNumber string) (*PhoneNumber, error) {
	u := c.baseURL + "/IncomingPhoneNumbers.json"

	form := url.Values{}
	form.Set("PhoneNumber", phoneNumber)

	body, err := c.doPost(ctx, u, form)
	if err != nil {
		return nil, fmt.Errorf("buy number: %w", err)
	}

	var resp phoneNumberResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("buy number: unmarshal: %w", err)
	}

	return &PhoneNumber{
		SID:          resp.SID,
		PhoneNumber:  resp.PhoneNumber,
		FriendlyName: resp.FriendlyName,
	}, nil
}

// ReleaseNumber releases a provisioned phone number back to Twilio.
func (c *Client) ReleaseNumber(ctx context.Context, numberSID string) error {
	u := c.baseURL + "/IncomingPhoneNumbers/" + numberSID + ".json"

	if err := c.doDelete(ctx, u); err != nil {
		return fmt.Errorf("release number: %w", err)
	}

	return nil
}

// ListMessages returns recent SMS messages sent to the given number.
// Results are ordered by date descending (most recent first).
func (c *Client) ListMessages(ctx context.Context, to string, limit int) ([]SMSMessage, error) {
	u := c.baseURL + "/Messages.json?To=" + url.QueryEscape(to) + "&PageSize=" + fmt.Sprintf("%d", limit)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	var resp messagesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("list messages: unmarshal: %w", err)
	}

	messages := make([]SMSMessage, len(resp.Messages))
	for i, m := range resp.Messages {
		t, _ := time.Parse(time.RFC1123Z, m.DateSent)
		messages[i] = SMSMessage{
			SID:      m.SID,
			From:     m.From,
			To:       m.To,
			Body:     m.Body,
			DateSent: t,
			Status:   m.Status,
		}
	}

	return messages, nil
}

func (c *Client) doGet(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.do(req)
}

func (c *Client) doPost(ctx context.Context, u string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.do(req)
}

func (c *Client) doDelete(ctx context.Context, u string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.doRaw(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return body, nil
}

func (c *Client) doRaw(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.accountSID, c.authToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var apiErr apiErrorResponse
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
			return nil, &Error{
				StatusCode: resp.StatusCode,
				Code:       apiErr.Code,
				Message:    apiErr.Message,
			}
		}

		return nil, &Error{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	return resp, nil
}

// Error represents a Twilio API error.
type Error struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *Error) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("twilio: %s (code %d, status %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("twilio: %s (status %d)", e.Message, e.StatusCode)
}

// json wire types for API responses

type availableNumbersResponse struct {
	AvailablePhoneNumbers []struct {
		PhoneNumber  string `json:"phone_number"`
		FriendlyName string `json:"friendly_name"`
		Capabilities struct {
			SMS   bool `json:"sms"`
			Voice bool `json:"voice"`
		} `json:"capabilities"`
	} `json:"available_phone_numbers"`
}

type phoneNumberResponse struct {
	SID          string `json:"sid"`
	PhoneNumber  string `json:"phone_number"`
	FriendlyName string `json:"friendly_name"`
}

type messagesResponse struct {
	Messages []struct {
		SID      string `json:"sid"`
		From     string `json:"from"`
		To       string `json:"to"`
		Body     string `json:"body"`
		DateSent string `json:"date_sent"`
		Status   string `json:"status"`
	} `json:"messages"`
}

type apiErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}
