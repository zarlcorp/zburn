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
	id := g.Generate("")

	tests := []struct {
		name  string
		check func() bool
	}{
		{"ID length", func() bool { return len(id.ID) == 8 }},
		{"ID is hex", func() bool { return regexp.MustCompile(`^[0-9a-f]{8}$`).MatchString(id.ID) }},
		{"FirstName non-empty", func() bool { return id.FirstName != "" }},
		{"LastName non-empty", func() bool { return id.LastName != "" }},
		{"Email non-empty", func() bool { return id.Email != "" }},
		{"Email has @ sign", func() bool { return strings.Contains(id.Email, "@") }},
		{"Phone non-empty", func() bool { return id.Phone != "" }},
		{"Phone has 555", func() bool { return strings.HasPrefix(id.Phone, "(555) ") }},
		{"Street non-empty", func() bool { return id.Street != "" }},
		{"City non-empty", func() bool { return id.City != "" }},
		{"State length", func() bool { return len(id.State) == 2 }},
		{"Zip length", func() bool { return len(id.Zip) == 5 }},
		{"DOB non-zero", func() bool { return !id.DOB.IsZero() }},
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

func TestGenerateDefaultDomain(t *testing.T) {
	g := New()
	id := g.Generate("")
	if !strings.HasSuffix(id.Email, "@"+defaultDomain) {
		t.Errorf("empty domain should use default, got %q", id.Email)
	}
}

func TestGenerateCustomDomain(t *testing.T) {
	g := New()
	id := g.Generate("custom.example.com")
	if !strings.HasSuffix(id.Email, "@custom.example.com") {
		t.Errorf("expected custom domain, got %q", id.Email)
	}
}

func TestEmailPatterns(t *testing.T) {
	g := New()

	// each pattern regex describes what the local part looks like
	patterns := []struct {
		name string
		re   *regexp.Regexp
	}{
		{"firstname.lastname", regexp.MustCompile(`^[a-z]+\.[a-z]+@`)},
		{"firstinitiallastname", regexp.MustCompile(`^[a-z][a-z]+@`)},
		{"firstnamelastname", regexp.MustCompile(`^[a-z]+[a-z]+@`)},
		{"firstname.lastname+digits", regexp.MustCompile(`^[a-z]+\.[a-z]+\d{2}@`)},
		{"firstinitiallastname+digits", regexp.MustCompile(`^[a-z][a-z]+\d{2}@`)},
		{"firstinitial.lastname", regexp.MustCompile(`^[a-z]\.[a-z]+@`)},
		{"lastname.firstname", regexp.MustCompile(`^[a-z]+\.[a-z]+@`)},
		{"adjective+noun+4digits", regexp.MustCompile(`^[a-z]+[a-z]+\d{4}@`)},
	}

	// generate many emails and track which patterns we see
	seen := make(map[int]bool)
	for range 500 {
		email := g.Email("John", "Doe", "test.com")

		// must have domain
		if !strings.HasSuffix(email, "@test.com") {
			t.Fatalf("wrong domain in %q", email)
		}

		local := strings.TrimSuffix(email, "@test.com")

		// classify which pattern it matched
		matched := false
		for i, p := range patterns {
			if p.re.MatchString(local + "@") {
				seen[i] = true
				matched = true
				break
			}
		}
		if !matched {
			// all emails should be lowercase alphanumeric with optional dots
			if !regexp.MustCompile(`^[a-z0-9.]+$`).MatchString(local) {
				t.Errorf("email local part %q contains unexpected characters", local)
			}
		}
	}

	// with 8 patterns and 500 iterations, we should see most patterns
	// (probability of missing one is ~(7/8)^500 ≈ 0)
	if len(seen) < 4 {
		t.Errorf("expected variety in email patterns, only saw %d distinct patterns", len(seen))
	}
}

func TestEmailNameIncorporation(t *testing.T) {
	g := New()

	// generate many emails and verify name appears in name-based patterns
	nameFound := 0
	total := 200
	for range total {
		email := g.Email("Alice", "Wonder", "example.com")
		local := strings.Split(email, "@")[0]
		if strings.Contains(local, "alice") || strings.Contains(local, "wonder") {
			nameFound++
		}
	}

	// 7 of 8 patterns include the name, so ~87.5% should contain name parts
	if nameFound < total/2 {
		t.Errorf("expected most emails to contain name parts, got %d/%d", nameFound, total)
	}
}

func TestEmailDefaultDomain(t *testing.T) {
	g := New()
	email := g.Email("Test", "User", "")
	if !strings.HasSuffix(email, "@"+defaultDomain) {
		t.Errorf("empty domain should fall back to default, got %q", email)
	}
}

func TestEmailCustomDomain(t *testing.T) {
	g := New()
	email := g.Email("Test", "User", "my.domain.org")
	if !strings.HasSuffix(email, "@my.domain.org") {
		t.Errorf("expected custom domain, got %q", email)
	}
}

func TestEmailAllLowercase(t *testing.T) {
	g := New()
	for range 100 {
		email := g.Email("JOHN", "DOE", "EXAMPLE.COM")
		local := strings.Split(email, "@")[0]
		if local != strings.ToLower(local) {
			t.Errorf("local part should be lowercase, got %q", local)
		}
	}
}

func TestEmailRandomness(t *testing.T) {
	g := New()
	a := g.Email("John", "Doe", "")
	b := g.Email("John", "Doe", "")
	// even with same name, different pattern or digits should yield different results
	// try a few times to avoid flakes
	different := false
	for range 10 {
		c := g.Email("John", "Doe", "")
		if a != c {
			different = true
			break
		}
	}
	if !different && a == b {
		t.Errorf("emails should vary: got %q repeatedly", a)
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
		id := g.Generate("")
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
		id := g.Generate("")
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
		id := g.Generate("")
		if !re.MatchString(id.Street) {
			t.Errorf("street %q does not match expected pattern", id.Street)
		}
	}
}

func TestZip(t *testing.T) {
	g := New()
	re := regexp.MustCompile(`^\d{5}$`)
	for range 20 {
		id := g.Generate("")
		if !re.MatchString(id.Zip) {
			t.Errorf("zip %q does not match 5-digit pattern", id.Zip)
		}
	}
}

func TestGenerateRandomness(t *testing.T) {
	g := New()
	a := g.Generate("")
	b := g.Generate("")

	// IDs should always differ (8 hex chars = 32 bits of randomness)
	if a.ID == b.ID {
		t.Errorf("consecutive IDs should differ: got %q twice", a.ID)
	}
}

func TestGenerateEmailUsesName(t *testing.T) {
	g := New()
	// generate several identities and verify the email relates to the name
	nameInEmail := 0
	for range 100 {
		id := g.Generate("")
		first := strings.ToLower(id.FirstName)
		last := strings.ToLower(id.LastName)
		local := strings.Split(id.Email, "@")[0]
		if strings.Contains(local, first) || strings.Contains(local, last) {
			nameInEmail++
		}
	}
	// 7/8 patterns use the name
	if nameInEmail < 50 {
		t.Errorf("expected most emails to contain identity name, got %d/100", nameInEmail)
	}
}
