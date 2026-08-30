// Package order holds the work-order domain logic. deadline.go is the single
// home for the time arithmetic behind the two scheduler layers (research.md
// R-07): the compute-on-read predicates in the read queries and the in-process
// ticker both call these functions, so the same order can never look confirmed
// on one page and pending on another. CLAUDE.md rule 5 forbids reading the wall
// clock in business logic, so every function here takes the current instant as a
// parameter; the caller obtains it from an injected platform.Clock.
package order

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
	// DeadlineApproachingLead is how far ahead of a delivery deadline the FR-051
	// "deadline mendekat" notice goes out: an active, not-yet-shipped order enters
	// the warning band seven days before its deadline day.
	DeadlineApproachingLead = 7 * 24 * time.Hour
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

// AutoConfirmBase resolves the effective start of the 7-day auto-confirm clock
// (FR-068) from a work order's two time columns, the single home for the
// COALESCE(auto_confirm_base_at, shipped_at) rule so no caller reimplements it.
// Normally an order's base is shipped_at, the moment shipment was declared. When
// a dispute closed "continued" on an order that had already shipped, the resolve
// path stamps auto_confirm_base_at with the mediation-close instant and the clock
// restarts from there (data-model.md); shipped_at is left as the historical fact
// of when goods shipped. baseAt is work_order.auto_confirm_base_at (NULL for the
// common case), shippedAt is work_order.shipped_at. The storage-level
// auto_confirm_base_after_shipped CHECK guarantees a non-null base is never before
// shipped_at, so the returned instant only ever moves the clock forward.
func AutoConfirmBase(baseAt, shippedAt pgtype.Timestamptz) time.Time {
	if baseAt.Valid {
		return baseAt.Time
	}
	return shippedAt.Time
}

// AutoConfirmAt returns the instant a shipped order is treated as confirmed
// (FR-068): the auto-confirm base plus the 7-day window. baseAt is the effective
// base, normally shipped_at, but the mediation-close instant on an order whose
// dispute closed "continued" after it had already shipped (data-model.md); the
// caller resolves that COALESCE(auto_confirm_base_at, shipped_at) once, so this
// function only adds the window and never has to know which base it was given.
func AutoConfirmAt(baseAt time.Time) time.Time {
	return baseAt.Add(AutoConfirmWindow)
}

// IsAutoConfirmDue reports whether a shipped order has passed its auto-confirm
// instant and so must be closed by the scheduler (FR-068). baseAt is the effective
// base (see AutoConfirmAt). The boundary is inclusive: at exactly AutoConfirmAt the
// order is due, which is the instant the read predicate and the job must agree on.
// An open dispute halts the count entirely (FR-070): the whole decision of whether
// an order is due lives here, not half in this function and half in the scheduler
// query, so the read layer cannot even ask "is this due" without stating whether a
// dispute is open. That makes it impossible for the two layers to diverge on a
// disputed order the way they would if the dispute check were a separate
// conjunction bolted on beside each call site.
func IsAutoConfirmDue(baseAt, now time.Time, hasOpenDispute bool) bool {
	if hasOpenDispute {
		return false
	}
	return !now.Before(AutoConfirmAt(baseAt))
}

// IsAutoConfirmApproaching reports whether a shipped order is within the warning
// lead of its auto-confirm instant but not yet due, so the buyer gets the
// "deadline mendekat" notice before closure (FR-069). baseAt is the effective base
// (see AutoConfirmAt). Once due, it is no longer "approaching"; that case is
// handled by IsAutoConfirmDue.
func IsAutoConfirmApproaching(baseAt, now time.Time) bool {
	at := AutoConfirmAt(baseAt)
	return !now.Before(at.Add(-AutoConfirmWarnLead)) && now.Before(at)
}

// PastDeadlineCutoff returns the start of now's WIB calendar day. A work order
// whose deadline (a date column) falls strictly before this instant is late
// (FR-045): the order stays on time through the whole of its deadline day, so
// only a later WIB day counts as past due. Both the admin compute-on-read query
// and the notifier scan take this one value as their `wo.deadline < cutoff`
// threshold, the same way the auto-confirm layers share AutoConfirmWindow, so a
// late order can never appear on the admin list yet be skipped by the job, or
// the reverse. Computing it in Go keeps the wall clock out of SQL (Rule 5).
func PastDeadlineCutoff(now time.Time) time.Time {
	ny, nm, nd := now.In(platform.Jakarta).Date()
	return time.Date(ny, nm, nd, 0, 0, 0, 0, platform.Jakarta)
}

// IsPastDeadline reports whether a work order's delivery deadline has passed
// (FR-045). deadline is a work_order.deadline value, a date with no time of day,
// so the order is late only once a strictly later WIB calendar day has begun; it
// stays on time through the evening of the deadline day. It compares the
// deadline's WIB day against PastDeadlineCutoff(now), the shared threshold, so a
// Go-side check and the SQL predicate agree at the Jakarta-midnight boundary and
// the date-versus-timestamp mismatch cannot flip the answer at UTC midnight.
func IsPastDeadline(deadline, now time.Time) bool {
	dy, dm, dd := deadline.In(platform.Jakarta).Date()
	deadlineDay := time.Date(dy, dm, dd, 0, 0, 0, 0, platform.Jakarta)
	return PastDeadlineCutoff(now).After(deadlineDay)
}

// DeadlineApproachingCutoff returns the latest deadline day still inside the
// FR-051 warning band: the start of now's WIB calendar day plus the 7-day lead.
// An active, not-yet-shipped order whose deadline (a date column) falls on or
// before this day, but is not already past due, is warned once that its delivery
// deadline is near. Computing it in Go keeps the wall clock out of SQL (Rule 5),
// and the shared PastDeadlineCutoff excludes orders that are already late, so the
// approaching notice and the past-deadline notice never both fire for one order
// on the same tick.
func DeadlineApproachingCutoff(now time.Time) time.Time {
	return PastDeadlineCutoff(now).Add(DeadlineApproachingLead)
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
