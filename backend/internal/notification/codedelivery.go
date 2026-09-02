package notification

import (
	"context"
	"fmt"
)

// CodeDelivery sends verification and recovery codes over the same email and
// WhatsApp transports the delivery job uses, but out of band of the notification
// queue: the account service issues a code and hands the plaintext here exactly
// once, so there is no in-app notification row for a one-time code. It satisfies
// account.CodeDelivery structurally, so this package does not import account and
// no dependency cycle forms.
//
// Either transport may be nil (email before Mailjet is configured, WhatsApp
// before the link is up); a nil transport makes the matching Send a no-op error
// the caller swallows, matching the best-effort contract of resend and recover.
type CodeDelivery struct {
	email    EmailSender
	whatsapp WhatsAppSender
}

// NewCodeDelivery builds the code delivery adapter from the same senders wired
// into the notification Service. Passing the senders (not the Service) keeps the
// account service unaware of the notification queue.
func NewCodeDelivery(email EmailSender, whatsapp WhatsAppSender) *CodeDelivery {
	return &CodeDelivery{email: email, whatsapp: whatsapp}
}

// SendEmailCode delivers a registration email verification code.
func (d *CodeDelivery) SendEmailCode(ctx context.Context, email, code string) error {
	if d.email == nil {
		return fmt.Errorf("pengirim email tidak tersedia")
	}
	subject := "Kode verifikasi email Devotion"
	body := fmt.Sprintf(
		"Kode verifikasi email Anda: %s\n\n"+
			"Masukkan kode ini untuk mengaktifkan akun Devotion Anda. "+
			"Kode berlaku 15 menit. Abaikan pesan ini bila Anda tidak mendaftar.",
		code,
	)
	return d.email.Send(ctx, email, subject, body)
}

// SendPhoneCode delivers a phone verification code over WhatsApp.
func (d *CodeDelivery) SendPhoneCode(ctx context.Context, phone, code string) error {
	if d.whatsapp == nil {
		return fmt.Errorf("pengirim whatsapp tidak tersedia")
	}
	body := fmt.Sprintf(
		"Kode verifikasi nomor Devotion Anda: %s\n\n"+
			"Kode berlaku 15 menit.",
		code,
	)
	return d.whatsapp.SendText(ctx, phone, body)
}

// SendRecoveryCode delivers a password recovery code by email.
func (d *CodeDelivery) SendRecoveryCode(ctx context.Context, email, code string) error {
	if d.email == nil {
		return fmt.Errorf("pengirim email tidak tersedia")
	}
	subject := "Kode pemulihan kata sandi Devotion"
	body := fmt.Sprintf(
		"Kode pemulihan kata sandi Anda: %s\n\n"+
			"Masukkan kode ini untuk mengatur ulang kata sandi. "+
			"Kode berlaku 15 menit. Abaikan pesan ini bila Anda tidak meminta pemulihan.",
		code,
	)
	return d.email.Send(ctx, email, subject, body)
}
