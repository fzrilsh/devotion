// Package order holds the work-order domain logic. deadline.go is the single
// home for the time arithmetic behind the two scheduler layers (research.md
// R-07): the compute-on-read predicates in the read queries and the in-process
// ticker both call these functions, so the same order can never look confirmed
// on one page and pending on another. CLAUDE.md rule 5 forbids time.Now() in
// business logic, so every function here takes the current instant as a
// parameter; the caller obtains it from an injected platform.Clock.
package order

import (
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// Window durations fixed by the spec. FR-068: a shipped order auto-confirms 7
// days after the "shipped" status. FR-069/AS-4: the buyer is warned when the
// deadline is two days out. FR-021: a listing calendar untouched for more than
// 7 days is stale. The 72-hour quota reply window (FR-082) is not here because
// it is stored per row as quota_request.reply_due_at, computed once from the
// Clock when the request is sent.
const (
	AutoConfirmWindow   = 7 * 24 * time.Hour
	AutoConfirmWarnLead = 2 * 24 * time.Hour
	CalendarStaleWindow = 7 * 24 * time.Hour
)

// ReadinessDeadline returns the Monday of the week production can start:
// the agreement date plus the listing's readiness lead, rounded to its week
// start (FR-087). It is stored as work_order.readiness_week_start at agreement
// time, never recomputed later, because the listing's lead may change while an
// existing allocation must not shift. Kept here so the write path and any read
// that re-derives it agree on the week boundary.
func ReadinessDeadline(agreedAt time.Time, readinessLeadDays int) time.Time {
	return platform.WeekStart(agreedAt.Add(time.Duration(readinessLeadDays) * 24 * time.Hour))
}

// AutoConfirmAt returns the instant a shipped order is treated as confirmed
// (FR-068): shippedAt plus the 7-day window.
func AutoConfirmAt(shippedAt time.Time) time.Time {
	return shippedAt.Add(AutoConfirmWindow)
}

// IsAutoConfirmDue reports whether a shipped order has passed its auto-confirm
// instant and so must be closed by the scheduler (FR-068). The boundary is
// inclusive: at exactly AutoConfirmAt the order is due, which is the instant
// the read predicate and the job must agree on.
func IsAutoConfirmDue(shippedAt, now time.Time) bool {
	return !now.Before(AutoConfirmAt(shippedAt))
}

// IsAutoConfirmApproaching reports whether a shipped order is within the warning
// lead of its auto-confirm instant but not yet due, so the buyer gets the
// "deadline mendekat" notice before closure (FR-069). Once due, it is no longer
// "approaching"; that case is handled by IsAutoConfirmDue.
func IsAutoConfirmApproaching(shippedAt, now time.Time) bool {
	at := AutoConfirmAt(shippedAt)
	return !now.Before(at.Add(-AutoConfirmWarnLead)) && now.Before(at)
}

// IsCalendarStale reports whether a listing's calendar has gone untouched
// longer than the stale window (FR-021). The marker is informative and must not
// change search order; this predicate only decides the marker and the reminder.
func IsCalendarStale(calendarUpdatedAt, now time.Time) bool {
	return calendarUpdatedAt.Before(now.Add(-CalendarStaleWindow))
}

// IsRequestExpired reports whether a quota request has passed its stored reply
// window (FR-037/FR-082). replyDueAt is quota_request.reply_due_at, set from the
// Clock when the request was sent, so the 72-hour figure lives at write time and
// this predicate only compares. The boundary is inclusive.
func IsRequestExpired(replyDueAt, now time.Time) bool {
	return !now.Before(replyDueAt)
}
