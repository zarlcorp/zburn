package twilio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := NewClient(Config{
		AccountSID: "AC_test_sid",
		AuthToken:  "test_auth_token",
	})
	c.baseURL = srv.URL
	return c
}

func TestBasicAuthHeader(t *testing.T) {
	var gotUser, gotPass string
	var gotAuth bool

	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotAuth = r.BasicAuth()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"available_phone_numbers": []any{}})
	}))

	_, _ = c.SearchNumbers(context.Background(), "US")

	if !gotAuth {
		t.Fatal("basic auth not set")
	}
	if gotUser != "AC_test_sid" {
		t.Errorf("user: got %q, want %q", gotUser, "AC_test_sid")
	}
	if gotPass != "test_auth_token" {
		t.Errorf("pass: got %q, want %q", gotPass, "test_auth_token")
	}
}

func TestSearchNumbers(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		if want := "/AvailablePhoneNumbers/GB/Local.json"; r.URL.Path != want {
			t.Errorf("path: got %s, want %s", r.URL.Path, want)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"available_phone_numbers": []map[string]any{
				{
					"phone_number":  "+441234567890",
					"friendly_name": "(123) 456-7890",
					"capabilities":  map[string]bool{"sms": true, "voice": true},
				},
				{
					"phone_number":  "+441234567891",
					"friendly_name": "(123) 456-7891",
					"capabilities":  map[string]bool{"sms": true, "voice": false},
				},
			},
		})
	}))

	numbers, err := c.SearchNumbers(context.Background(), "GB")
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(numbers) != 2 {
		t.Fatalf("count: got %d, want 2", len(numbers))
	}

	if numbers[0].PhoneNumber != "+441234567890" {
		t.Errorf("numbers[0].PhoneNumber: got %q", numbers[0].PhoneNumber)
	}
	if numbers[0].FriendlyName != "(123) 456-7890" {
		t.Errorf("numbers[0].FriendlyName: got %q", numbers[0].FriendlyName)
	}
	if !numbers[0].Capabilities.SMS {
		t.Error("numbers[0] should have SMS capability")
	}
	if !numbers[0].Capabilities.Voice {
		t.Error("numbers[0] should have Voice capability")
	}
	if numbers[1].Capabilities.Voice {
		t.Error("numbers[1] should not have Voice capability")
	}
}

func TestSearchNumbersEmptyResult(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"available_phone_numbers": []any{}})
	}))

	numbers, err := c.SearchNumbers(context.Background(), "US")
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(numbers) != 0 {
		t.Fatalf("count: got %d, want 0", len(numbers))
	}
}

func TestBuyNumber(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if want := "/IncomingPhoneNumbers.json"; r.URL.Path != want {
			t.Errorf("path: got %s, want %s", r.URL.Path, want)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type: got %q", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostFormValue("PhoneNumber"); got != "+15551234567" {
			t.Errorf("PhoneNumber param: got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"sid":           "PN_test_number_sid",
			"phone_number":  "+15551234567",
			"friendly_name": "(555) 123-4567",
		})
	}))

	pn, err := c.BuyNumber(context.Background(), "+15551234567")
	if err != nil {
		t.Fatalf("buy: %v", err)
	}

	if pn.SID != "PN_test_number_sid" {
		t.Errorf("SID: got %q", pn.SID)
	}
	if pn.PhoneNumber != "+15551234567" {
		t.Errorf("PhoneNumber: got %q", pn.PhoneNumber)
	}
	if pn.FriendlyName != "(555) 123-4567" {
		t.Errorf("FriendlyName: got %q", pn.FriendlyName)
	}
}

