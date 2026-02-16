package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// apiBase is a var so tests can point it at httptest servers.
var apiBase = "https://gmail.googleapis.com/gmail/v1/users/me"

func setAPIBase(u string) { apiBase = u }

// Message holds a parsed Gmail message.
type Message struct {
	ID      string
	From    string
	To      string
	Subject string
	Date    time.Time
	Body    string // plain text content
}

// Client accesses the Gmail API using a bearer token.
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// NewClient creates a Gmail API client with the given access token.
func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  http.DefaultClient,
	}
}

// listResponse maps the JSON from the messages.list endpoint.
type listResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// apiMessage maps the JSON from the messages.get endpoint.
type apiMessage struct {
	ID      string `json:"id"`
	Snippet string `json:"snippet"`
	Payload struct {
		MimeType string `json:"mimeType"`
		Headers  []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		Body struct {
			Data string `json:"data"`
		} `json:"body"`
		Parts []apiPart `json:"parts"`
	} `json:"payload"`
}

// apiPart maps a MIME part in the Gmail API response.
type apiPart struct {
	MimeType string `json:"mimeType"`
	Body     struct {
		Data string `json:"data"`
	} `json:"body"`
	Parts []apiPart `json:"parts"`
}

// ListMessages returns message IDs matching a Gmail search query.
func (c *Client) ListMessages(ctx context.Context, query string, maxResults int) ([]Message, error) {
	u := fmt.Sprintf("%s/messages?q=%s&maxResults=%d", apiBase, query, maxResults)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("list messages: decode: %w", err)
	}

	msgs := make([]Message, len(lr.Messages))
	for i, m := range lr.Messages {
		msgs[i] = Message{ID: m.ID}
	}

	return msgs, nil
}

// GetMessage fetches a full message by ID, including headers and body text.
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	u := fmt.Sprintf("%s/messages/%s?format=full", apiBase, messageID)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", messageID, err)
	}

	var am apiMessage
	if err := json.Unmarshal(body, &am); err != nil {
		return nil, fmt.Errorf("get message %s: decode: %w", messageID, err)
	}

	return parseMessage(am)
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// gmail messages are bounded in size, safe to read fully
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return b, nil
}

func parseMessage(am apiMessage) (*Message, error) {
	msg := &Message{ID: am.ID}

	for _, h := range am.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			msg.From = h.Value
		case "to":
			msg.To = h.Value
		case "subject":
			msg.Subject = h.Value
		case "date":
			msg.Date = parseDate(h.Value)
		}
	}

	msg.Body = extractBody(am.Payload.MimeType, am.Payload.Body.Data, am.Payload.Parts)

	return msg, nil
}

// extractBody finds the text/plain content from the message payload.
// it handles single-part messages and recursively searches multipart messages.
func extractBody(mimeType, bodyData string, parts []apiPart) string {
	// single-part text/plain message
	if strings.HasPrefix(mimeType, "text/plain") && bodyData != "" {
		if decoded, err := decodeBase64URL(bodyData); err == nil {
			return decoded
		}
	}

	// search parts recursively for text/plain
	for _, p := range parts {
		if strings.HasPrefix(p.MimeType, "text/plain") && p.Body.Data != "" {
			if decoded, err := decodeBase64URL(p.Body.Data); err == nil {
				return decoded
			}
		}
		// recurse into nested parts (e.g. multipart/alternative inside multipart/mixed)
		if result := extractBody(p.MimeType, p.Body.Data, p.Parts); result != "" {
			return result
		}
	}

	return ""
}

// ParseMIMEBody extracts the text/plain part from a raw MIME multipart body.
// this handles actual MIME content (as opposed to the Gmail API's pre-parsed parts).
func ParseMIMEBody(contentType, body string) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return ""
	}

	boundary := params["boundary"]
	if boundary == "" {
		return ""
	}

	r := multipart.NewReader(strings.NewReader(body), boundary)
	for {
		part, err := r.NextPart()
		if err != nil {
			break
		}

		ct := part.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "text/plain") || (ct == "" && strings.HasPrefix(mediaType, "multipart/")) {
			b, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			if text := string(b); text != "" {
				return text
			}
		}
	}

	return ""
}

func decodeBase64URL(s string) (string, error) {
	b, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseDate tries common email date formats.
func parseDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
