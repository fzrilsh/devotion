package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// mailjetSMTPHost is Mailjet's public SMTP endpoint. It is a service address,
// not a credential, so it lives here rather than in Config; the API key and
// secret that authenticate against it come from Config and are never in code
// (CLAUDE.md rule 6, email via net/smtp with no SDK).
const mailjetSMTPHost = "in-v3.mailjet.com"

// mailjetSMTPPort is the submission port. 587 (STARTTLS) is the port net/smtp's
// SendMail negotiates TLS on after the initial plaintext greeting.
const mailjetSMTPPort = "587"

// deliverBatch caps how many pending channels one tick claims. A small batch
// keeps a single tick short on the 2GB box; anything still pending is picked up
// on the next tick, and attempted_at NULLS FIRST keeps never-tried rows ahead.
const deliverBatch = 50

// DeliverJob is the scheduler job that fans pending channel rows out to email
// and WhatsApp. It is registered by serve under LockKeyNotificationDeliver, so
// only one instance delivers during a deploy rollover. A send failure marks the
// channel failed (bumping attempts), never touches the in-app notification, and
// after three attempts the row is failed_permanent (FR-085). The in-app feed is
// unaffected either way (FR-054).
func (s *Service) DeliverJob() scheduler.Job {
	return scheduler.Job{
		Name:    "notification:deliver",
		LockKey: scheduler.LockKeyNotificationDeliver,
		Run: func(ctx context.Context, conn *pgxpool.Conn) error {
			return s.deliver(ctx, conn)
		},
	}
}

// deliver claims one batch of pending channels on the locked connection and
// attempts each. It uses the job's pinned connection (which holds the advisory
// lock) for the claim and the status writes, so the whole job shares one
// session. One channel's failure does not stop the rest.
func (s *Service) deliver(ctx context.Context, conn *pgxpool.Conn) error {
	q := sqlcgen.New(conn)
	rows, err := q.ClaimPendingChannels(ctx, deliverBatch)
	if err != nil {
		return err
	}
	for _, row := range rows {
		s.attempt(ctx, q, row)
	}
	return nil
}

// attempt delivers one channel and records the outcome. A nil sender for the
// channel is a failed attempt, not a silent drop: the row keeps its place in the
// queue and, after three failures, is marked failed_permanent so an operator can
// see the channel never had a transport. MarkChannelSent/MarkChannelFailed both
// bump attempts, so the CASE in MarkChannelFailed flips status at the third try.
func (s *Service) attempt(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.ClaimPendingChannelsRow) {
	err := s.send(ctx, row)
	now := tstz(s.clock.Now())
	if err != nil {
		_ = q.MarkChannelFailed(ctx, sqlcgen.MarkChannelFailedParams{
			ID:          row.ChannelID,
			LastError:   pgText(truncateErr(err)),
			AttemptedAt: now,
		})
		return
	}
	_ = q.MarkChannelSent(ctx, sqlcgen.MarkChannelSentParams{
		ID:          row.ChannelID,
		AttemptedAt: now,
	})
}

// send dispatches to the transport for the row's channel. A missing transport is
// an error so the attempt is recorded as failed rather than pretending success.
func (s *Service) send(ctx context.Context, row sqlcgen.ClaimPendingChannelsRow) error {
	switch row.Channel {
	case sqlcgen.NotificationChannelTypeEmail:
		if s.email == nil {
			return fmt.Errorf("pengirim email tidak tersedia")
		}
		return s.email.Send(ctx, row.Email, row.Title, row.Body)
	case sqlcgen.NotificationChannelTypeWhatsapp:
		if s.whatsapp == nil {
			return fmt.Errorf("pengirim whatsapp tidak tersedia")
		}
		return s.whatsapp.SendText(ctx, row.Phone, row.Body+"\n\n"+row.Title)
	default:
		return fmt.Errorf("kanal tidak dikenal: %s", row.Channel)
	}
}

// truncateErr caps last_error so a verbose transport error cannot bloat the row.
// The column is for operator triage, not full transcripts.
func truncateErr(err error) string {
	const max = 500
	s := err.Error()
	if len(s) > max {
		return s[:max]
	}
	return s
}

// MailjetSender is the concrete EmailSender: SMTP submission to Mailjet with
// net/smtp and no SDK (CLAUDE.md rule 6). The API key and secret are the SMTP
// username and password; from is MAIL_FROM. It is wired in serve from Config, so
// this package never holds a credential literal.
type MailjetSender struct {
	from   string
	apiKey string
	secret string
}

// NewMailjetSender builds the SMTP email sender. All three come from Config.
func NewMailjetSender(from, apiKey, secret string) *MailjetSender {
	return &MailjetSender{from: from, apiKey: apiKey, secret: secret}
}

// Send submits one plaintext email over STARTTLS. net/smtp.SendMail issues
// STARTTLS automatically when the server advertises it, which Mailjet does on
// 587, so the credentials never cross the wire in the clear.
func (m *MailjetSender) Send(ctx context.Context, to, subject, body string) error {
	auth := smtp.PlainAuth("", m.apiKey, m.secret, mailjetSMTPHost)
	msg := buildMessage(m.from, to, subject, body)
	return smtp.SendMail(mailjetSMTPHost+":"+mailjetSMTPPort, auth, m.from, []string{to}, msg)
}

// buildMessage assembles a minimal RFC 5322 message. Bahasa Indonesia bodies are
// UTF-8, declared in the Content-Type header so a subject or body with non-ASCII
// characters renders correctly.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return []byte(b.String())
}
