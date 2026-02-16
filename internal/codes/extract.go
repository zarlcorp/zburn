// Package codes extracts 2FA verification codes from email and SMS message bodies.
package codes

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Code represents a potential verification code found in text.
type Code struct {
	Value string // the code itself, e.g. "123456"
	Type  string // "numeric", "alphanumeric"
}

// context keywords that indicate a nearby number is a verification code
var keywords = []string{
	"verification",
	"code",
	"otp",
	"one-time",
	"confirm",
	"pin",
	"security code",
	"2fa",
	"authenticate",
	"verify",
}

// numeric codes: 4, 6, or 8 digits with word boundaries
var numericRe = regexp.MustCompile(`\b(\d{4}|\d{6}|\d{8})\b`)

// alphanumeric codes: 6 chars mixing letters and digits (e.g. A1B2C3)
var alphanumericRe = regexp.MustCompile(`\b([A-Za-z0-9]{6})\b`)

// patterns that indicate a number is not a code
var (
	priceRe     = regexp.MustCompile(`\$\d`)
	urlRe       = regexp.MustCompile(`https?://\S+`)
	emailRe     = regexp.MustCompile(`\S+@\S+\.\S+`)
	phoneRe     = regexp.MustCompile(`\b\d{10,}\b`)
	yearRe      = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	timestampRe = regexp.MustCompile(`\d{1,2}:\d{2}`)
)

// Extract returns all potential verification codes found in the text,
// ordered by confidence (most likely first).
func Extract(text string) []Code {
	if text == "" {
		return nil
	}

	// pre-clean: remove URLs and email addresses so embedded numbers
	// don't get picked up
	cleaned := urlRe.ReplaceAllString(text, " ")
	cleaned = emailRe.ReplaceAllString(cleaned, " ")

	lower := strings.ToLower(cleaned)

	type candidate struct {
		code       Code
		confidence int
	}

	seen := make(map[string]bool)
	var candidates []candidate

	// extract numeric codes
	for _, match := range numericRe.FindAllStringIndex(cleaned, -1) {
		val := cleaned[match[0]:match[1]]

		if seen[val] {
			continue
		}

		if isFiltered(cleaned, lower, val, match[0], match[1]) {
			continue
		}

		c := candidate{
			code: Code{Value: val, Type: "numeric"},
		}
		c.confidence = score(lower, val, match[0], match[1])
		seen[val] = true
		candidates = append(candidates, c)
	}

	// extract alphanumeric codes (must contain both letters and digits)
	for _, match := range alphanumericRe.FindAllStringIndex(cleaned, -1) {
		val := cleaned[match[0]:match[1]]

		if seen[val] {
			continue
		}

		if !hasMixedAlphaDigit(val) {
			continue
		}

		// skip if it looks like a common word
		if isCommonWord(val) {
			continue
		}

		c := candidate{
			code: Code{Value: val, Type: "alphanumeric"},
		}
		c.confidence = score(lower, val, match[0], match[1])
		seen[val] = true
		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].confidence > candidates[j].confidence
	})

	result := make([]Code, len(candidates))
	for i, c := range candidates {
		result[i] = c.code
	}
	return result
}

