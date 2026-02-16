// Package identity generates disposable personal data.
// All generation uses crypto/rand — no math/rand, no side effects.
package identity

import "time"

// Identity holds a complete generated persona.
type Identity struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Street    string    `json:"street"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	Zip       string    `json:"zip"`
	DOB       time.Time `json:"dob"`
	CreatedAt time.Time `json:"created_at"`
}
