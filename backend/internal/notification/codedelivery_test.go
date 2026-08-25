package notification

import (
	"context"
	"strings"
	"testing"
)

// recordingSender captures the arguments of the last Send/SendText so a test can
// assert the code adapter passes the right recipient and body to the transport.
type recordingSender struct {
	to, subject, body string
	phone, phoneBody  string
	calls             int
}

func (r *recordingSender) Send(_ context.Context, to, subject, body string) error {
	r.to, r.subject, r.body = to, subject, body
	r.calls++
	return nil
}

func (r *recordingSender) SendText(_ context.Context, phone, body string) error {
	r.phone, r.phoneBody = phone, body
	r.calls++
	return nil
}

// TestCodeDelivery_SendEmailCode_UsesEmailTransport proves registration's email
// code is handed to the email transport with the code in the body, which is the
// missing wiring that kept /register from sending anything. FR-001.
func TestCodeDelivery_SendEmailCode_UsesEmailTransport_FR001(t *testing.T) {
	rec := &recordingSender{}
	d := NewCodeDelivery(rec, nil)

	if err := d.SendEmailCode(context.Background(), "usaha@example.com", "123456"); err != nil {
		t.Fatalf("SendEmailCode: %v", err)
	}
	if rec.to != "usaha@example.com" {
		t.Fatalf("to = %q, mau usaha@example.com", rec.to)
	}
	if !strings.Contains(rec.body, "123456") {
		t.Fatalf("body tidak memuat kode: %q", rec.body)
	}
	if rec.subject == "" {
		t.Fatal("subject kosong")
	}
}

// TestCodeDelivery_SendPhoneCode_UsesWhatsAppTransport proves the phone code goes
// over WhatsApp with the code in the message. FR-001.
func TestCodeDelivery_SendPhoneCode_UsesWhatsAppTransport_FR001(t *testing.T) {
	rec := &recordingSender{}
	d := NewCodeDelivery(nil, rec)

	if err := d.SendPhoneCode(context.Background(), "+628123456789", "654321"); err != nil {
		t.Fatalf("SendPhoneCode: %v", err)
	}
	if rec.phone != "+628123456789" {
		t.Fatalf("phone = %q", rec.phone)
	}
	if !strings.Contains(rec.phoneBody, "654321") {
		t.Fatalf("body tidak memuat kode: %q", rec.phoneBody)
	}
}

// TestCodeDelivery_SendRecoveryCode_UsesEmailTransport proves recovery codes go
// by email with the code in the body. FR-001.
func TestCodeDelivery_SendRecoveryCode_UsesEmailTransport_FR001(t *testing.T) {
	rec := &recordingSender{}
	d := NewCodeDelivery(rec, nil)

	if err := d.SendRecoveryCode(context.Background(), "usaha@example.com", "777888"); err != nil {
		t.Fatalf("SendRecoveryCode: %v", err)
	}
	if !strings.Contains(rec.body, "777888") {
		t.Fatalf("body tidak memuat kode: %q", rec.body)
	}
}

// TestCodeDelivery_NilTransportIsBestEffortError proves a missing transport
// returns an error rather than panicking, so the account service (which swallows
// the error) never fails a registration when a channel is unconfigured. FR-001.
func TestCodeDelivery_NilTransportIsBestEffortError_FR001(t *testing.T) {
	d := NewCodeDelivery(nil, nil)
	if err := d.SendEmailCode(context.Background(), "usaha@example.com", "1"); err == nil {
		t.Fatal("mau error saat email transport nil")
	}
	if err := d.SendPhoneCode(context.Background(), "+628", "1"); err == nil {
		t.Fatal("mau error saat whatsapp transport nil")
	}
}
