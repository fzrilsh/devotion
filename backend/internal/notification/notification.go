// Package notification owns the in-platform notification feed and the fan-out to
// external channels (email and WhatsApp). Its shape is fixed by three rules that
// pull against each other:
//
//   - The in-app notification always persists and is always visible (FR-054): it
//     is the one channel a manual tester can always observe, so it never depends
//     on a channel preference or a successful send.
//   - A notification row is written inside the transaction of the event that
//     triggered it (FR-086), so a rolled-back order change leaves no orphan
//     notification. Enqueue therefore takes a pgx.Tx, not the pool: writing a
//     notification outside its triggering transaction is not expressible.
//   - External delivery is best effort and happens later, in the scheduler job.
//     A send failure must never fail the triggering transaction, and each channel
//     gets at most three attempts before it is marked failed_permanent (FR-085).
//
// Whether an event is transactional is a property of the event, computed by
// IsTransactional, never a column a caller can set: mislabeling a core event as
// non-transactional would let a user silence a notification the product says
// cannot be silenced.
package notification

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// EmailSender delivers one email. The concrete sender talks SMTP to Mailjet with
// net/smtp and no SDK (CLAUDE.md rule 6); it lives in delivery.go. A nil sender
// means email delivery is unavailable, so a claimed email channel is failed
// rather than silently dropped.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// WhatsAppSender delivers one WhatsApp text. The concrete sender is the whatsmeow
// client wired in T024a; until then a nil sender makes a whatsapp channel fail
// its attempt like any other transient error, so the queue does not stall on it.
type WhatsAppSender interface {
	SendText(ctx context.Context, phone, body string) error
}

// Service owns the notification endpoints, the Enqueue entry point domain
// packages call inside their transactions, and the delivery job the scheduler
// runs. It holds the pool for standalone reads, the Clock for every timestamp
// (Rule 5), and the two channel senders (either may be nil). auth is the shared
// authenticator (the account service) so the four routes gate without this
// package owning a session store.
type Service struct {
	pool     *pgxpool.Pool
	clock    platform.Clock
	auth     httpx.Authenticator
	email    EmailSender
	whatsapp WhatsAppSender
}

// New builds a Service. email and whatsapp may be nil before their transports
// exist; a nil sender fails the matching channel's attempt rather than pretending
// it was delivered. auth is reused from the account service so notification
// routes enforce authentication without a second session store.
func New(pool *pgxpool.Pool, clock platform.Clock, auth httpx.Authenticator, email EmailSender, whatsapp WhatsAppSender) *Service {
	return &Service{pool: pool, clock: clock, auth: auth, email: email, whatsapp: whatsapp}
}

// queries returns a Queries bound to the pool for a standalone statement.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// IsTransactional reports whether an event is a core transactional notification
// that always delivers and cannot be silenced by a channel preference (FR-091).
// Only the three genuinely optional events are non-transactional; everything
// else, including confirmation_due_approaching, is transactional because it ends
// in an automatic state change the user cannot afford to miss. This is a function
// over the enum so the classification cannot drift per call site or be set wrong
// by a caller.
func IsTransactional(e sqlcgen.EventType) bool {
	switch e {
	case sqlcgen.EventTypeCalendarStale,
		sqlcgen.EventTypeDeadlineApproaching,
		sqlcgen.EventTypeRatingRequest:
		return false
	default:
		return true
	}
}

// Enqueue writes one in-platform notification and its external channel rows
// inside tx, the transaction of the event that triggered it (FR-086). The
// in-app row is always written (FR-054); link is the optional deep link stored
// in the notification.link column (nil for events with no target). External
// channels fan out by rule: a transactional event queues both email and WhatsApp
// unconditionally, a non-transactional event queues only the channels the account
// left enabled in its preferences (FR-091). The channel rows start pending; the
// scheduler job delivers them later, so a delivery failure never touches tx.
func (s *Service) Enqueue(ctx context.Context, tx pgx.Tx, accountID pgtype.UUID, event sqlcgen.EventType, title, body string, link *string) error {
	q := s.queries().WithTx(tx)

	transactional := IsTransactional(event)
	notifID, err := q.InsertNotification(ctx, sqlcgen.InsertNotificationParams{
		AccountID:     accountID,
		Event:         event,
		Transactional: transactional,
		Title:         title,
		Body:          body,
		Link:          textOrNull(link),
		CreatedAt:     tstz(s.clock.Now()),
	})
	if err != nil {
		return err
	}

	email, whatsapp := true, true
	if !transactional {
		prefs, err := q.GetNotifPreferences(ctx, accountID)
		if err != nil {
			return err
		}
		email, whatsapp = prefs.NotifNontxEmail, prefs.NotifNontxWhatsapp
	}

	if email {
		if err := q.InsertNotificationChannel(ctx, sqlcgen.InsertNotificationChannelParams{
			NotificationID: notifID,
			Channel:        sqlcgen.NotificationChannelTypeEmail,
		}); err != nil {
			return err
		}
	}
	if whatsapp {
		if err := q.InsertNotificationChannel(ctx, sqlcgen.InsertNotificationChannelParams{
			NotificationID: notifID,
			Channel:        sqlcgen.NotificationChannelTypeWhatsapp,
		}); err != nil {
			return err
		}
	}
	return nil
}
