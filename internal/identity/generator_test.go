package identity

import (
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode"
)

func TestGenerate(t *testing.T) {
	g := New()
	id := g.Generate()

	tests := []struct {
		name  string
		check func() bool
	}{
		{"ID length", func() bool { return len(id.ID) == 8 }},
		{"ID is hex", func() bool { return regexp.MustCompile(`^[0-9a-f]{8}$`).MatchString(id.ID) }},
		{"FirstName non-empty", func() bool { return id.FirstName != "" }},
		{"LastName non-empty", func() bool { return id.LastName != "" }},
		{"Email non-empty", func() bool { return id.Email != "" }},
		{"Email has domain", func() bool { return strings.HasSuffix(id.Email, "@"+emailDomain) }},
		{"Phone non-empty", func() bool { return id.Phone != "" }},
		{"Phone has 555", func() bool { return strings.HasPrefix(id.Phone, "(555) ") }},
		{"Street non-empty", func() bool { return id.Street != "" }},
		{"City non-empty", func() bool { return id.City != "" }},
		{"State length", func() bool { return len(id.State) == 2 }},
		{"Zip length", func() bool { return len(id.Zip) == 5 }},
		{"DOB non-zero", func() bool { return !id.DOB.IsZero() }},
		{"Password non-empty", func() bool { return id.Password != "" }},
		{"CreatedAt non-zero", func() bool { return !id.CreatedAt.IsZero() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("check failed for identity: %+v", id)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	g := New()
	re := regexp.MustCompile(`^[a-z]+[a-z]+\d{4}@` + regexp.QuoteMeta(emailDomain) + `$`)

	for range 20 {
		email := g.Email()
		if !re.MatchString(email) {
			t.Errorf("email %q does not match pattern", email)
		}
	}
}

func TestEmailRandomness(t *testing.T) {
	g := New()
	a := g.Email()
	b := g.Email()
	// with ~50 adjectives * ~50 nouns * 10000 digits, collision is extremely unlikely
	if a == b {
		t.Errorf("consecutive emails should differ: got %q twice", a)
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected output length
	}{
		{"default length", 20, 20},
		{"short", 8, 8},
		{"minimum clamp", 2, 4},
		{"exact minimum", 4, 4},
		{"long", 64, 64},
	}

	g := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw := g.Password(tt.length)
			if len(pw) != tt.want {
				t.Errorf("Password(%d) length = %d, want %d", tt.length, len(pw), tt.want)
			}
		})
	}
}

func TestPasswordCharacterClasses(t *testing.T) {
	g := New()

	// run multiple times to ensure the guarantee holds
	for range 50 {
		pw := g.Password(20)
		var hasLower, hasUpper, hasDigit, hasSymbol bool
		for _, r := range pw {
			switch {
			case unicode.IsLower(r):
				hasLower = true
			case unicode.IsUpper(r):
				hasUpper = true
			case unicode.IsDigit(r):
				hasDigit = true
			default:
				hasSymbol = true
			}
		}

		if !hasLower {
			t.Errorf("password %q missing lowercase", pw)
		}
		if !hasUpper {
			t.Errorf("password %q missing uppercase", pw)
		}
		if !hasDigit {
			t.Errorf("password %q missing digit", pw)
		}
		if !hasSymbol {
			t.Errorf("password %q missing symbol", pw)
		}
	}
}

func TestPasswordRandomness(t *testing.T) {
	g := New()
	a := g.Password(20)
	b := g.Password(20)
	if a == b {
		t.Errorf("consecutive passwords should differ: got %q twice", a)
	}
}

func TestName(t *testing.T) {
	g := New()
	for range 20 {
		first, last := g.Name()
		if first == "" {
			t.Error("first name is empty")
		}
		if last == "" {
			t.Error("last name is empty")
		}
	}
}

func TestNameRandomness(t *testing.T) {
	g := New()
	f1, l1 := g.Name()
	// try a few times — with 100 options each, consecutive identical pairs are rare
	different := false
	for range 5 {
		f2, l2 := g.Name()
		if f1 != f2 || l1 != l2 {
			different = true
			break
		}
	}
	if !different {
		t.Errorf("name generation appears non-random: got %s %s every time", f1, l1)
	}
}

func TestPhone(t *testing.T) {
	g := New()
	re := regexp.MustCompile(`^\(555\) \d{3}-\d{4}$`)
	for range 20 {
		id := g.Generate()
		if !re.MatchString(id.Phone) {
			t.Errorf("phone %q does not match (555) XXX-XXXX pattern", id.Phone)
		}
	}
}

func TestDOBRange(t *testing.T) {
	g := New()
	now := time.Now()
	minDOB := now.AddDate(-66, 0, 0) // slightly wider to avoid edge cases
	maxDOB := now.AddDate(-21, 0, 1)

	for range 100 {
		id := g.Generate()
		if id.DOB.Before(minDOB) || id.DOB.After(maxDOB) {
			age := now.Sub(id.DOB).Hours() / 24 / 365.25
			t.Errorf("DOB %s out of range (age ~%.1f)", id.DOB.Format("2006-01-02"), age)
		}
	}
}

func TestStreet(t *testing.T) {
	g := New()
	// pattern: number, space, word(s), space, suffix
	re := regexp.MustCompile(`^\d+ [A-Za-z]+ [A-Za-z]+$`)
	for range 20 {
		id := g.Generate()
		if !re.MatchString(id.Street) {
			t.Errorf("street %q does not match expected pattern", id.Street)
		}
	}
}

func TestZip(t *testing.T) {
	g := New()
	re := regexp.MustCompile(`^\d{5}$`)
	for range 20 {
		id := g.Generate()
		if !re.MatchString(id.Zip) {
			t.Errorf("zip %q does not match 5-digit pattern", id.Zip)
		}
	}
}

func TestGenerateRandomness(t *testing.T) {
	g := New()
	a := g.Generate()
	b := g.Generate()

	// IDs should always differ (8 hex chars = 32 bits of randomness)
	if a.ID == b.ID {
		t.Errorf("consecutive IDs should differ: got %q twice", a.ID)
	}
}