// isFiltered returns true if the numeric value at the given position should
// be excluded (years, phone numbers, prices, timestamps).
func isFiltered(text, lower, val string, start, end int) bool {
	digits := len(val)

	// phone numbers: 10+ digits already handled by the regex only matching
	// 4, 6, or 8 — but check surrounding context for phone patterns
	if digits >= 4 {
		// check if preceded by a price symbol
		if start > 0 && text[start-1] == '$' {
			return true
		}
		// check wider context for price pattern
		if start >= 1 {
			prefix := text[max(0, start-2):start]
			if strings.Contains(prefix, "$") {
				return true
			}
		}
	}

	// filter years: 4-digit numbers matching common year patterns
	if digits == 4 && yearRe.MatchString(val) {
		ctx := surroundingContext(lower, start, end, 30)
		yearWords := []string{"copyright", "(c)", "year", "since", "est.", "founded"}
		for _, w := range yearWords {
			if strings.Contains(ctx, w) {
				return true
			}
		}
		// also filter if it looks like a standalone year with no code keywords
		if !hasKeywordNearby(lower, start, end, 60) {
			return true
		}
	}

	// filter timestamps like 12:34
	if digits == 4 {
		if end < len(text) && text[end] == ':' {
			return true
		}
		if start > 0 && text[start-1] == ':' {
			return true
		}
	}

	// filter if part of a longer number sequence (phone-like)
	if start > 0 && unicode.IsDigit(rune(text[start-1])) {
		return true
	}
	if end < len(text) && unicode.IsDigit(rune(text[end])) {
		return true
	}

	// filter decimal numbers (prices like 123.45)
	if end < len(text) && text[end] == '.' {
		if end+1 < len(text) && unicode.IsDigit(rune(text[end+1])) {
			return true
		}
	}
	if start > 0 && text[start-1] == '.' {
		if start >= 2 && unicode.IsDigit(rune(text[start-2])) {
			return true
		}
	}

	return false
}

// score assigns a confidence value to a candidate code based on context.
func score(lower, val string, start, end int) int {
	s := 0

	digits := 0
	for _, r := range val {
		if unicode.IsDigit(r) {
			digits++
		}
	}

	// 6-digit numeric codes are the most common 2FA format
	if digits == 6 && len(val) == 6 {
		s += 30
	} else if digits == 8 && len(val) == 8 {
		s += 20
	} else if digits == 4 && len(val) == 4 {
		s += 15
	}

	// alphanumeric codes are less common
	if len(val) == 6 && digits < 6 {
		s += 10
	}

	// keyword proximity boost
	if hasKeywordNearby(lower, start, end, 60) {
		s += 50
	}

	// structural clues: code follows "is", ":", or "-"
	prefix := surroundingContext(lower, start, start, 10)
	prefix = strings.TrimRight(prefix, " ")
	if strings.HasSuffix(prefix, ":") || strings.HasSuffix(prefix, "is") || strings.HasSuffix(prefix, "-") {
		s += 20
	}

	// code on its own line or surrounded by whitespace
	if isIsolated(lower, start, end) {
		s += 10
	}

	return s
}

// hasKeywordNearby checks if any verification-related keyword appears
// within radius characters of the code position.
func hasKeywordNearby(lower string, start, end, radius int) bool {
	ctx := surroundingContext(lower, start, end, radius)
	for _, kw := range keywords {
		if strings.Contains(ctx, kw) {
			return true
		}
	}
	return false
}

// surroundingContext returns the text within radius chars around [start, end).
func surroundingContext(text string, start, end, radius int) string {
	lo := start - radius
	if lo < 0 {
		lo = 0
	}
	hi := end + radius
	if hi > len(text) {
		hi = len(text)
	}
	return text[lo:hi]
}

// isIsolated checks if the code is surrounded by whitespace or line boundaries.
func isIsolated(text string, start, end int) bool {
	before := start == 0 || text[start-1] == '\n' || text[start-1] == ' ' || text[start-1] == '\t'
	after := end >= len(text) || text[end] == '\n' || text[end] == ' ' || text[end] == '\t'
	return before && after
}

// hasMixedAlphaDigit returns true if s contains both letters and digits.
func hasMixedAlphaDigit(s string) bool {
	hasLetter := false
	hasDigit := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// isCommonWord filters out 6-char strings that look like normal words
// rather than verification codes.
func isCommonWord(s string) bool {
	lower := strings.ToLower(s)
	// if it's all letters, it's probably a word not a code
	allLetters := true
	for _, r := range lower {
		if !unicode.IsLetter(r) {
			allLetters = false
			break
		}
	}
	return allLetters
}
