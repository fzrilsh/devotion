package observability

import (
	"testing"

	"github.com/getsentry/sentry-go"
)

// TestScrub_DropsRequestUserAndExtra proves the allowlist: an event carrying a
// password in the request body, a session token in a cookie, a phone number in
// the user record, and an identity-document reference in Extra comes out with
// none of them. The scrub rebuilds the event from safe fields, so anything not
// explicitly copied is gone. This is the guard against leaking secrets and
// identity-document data to Sentry (FR-082, CLAUDE.md Keamanan).
func TestScrub_DropsRequestUserAndExtra_FR082(t *testing.T) {
	event := sentry.NewEvent()
	event.Message = "gagal memproses unggahan"
	event.Level = sentry.LevelError
	event.Tags = map[string]string{"request_id": "req-123", "phone": "628123456789"}

	event.Request = &sentry.Request{
		URL:         "https://devotion.example/api/auth/login",
		Cookies:     "devotion_session=secret-token-value",
		Data:        `{"password":"rahasia123","phone":"628123456789"}`,
		QueryString: "token=leaked",
		Headers:     map[string]string{"Authorization": "Bearer secret-token-value"},
	}
	event.User = sentry.User{
		ID:        "user-1",
		Email:     "korban@example.com",
		IPAddress: "203.0.113.7",
		Data:      map[string]string{"phone": "628123456789", "ktp": "identity-document-ref"},
	}
	event.Extra = map[string]any{
		"identity_document": "ktp-scan-uuid",
		"password":          "rahasia123",
	}

	out := scrub(event, nil)

	if out.Request != nil {
		t.Fatalf("Request diteruskan, mau nil: %+v", out.Request)
	}
	if out.User.ID != "" || out.User.Email != "" || out.User.IPAddress != "" || out.User.Data != nil {
		t.Fatalf("User diteruskan, mau kosong: %+v", out.User)
	}
	if len(out.Extra) != 0 {
		t.Fatalf("Extra diteruskan, mau kosong: %+v", out.Extra)
	}
	if len(out.Contexts) != 0 {
		t.Fatalf("Contexts diteruskan, mau kosong: %+v", out.Contexts)
	}

	// The safe fields survive: the message, level, and request_id tag are what
	// makes a report useful without carrying user data.
	if out.Message != "gagal memproses unggahan" {
		t.Fatalf("Message = %q, mau pesan asli", out.Message)
	}
	if out.Level != sentry.LevelError {
		t.Fatalf("Level = %q, mau error", out.Level)
	}
	if out.Tags["request_id"] != "req-123" {
		t.Fatalf("tag request_id = %q, mau req-123", out.Tags["request_id"])
	}

	// The phone-bearing tag is not on the allowlist, so it is dropped even
	// though it rode in on the tags map.
	if _, ok := out.Tags["phone"]; ok {
		t.Fatal("tag phone diteruskan, mau dibuang")
	}
}

// TestInit_EmptyDSN_NoOp proves a blank DSN yields a working no-op flush and no
// error, so development and an opted-out production need no DSN guard of their
// own.
func TestInit_EmptyDSN_NoOp(t *testing.T) {
	flush, err := Init("", "test")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if flush == nil {
		t.Fatal("flush nil, mau no-op")
	}
	flush()
}
