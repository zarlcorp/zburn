package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// password character classes
const (
	lowerChars   = "abcdefghijklmnopqrstuvwxyz"
	upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars   = "0123456789"
	symbolChars  = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	allPassChars = lowerChars + upperChars + digitChars + symbolChars

	defaultPasswordLen = 20
)

// Generator produces random identity data using crypto/rand.
type Generator struct{}

// New creates a generator.
func New() *Generator {
	return &Generator{}
}

// Generate produces a complete random identity.
func (g *Generator) Generate() Identity {
	first, last := g.Name()
	return Identity{
		ID:        g.hexID(),
		FirstName: first,
		LastName:  last,
		Email:     g.Email(),
		Phone:     g.phone(),
		Street:    g.street(),
		City:      pick(cities),
		State:     pick(states),
		Zip:       g.zip(),
		DOB:       g.dob(),
		Password:  g.Password(defaultPasswordLen),
		CreatedAt: time.Now(),
	}
}

// Email generates an email in the form <adjective><noun><4digits>@zburn.id.
func (g *Generator) Email() string {
	adj := pick(adjectives)
	noun := pick(nouns)
	digits := fmt.Sprintf("%04d", randIntn(10000))
	return adj + noun + digits + "@" + emailDomain
}

// Password generates a password of the given length containing at least
// one character from each class (lower, upper, digit, symbol).
func (g *Generator) Password(length int) string {
	if length < 4 {
		length = 4
	}

	buf := make([]byte, length)

	// guarantee one from each class
	buf[0] = pickByte(lowerChars)
	buf[1] = pickByte(upperChars)
	buf[2] = pickByte(digitChars)
	buf[3] = pickByte(symbolChars)

	for i := 4; i < length; i++ {
		buf[i] = pickByte(allPassChars)
	}

	// shuffle using Fisher-Yates
	for i := length - 1; i > 0; i-- {
		j := randIntn(i + 1)
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
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

// pickByte returns a random byte from a string.
func pickByte(s string) byte {
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
