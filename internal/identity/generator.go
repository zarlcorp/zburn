package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/zarlcorp/core/pkg/zcrypto"
)

// numEmailPatterns is the total number of email local-part patterns.
const numEmailPatterns = 8

// Generator produces random identity data using crypto/rand.
type Generator struct{}

// New creates a generator.
func New() *Generator {
	return &Generator{}
}

// Generate produces a complete random identity.
// When domain is empty, falls back to the default zburn.id domain.
func (g *Generator) Generate(domain string) Identity {
	if domain == "" {
		domain = defaultDomain
	}
	first, last := g.Name()
	return Identity{
		ID:        g.hexID(),
		FirstName: first,
		LastName:  last,
		Email:     g.Email(first, last, domain),
		Phone:     g.phone(),
		Street:    g.street(),
		City:      pick(cities),
		State:     pick(states),
		Zip:       g.zip(),
		DOB:       g.dob(),
		CreatedAt: time.Now(),
	}
}

// Email generates an email address using the identity's name and domain.
// One of several patterns is chosen randomly per call:
//   - firstname.lastname
//   - firstinitiallastname
//   - firstnamelastname
//   - firstname.lastname + 2 digits
//   - firstinitiallastname + 2 digits
//   - firstinitial.lastname
//   - lastname.firstname
//   - adjective + noun + 4 digits (random, name-independent)
func (g *Generator) Email(firstName, lastName, domain string) string {
	if domain == "" {
		domain = defaultDomain
	}
	first := strings.ToLower(firstName)
	last := strings.ToLower(lastName)
	initial := string(first[0])

	pattern := randIntn(numEmailPatterns)
	var local string
	switch pattern {
	case 0: // firstname.lastname
		local = first + "." + last
	case 1: // firstinitiallastname
		local = initial + last
	case 2: // firstnamelastname
		local = first + last
	case 3: // firstname.lastname + 2 digits
		local = first + "." + last + fmt.Sprintf("%02d", randIntn(100))
	case 4: // firstinitiallastname + 2 digits
		local = initial + last + fmt.Sprintf("%02d", randIntn(100))
	case 5: // firstinitial.lastname
		local = initial + "." + last
	case 6: // lastname.firstname
		local = last + "." + first
	case 7: // adjective + noun + 4 digits
		local = pick(adjectives) + pick(nouns) + fmt.Sprintf("%04d", randIntn(10000))
	}
	return local + "@" + domain
}

// Password generates a password of the given length containing at least
// one character from each class (lower, upper, digit, symbol).
func (g *Generator) Password(length int) string {
	return zcrypto.GeneratePassword(length)
}

// Name generates a random first/last name pair.
func (g *Generator) Name() (first, last string) {
	return pick(firstNames), pick(lastNames)
}

// hexID generates an 8-character hex string.
func (g *Generator) hexID() string {
	b := make([]byte, 4)
	mustRead(b)
	return hex.EncodeToString(b)
}

// phone generates a US fictional phone number: (555) XXX-XXXX.
func (g *Generator) phone() string {
	line := randIntn(10000)
	// second segment 100-999 to look realistic
	prefix := 100 + randIntn(900)
	return fmt.Sprintf("(555) %03d-%04d", prefix, line)
}

// street generates a street address like "1234 Oak Ave".
func (g *Generator) street() string {
	num := 100 + randIntn(9900)
	name := pick(streetNames)
	suffix := pick(streetSuffixes)
	return fmt.Sprintf("%d %s %s", num, name, suffix)
}

// zip generates a 5-digit US zip code.
func (g *Generator) zip() string {
	return fmt.Sprintf("%05d", randIntn(100000))
}

// dob generates a date of birth between 21 and 65 years ago.
func (g *Generator) dob() time.Time {
	now := time.Now()
	minAge := 21
	maxAge := 65
	age := minAge + randIntn(maxAge-minAge+1)
	// subtract years, then randomize day within that year
	base := now.AddDate(-age, 0, 0)
	dayOffset := randIntn(365)
	return base.AddDate(0, 0, -dayOffset).Truncate(24 * time.Hour)
}

// pick returns a random element from a string slice.
func pick(s []string) string {
	return s[randIntn(len(s))]
}

// randIntn returns a cryptographically random int in [0, n).
func randIntn(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// crypto/rand failure is unrecoverable
		panic("crypto/rand: " + err.Error())
	}
	return int(v.Int64())
}

// mustRead fills b with cryptographically random bytes.
func mustRead(b []byte) {
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
}
