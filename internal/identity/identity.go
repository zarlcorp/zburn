// Package identity generates disposable personal data.
// All generation uses crypto/rand — no math/rand, no side effects.
package identity

import "time"

// Identity holds a complete generated persona.
type Identity struct {
	ID        string    // 8-char hex
	FirstName string
	LastName  string
	Email     string    // <adjective><noun><4digits>@zburn.id
	Phone     string    // US format: (555) XXX-XXXX
	Street    string
	City      string
	State     string    // US state abbreviation
	Zip       string
	DOB       time.Time
	Password  string
	CreatedAt time.Time
}
