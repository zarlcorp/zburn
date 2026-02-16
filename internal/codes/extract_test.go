package codes

import (
	"testing"
)

func TestExtractCommonFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Code
	}{
		{
			name:  "code is after text",
			input: "Your verification code is 123456",
			want:  Code{Value: "123456", Type: "numeric"},
		},
		{
			name:  "code before text",
			input: "123456 is your code",
			want:  Code{Value: "123456", Type: "numeric"},
		},
		{
			name:  "code after colon",
			input: "Code: 123456",
			want:  Code{Value: "123456", Type: "numeric"},
		},
		{
			name:  "4 digit OTP",
			input: "Your OTP: 1234",
			want:  Code{Value: "1234", Type: "numeric"},
		},
		{
			name:  "8 digit verification",
			input: "Use 12345678 to verify",
			want:  Code{Value: "12345678", Type: "numeric"},
		},
		{
			name:  "alphanumeric confirmation",
			input: "Enter A1B2C3 to confirm",
			want:  Code{Value: "A1B2C3", Type: "alphanumeric"},
		},
		{
			name:  "code after dash",
			input: "Security code - 567890",
			want:  Code{Value: "567890", Type: "numeric"},
		},
		{
			name:  "code with is keyword",
			input: "Your one-time code is 998877",
			want:  Code{Value: "998877", Type: "numeric"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := Extract(tt.input)
			if len(codes) == 0 {
				t.Fatalf("expected at least one code, got none")
			}
			got := codes[0]
			if got.Value != tt.want.Value {
				t.Errorf("value: got %q, want %q", got.Value, tt.want.Value)
			}
			if got.Type != tt.want.Type {
				t.Errorf("type: got %q, want %q", got.Type, tt.want.Type)
			}
		})
	}
}

func TestExtractFiltersCopyright(t *testing.T) {
	codes := Extract("Copyright 2025 Acme Inc. All rights reserved.")
	if len(codes) != 0 {
		t.Errorf("expected no codes, got %v", codes)
	}
}

func TestExtractFiltersPhoneNumbers(t *testing.T) {
	codes := Extract("Call us at 1234567890 for support")
	for _, c := range codes {
		if c.Value == "1234567890" {
			t.Error("extracted phone number as code")
		}
	}
}

func TestExtractFiltersPrice(t *testing.T) {
	codes := Extract("Total: $123.45 due today")
	for _, c := range codes {
		if c.Value == "123" || c.Value == "12345" {
			t.Errorf("extracted price component %q as code", c.Value)
		}
	}
}

func TestExtractFiltersTimestamp(t *testing.T) {
	codes := Extract("Meeting at 10:30 in conference room")
	for _, c := range codes {
		if c.Value == "1030" {
			t.Errorf("extracted timestamp as code")
		}
	}
}

func TestExtractFiltersYear(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"standalone year", "Founded in 2024, we serve customers worldwide"},
		{"year with copyright", "(c) 2026 All rights reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := Extract(tt.input)
			if len(codes) != 0 {
				t.Errorf("expected no codes, got %v", codes)
			}
		})
	}
}

func TestExtractFiltersURL(t *testing.T) {
	codes := Extract("Visit https://example.com/verify/123456 for details")
	for _, c := range codes {
		if c.Value == "123456" {
			t.Error("extracted number from URL as code")
		}
	}
}

func TestExtractFiltersEmail(t *testing.T) {
	codes := Extract("Contact user123456@example.com for help")
	for _, c := range codes {
		if c.Value == "123456" {
			t.Error("extracted number from email as code")
		}
	}
}

func TestExtractMultipleCodesHighestConfidenceFirst(t *testing.T) {
	text := "Your account number is 8765. Your verification code is 123456. Reference: 4321."
	codes := Extract(text)
	if len(codes) < 2 {
		t.Fatalf("expected at least 2 codes, got %d", len(codes))
	}
	// 6-digit with keyword should be first
	if codes[0].Value != "123456" {
		t.Errorf("first code: got %q, want %q", codes[0].Value, "123456")
	}
}

func TestExtractEmptyInput(t *testing.T) {
	codes := Extract("")
	if codes != nil {
		t.Errorf("expected nil, got %v", codes)
	}
}

func TestExtractNoMatch(t *testing.T) {
	codes := Extract("Hello, this is a regular message with no codes.")
	if len(codes) != 0 {
		t.Errorf("expected no codes, got %v", codes)
	}
}

func TestExtractReturnsNilNotEmpty(t *testing.T) {
	codes := Extract("nothing here")
	if codes != nil {
		t.Error("expected nil for no matches")
	}
}

func TestExtractRealWorldMessages(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "google",
			input: "G-123456 is your Google verification code.",
			want:  "123456",
		},
		{
			name:  "aws",
			input: "Your AWS verification code is: 456789",
			want:  "456789",
		},
		{
			name:  "slack",
			input: "Your Slack confirmation code is: 987654",
			want:  "987654",
		},
		{
			name:  "stripe",
			input: "Your Stripe verification code is 112233. Don't share this code with anyone.",
			want:  "112233",
		},
		{
			name:  "github",
			input: "Your GitHub authentication code:\n\n654321\n\nThis code will expire in 10 minutes.",
			want:  "654321",
		},
		{
			name:  "generic sms",
			input: "Your OTP is 7890. Valid for 5 min.",
			want:  "7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := Extract(tt.input)
			if len(codes) == 0 {
				t.Fatal("expected at least one code, got none")
			}
			if codes[0].Value != tt.want {
				t.Errorf("got %q, want %q", codes[0].Value, tt.want)
			}
		})
	}
}

func TestExtractSixDigitRanksAboveFourDigit(t *testing.T) {
	text := "Your PIN is 1234 and your verification code is 567890"
	codes := Extract(text)
	if len(codes) < 2 {
		t.Fatalf("expected at least 2 codes, got %d", len(codes))
	}
	if codes[0].Value != "567890" {
		t.Errorf("expected 6-digit code first, got %q", codes[0].Value)
	}
}

func TestExtractAlphanumericWithKeyword(t *testing.T) {
	text := "Your verification code is X9Y8Z7"
	codes := Extract(text)
	if len(codes) == 0 {
		t.Fatal("expected at least one code, got none")
	}
	if codes[0].Value != "X9Y8Z7" {
		t.Errorf("got %q, want %q", codes[0].Value, "X9Y8Z7")
	}
	if codes[0].Type != "alphanumeric" {
		t.Errorf("type: got %q, want %q", codes[0].Type, "alphanumeric")
	}
}

func TestExtractDoesNotExtractPlainWords(t *testing.T) {
	// "please" is 6 chars, all letters - should not be extracted
	codes := Extract("please verify your account")
	for _, c := range codes {
		if c.Value == "please" {
			t.Error("extracted plain word as code")
		}
	}
}