func TestReleaseNumber(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: got %s, want DELETE", r.Method)
		}
		if want := "/IncomingPhoneNumbers/PN_test_sid.json"; r.URL.Path != want {
			t.Errorf("path: got %s, want %s", r.URL.Path, want)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	if err := c.ReleaseNumber(context.Background(), "PN_test_sid"); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestListMessages(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}

		q := r.URL.Query()
		if got := q.Get("To"); got != "+15551234567" {
			t.Errorf("To param: got %q", got)
		}
		if got := q.Get("PageSize"); got != "5" {
			t.Errorf("PageSize param: got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{
				{
					"sid":       "SM_msg1",
					"from":      "+15559876543",
					"to":        "+15551234567",
					"body":      "Your code is 123456",
					"date_sent": "Mon, 16 Feb 2026 10:30:00 +0000",
					"status":    "received",
				},
				{
					"sid":       "SM_msg2",
					"from":      "+15559876544",
					"to":        "+15551234567",
					"body":      "Welcome to the service",
					"date_sent": "Mon, 16 Feb 2026 09:00:00 +0000",
					"status":    "received",
				},
			},
		})
	}))

	msgs, err := c.ListMessages(context.Background(), "+15551234567", 5)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("count: got %d, want 2", len(msgs))
	}

	if msgs[0].SID != "SM_msg1" {
		t.Errorf("msgs[0].SID: got %q", msgs[0].SID)
	}
	if msgs[0].From != "+15559876543" {
		t.Errorf("msgs[0].From: got %q", msgs[0].From)
	}
	if msgs[0].To != "+15551234567" {
		t.Errorf("msgs[0].To: got %q", msgs[0].To)
	}
	if msgs[0].Body != "Your code is 123456" {
		t.Errorf("msgs[0].Body: got %q", msgs[0].Body)
	}
	if msgs[0].Status != "received" {
		t.Errorf("msgs[0].Status: got %q", msgs[0].Status)
	}
	if msgs[0].DateSent.IsZero() {
		t.Error("msgs[0].DateSent is zero")
	}
}

func TestListMessagesEmpty(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	}))

	msgs, err := c.ListMessages(context.Background(), "+15551234567", 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(msgs) != 0 {
		t.Fatalf("count: got %d, want 0", len(msgs))
	}
}

func TestErrorInsufficientFunds(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20003,
			"message": "Insufficient funds",
			"status":  402,
		})
	}))

	_, err := c.BuyNumber(context.Background(), "+15551234567")
	if err == nil {
		t.Fatal("expected error")
	}

	var twilioErr *Error
	if !errors.As(err, &twilioErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}

	if twilioErr.StatusCode != 402 {
		t.Errorf("status: got %d, want 402", twilioErr.StatusCode)
	}
	if twilioErr.Code != 20003 {
		t.Errorf("code: got %d, want 20003", twilioErr.Code)
	}
	if twilioErr.Message != "Insufficient funds" {
		t.Errorf("message: got %q", twilioErr.Message)
	}
}

func TestErrorNumberUnavailable(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    21422,
			"message": "Phone number is not available",
			"status":  400,
		})
	}))

	_, err := c.BuyNumber(context.Background(), "+15551234567")
	if err == nil {
		t.Fatal("expected error")
	}

	var twilioErr *Error
	if !errors.As(err, &twilioErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}

	if twilioErr.StatusCode != 400 {
		t.Errorf("status: got %d, want 400", twilioErr.StatusCode)
	}
	if twilioErr.Code != 21422 {
		t.Errorf("code: got %d, want 21422", twilioErr.Code)
	}
}

func TestErrorAuthFailure(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20003,
			"message": "Authentication Error - invalid username",
			"status":  401,
		})
	}))

	_, err := c.SearchNumbers(context.Background(), "US")
	if err == nil {
		t.Fatal("expected error")
	}

	var twilioErr *Error
	if !errors.As(err, &twilioErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}

	if twilioErr.StatusCode != 401 {
		t.Errorf("status: got %d, want 401", twilioErr.StatusCode)
	}
}

func TestErrorReleaseNotFound(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20404,
			"message": "The requested resource was not found",
			"status":  404,
		})
	}))

	err := c.ReleaseNumber(context.Background(), "PN_nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	var twilioErr *Error
	if !errors.As(err, &twilioErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}

	if twilioErr.StatusCode != 404 {
		t.Errorf("status: got %d, want 404", twilioErr.StatusCode)
	}
}

func TestErrorNonJSONResponse(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))

	_, err := c.SearchNumbers(context.Background(), "US")
	if err == nil {
		t.Fatal("expected error")
	}

	var twilioErr *Error
	if !errors.As(err, &twilioErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}

	if twilioErr.StatusCode != 500 {
		t.Errorf("status: got %d, want 500", twilioErr.StatusCode)
	}
	// should fall back to status text when JSON parsing fails
	if twilioErr.Message != "Internal Server Error" {
		t.Errorf("message: got %q, want %q", twilioErr.Message, "Internal Server Error")
	}
}

func TestErrorMessageFormat(t *testing.T) {
	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "with code",
			err:  Error{StatusCode: 400, Code: 21422, Message: "Phone number is not available"},
			want: "twilio: Phone number is not available (code 21422, status 400)",
		},
		{
			name: "without code",
			err:  Error{StatusCode: 500, Message: "Internal Server Error"},
			want: "twilio: Internal Server Error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error(): got %q, want %q", got, tt.want)
			}
		})
	}
}
