package credential

import "time"

// Credential holds login data linked to a generated identity.
type Credential struct {
	ID         string    `json:"id"`
	IdentityID string    `json:"identity_id"`
	Label      string    `json:"label"`
	URL        string    `json:"url"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	TOTPSecret string    `json:"totp_secret,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
