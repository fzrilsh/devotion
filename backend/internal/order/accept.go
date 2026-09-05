package order

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Register wires the accept route. It is Gated behind both business roles
// because either party may close the negotiation: whoever did not make the
// standing offer accepts it (FR-033, spec skenario 4). The gate keeps the route
// out of the router's uncovered set and rejects an admin or a session without a
// business role before the handler runs. The Register(r, auth) shape mirrors the
// other domain services, so Service holds no Authenticator.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer, httpx.RoleSubcontractor)
	r.Gated("POST /api/offers/{offerId}/accept", gate, s.handleAccept)
	s.registerWorkOrder(r, auth)
}

// handleAccept forms the agreement from an accepted offer (FR-034, FR-036). The
// route admits both business roles; the service loads the offer under a party
// guard so a caller who is neither party gets a 404, then rejects a caller
// trying to accept their own standing offer before running the R-04 allocation
// transaction.
func (s *Service) handleAccept(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	offerID, ok := parseUUID(r.PathValue("offerId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id penawaran tidak sah.")
		return
	}
	view, err := s.accept(r.Context(), acc.ID, offerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// allocationView is one period an order draws capacity from, matching the
// AvailabilityPeriod schema: remaining is capacity - allocated, forced to 0 when
// the week is marked full.
type allocationView struct {
	WeekStart  string `json:"week_start"`
	Capacity   int32  `json:"capacity"`
	Allocated  int32  `json:"allocated"`
	Remaining  int32  `json:"remaining"`
	MarkedFull bool   `json:"marked_full"`
}

// statusEntry is one row of the order's status history for WorkOrderDetail.
type statusEntry struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
	Note   *string   `json:"note"`
}

// paymentView is one PaymentRecord: a payment statement with no money amount
// (FR-040, FR-042). The accept path always returns an empty payments array
// because a fresh order has no statements yet; US5 (T056) is what records them.
type paymentView struct {
	PaymentID           string    `json:"payment_id"`
	Direction           string    `json:"direction"`
	Date                string    `json:"date"`
	DeclaredByProfileID string    `json:"declared_by_profile_id"`
	Note                *string   `json:"note"`
	CreatedAt           time.Time `json:"created_at"`
}

// paymentMismatch flags that the two parties' payment statements disagree
// (FR-043). The server computes it from the presence and dates of the statement
// rows, never from money (there is no amount to compare), so the client renders
// the notice from this rather than re-deriving it from the payments array. Kind
// is missing_counterpart when one side stated and the other has not, or
// date_differs when both stated but on different dates; DayDifference carries the
// absolute day gap and rides only on date_differs.
type paymentMismatch struct {
	Kind          string `json:"kind"`
	DayDifference *int   `json:"day_difference,omitempty"`
}

// workOrderView is the WorkOrderDetail response. It carries the state machine to
// the client via allowed_transitions and self_cancellable so the frontend renders
// buttons from the array instead of duplicating the machine (FR-039). At
// formation the order is "accepted": it may move to production, or be cancelled,
// or go to mediation, and it is self-cancellable (FR-066). payments is present
// but empty at formation; US5 (T056) populates it (FR-041..FR-043).
type workOrderView struct {
	WorkOrderID            string           `json:"work_order_id"`
	Status                 string           `json:"status"`
	BuyerProfileID         string           `json:"buyer_profile_id"`
	SubcontractorProfileID string           `json:"subcontractor_profile_id"`
	ProductItemID          string           `json:"product_item_id,omitempty"`
	Material               string           `json:"material,omitempty"`
	Quantity               int32            `json:"quantity"`
	Deadline               string           `json:"deadline"`
	TotalPrice             int64            `json:"total_price"`
	ReadinessLeadDays      int32            `json:"readiness_lead_days,omitempty"`
	ReadinessDeadline      string           `json:"readiness_deadline"`
	AllowedTransitions     []string         `json:"allowed_transitions"`
	SelfCancellable        bool             `json:"self_cancellable"`
	CanRecordPayment       bool             `json:"can_record_payment"`
	CanReview              bool             `json:"can_review"`
	AutoConfirmAt          *time.Time       `json:"auto_confirm_at"`
	Allocations            []allocationView `json:"allocations"`
	StatusHistory          []statusEntry    `json:"status_history"`
	Payments               []paymentView    `json:"payments"`
	PaymentMismatch        *paymentMismatch `json:"payment_mismatch"`
}

// accept forms the agreement for one accepted offer following research.md R-04.
// It resolves the offer and its parties, guards that the caller is a party and
// is not accepting their own standing offer (FR-033), rejects a request that
// already reached agreement or an offer whose readiness week would fall past
// the deadline, then in one transaction: locks the listing, grows the calendar to
// the deadline week (FR-088), locks every candidate period ascending by
// week_start, sums the remaining capacity across not-full periods and rejects a
// shortfall citing the actual figure (FR-035), inserts the work order storing its
// readiness week (FR-084), fills the earliest weeks first skipping full or
// exhausted periods (FR-018/FR-078) with one allocation row per used period
// (FR-077), marks the winning candidate agreed (the partial unique index makes a
// concurrent second agreement fail, FR-036), closes and notifies the other
// candidates (FR-034), and records the opening status. A failure on any period
// rolls the whole formation back.
func (s *Service) accept(ctx context.Context, accountID, offerID pgtype.UUID) (workOrderView, error) {
	row, err := s.queries().GetOfferForAccept(ctx, offerID)
	if err != nil {
		if isNoRows(err) {
			return workOrderView{}, &conflictError{code: httpx.CodeNotFound, detail: "Penawaran tidak ditemukan."}
		}
		return workOrderView{}, err
	}

	// The route admits both parties; this guards that the caller is a party to
	// this particular negotiation, so an unrelated account's offer id is a 404
	// rather than a leaked existence.
	isBuyer := row.BuyerAccount == accountID
	isSub := row.SubcontractorAccount == accountID
	if !isBuyer && !isSub {
		return workOrderView{}, &conflictError{code: httpx.CodeNotFound, detail: "Penawaran tidak ditemukan."}
	}

	// Closing the negotiation is the counterpart's approval of the standing
	// offer, not the proposer's (FR-033, spec skenario 4). The party who made the
	// standing offer cannot accept it themselves; they may only counter or wait.
	// This mirrors the alternation the counter path enforces.
	var caller sqlcgen.OfferParty
	if isSub {
		caller = sqlcgen.OfferPartySubcontractor
	} else {
		caller = sqlcgen.OfferPartyBuyer
	}
	if caller == row.ProposedBy {
		return workOrderView{}, &conflictError{
			code:   httpx.CodeForbidden,
			detail: "Anda tidak bisa menyetujui penawaran Anda sendiri; menunggu pihak lain menyetujui atau menawar balik.",
		}
	}

	// Only the standing (latest) offer of the chain can be accepted; accepting a
	// superseded round would agree to a price already countered (FR-033).
	if row.Sequence != row.LatestSequence {
		return workOrderView{}, &conflictError{
			code:   httpx.CodeRequestAlreadyAgreed,
			detail: "Penawaran ini sudah digantikan penawaran balik yang lebih baru.",
		}
	}

	if row.CandidateStatus != sqlcgen.CandidateStatusOffered {
		return workOrderView{}, &conflictError{
			code:   httpx.CodeRequestAlreadyAgreed,
			detail: "Kandidat ini tidak lagi dalam negosiasi terbuka.",
		}
	}

	// A request reaches at most one agreement. Detecting an existing agreement up
	// front gives the buyer the quotable REQUEST_ALREADY_AGREED reason without
	// waiting for the unique-index violation, which is the concurrency safety net.
	if row.RequestHasAgreement {
		return workOrderView{}, &conflictError{
			code:   httpx.CodeRequestAlreadyAgreed,
			detail: "Request ini sudah memiliki kesepakatan dengan kandidat lain.",
		}
	}

	now := s.clock.Now()
	readinessWeek := ReadinessDeadline(now, int(row.ReadinessLeadDays))
	deadlineWeek := platform.WeekStart(row.Deadline.Time)

	// No allocation may sit before the readiness week (FR-087); if readiness
	// already runs past the deadline week there is no room at all.
	if readinessWeek.After(deadlineWeek) {
		return workOrderView{}, &conflictError{
			code:   httpx.CodeReadinessAfterDeadline,
			detail: "Kesiapan produksi jatuh setelah tenggat pesanan, tidak ada minggu yang tersisa untuk dialokasikan.",
		}
	}

	// Pre-lock capacity read, using the same optimistic estimate as search for
	// weeks past the horizon (FR-080/FR-088). A shortfall here means the offer
	// never had enough capacity across the window, so reject fast with
	// INSUFFICIENT_CAPACITY without taking any lock. Passing here but coming up
	// short after the lock means the capacity was taken between the two reads,
	// which the transaction below reports as CAPACITY_ALREADY_TAKEN.
	est, err := s.queries().EstimateCapacityInRange(ctx, sqlcgen.EstimateCapacityInRangeParams{
		ListingID:     row.ListingID,
		ReadinessWeek: pgdate(readinessWeek),
		DeadlineWeek:  pgdate(deadlineWeek),
	})
	if err != nil {
		return workOrderView{}, err
	}
	estimated := est.RecordedRemaining + est.UncreatedRemaining
	if estimated < int64(row.Quantity) {
		return workOrderView{}, &metaError{
			code: httpx.CodeInsufficientCapacity,
			detail: "Kapasitas tersisa sampai tenggat hanya " + itoa64(estimated) +
				" potong, kurang dari " + itoa32(row.Quantity) + " potong yang dipesan.",
			meta: map[string]any{
				"quantity_requested": row.Quantity,
				"remaining_capacity": estimated,
				"until_week":         platform.FormatDate(deadlineWeek),
			},
		}
	}

	var view workOrderView
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		// Lock the listing so the horizon growth and the period range are taken
		// under the same lock the owner's calendar edits hold.
		if _, err := q.LockListingByID(ctx, row.ListingID); err != nil {
			return err
		}

		// FR-088: make sure every week up to the deadline exists before locking
		// the range, so a freshly agreed order is not short of periods the
		// calendar simply had not generated yet.
		if _, err := s.horizon.EnsureHorizon(ctx, tx, row.ListingID, deadlineWeek); err != nil {
			return err
		}

		periods, err := q.LockPeriodsInRange(ctx, sqlcgen.LockPeriodsInRangeParams{
			ListingID:   row.ListingID,
			WeekStart:   pgdate(readinessWeek),
			WeekStart_2: pgdate(deadlineWeek),
		})
		if err != nil {
			return err
		}

		// Sum the capacity actually available across the window: skip weeks marked
		// full, and count only the room left in each period. This is the figure
		// FR-035 quotes on a shortfall.
		var available int64
		for _, p := range periods {
			if p.MarkedFull {
				continue
			}
			remaining := p.TotalCapacity - p.UsedCapacity
			if remaining > 0 {
				available += int64(remaining)
			}
		}
		// The pre-lock estimate said there was room, but under the lock the range
		// is short: another agreement took the capacity between the two reads. This
		// is contention, not a lack of capacity from the start (FR-036), so it maps
		// to CAPACITY_ALREADY_TAKEN and suggests choosing another candidate or
		// moving the deadline, with the actual figure and deadline week in meta.
		if available < int64(row.Quantity) {
			return &metaError{
				code: httpx.CodeCapacityAlreadyTaken,
				detail: "Kapasitas untuk periode ini baru saja diambil kesepakatan lain, tersisa " +
					itoa64(available) + " potong dari " + itoa32(row.Quantity) +
					" potong yang dipesan. Pilih kandidat lain atau ubah tenggat pesanan.",
				meta: map[string]any{
					"quantity_requested": row.Quantity,
					"remaining_capacity": available,
					"until_week":         platform.FormatDate(deadlineWeek),
				},
			}
		}

		wo, err := q.InsertWorkOrder(ctx, sqlcgen.InsertWorkOrderParams{
			CandidateID:        row.CandidateID,
			OfferID:            row.OfferID,
			BuyerID:            row.BuyerID,
			SubcontractorID:    row.SubcontractorID,
			Quantity:           row.Quantity,
			TotalPrice:         row.TotalPrice,
			Deadline:           row.Deadline,
			ReadinessWeekStart: pgdate(readinessWeek),
			CreatedAt:          tstz(now),
		})
		if err != nil {
			return err
		}

		// Fill the earliest weeks first (FR-018), skipping full or exhausted
		// periods (FR-078). One allocation row per period actually used (FR-077);
		// RaiseUsedCapacity leans on used_capacity_within_total as the hard storage
		// guard (FR-079).
		remaining := row.Quantity
		var allocations []allocationView
		for _, p := range periods {
			if remaining == 0 {
				break
			}
			if p.MarkedFull {
				continue
			}
			room := p.TotalCapacity - p.UsedCapacity
			if room <= 0 {
				continue
			}
			take := room
			if take > remaining {
				take = remaining
			}

			if _, err := q.InsertAllocation(ctx, sqlcgen.InsertAllocationParams{
				WorkOrderID: wo.ID,
				PeriodID:    p.ID,
				Quantity:    take,
				CreatedAt:   tstz(now),
			}); err != nil {
				return err
			}
			raised, err := q.RaiseUsedCapacity(ctx, sqlcgen.RaiseUsedCapacityParams{
				ID:           p.ID,
				UsedCapacity: take,
				UpdatedAt:    tstz(now),
			})
			if err != nil {
				return err
			}
			remaining -= take
			allocations = append(allocations, allocationView{
				WeekStart:  platform.FormatDate(raised.WeekStart.Time),
				Capacity:   raised.TotalCapacity,
				Allocated:  raised.UsedCapacity,
				Remaining:  remainingCapacity(raised),
				MarkedFull: raised.MarkedFull,
			})
		}

		// Mark the winner agreed. idx_one_agreement_per_request is a partial unique
		// index on request_id where status='agreed', so two candidates of the SAME
		// request accepted concurrently collide here: the loser violates it. That is
		// the one-agreement-per-request rule (FR-034/FR-036), not capacity
		// contention, so the loser gets REQUEST_ALREADY_AGREED.
		if err := q.SetCandidateAgreed(ctx, sqlcgen.SetCandidateAgreedParams{
			ID:        row.CandidateID,
			UpdatedAt: tstz(now),
		}); err != nil {
			if isUniqueViolation(err) {
				return &conflictError{
					code:   httpx.CodeRequestAlreadyAgreed,
					detail: "Request ini sudah memiliki kesepakatan dengan kandidat lain.",
				}
			}
			return err
		}

		// Close the request's other candidates and let each subcontractor know the
		// request was agreed elsewhere (FR-034).
		others, err := q.ListOtherCandidatesToNotify(ctx, sqlcgen.ListOtherCandidatesToNotifyParams{
			RequestID: row.RequestID,
			ID:        row.CandidateID,
		})
		if err != nil {
			return err
		}
		if err := q.CloseOtherCandidates(ctx, sqlcgen.CloseOtherCandidatesParams{
			RequestID: row.RequestID,
			ID:        row.CandidateID,
			UpdatedAt: tstz(now),
		}); err != nil {
			return err
		}

		// Record the opening status transition (from nothing to accepted). The
		// accepting party is the actor; by_system is false.
		note := pgtype.Text{String: "Kesepakatan terbentuk.", Valid: true}
		if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
			WorkOrderID: wo.ID,
			OldStatus:   sqlcgen.NullWorkOrderStatus{},
			NewStatus:   wo.Status,
			ChangedBy:   accountID,
			BySystem:    false,
			Note:        note,
			CreatedAt:   tstz(now),
		}); err != nil {
			return err
		}

		// Notify both parties that the agreement formed (FR-034), then every losing
		// candidate's subcontractor. Either party may be the one who accepted, so
		// the wording follows the caller: the accepter reads that they approved,
		// the counterpart reads that their offer was approved. A notification row
		// is written in this transaction; delivery runs after commit.
		link := "/work-orders/" + uuidString(wo.ID)
		accepterMsg := "Anda menyetujui penawaran dan pesanan telah dibuat."
		proposerMsg := "Penawaran Anda disetujui dan pesanan telah dibuat."
		subMsg, buyerMsg := proposerMsg, accepterMsg
		if isSub {
			subMsg, buyerMsg = accepterMsg, proposerMsg
		}
		if err := s.notifier.Enqueue(ctx, tx, row.SubcontractorAccount,
			sqlcgen.EventTypeAgreementFormed,
			"Kesepakatan terbentuk",
			subMsg,
			&link); err != nil {
			return err
		}
		if err := s.notifier.Enqueue(ctx, tx, row.BuyerAccount,
			sqlcgen.EventTypeAgreementFormed,
			"Kesepakatan terbentuk",
			buyerMsg,
			&link); err != nil {
			return err
		}
		reqLink := "/quota-requests/" + uuidString(row.RequestID)
		for _, o := range others {
			if err := s.notifier.Enqueue(ctx, tx, o.SubcontractorAccount,
				sqlcgen.EventTypeAgreementFormed,
				"Request sudah disepakati",
				"Request kuota ini telah disepakati dengan kandidat lain.",
				&reqLink); err != nil {
				return err
			}
		}

		noteStr := note.String
		view = workOrderView{
			WorkOrderID:            uuidString(wo.ID),
			Status:                 string(wo.Status),
			BuyerProfileID:         uuidString(row.BuyerID),
			SubcontractorProfileID: uuidString(row.SubcontractorID),
			ProductItemID:          uuidString(row.ProductItemID),
			Material:               row.Material,
			Quantity:               wo.Quantity,
			Deadline:               platform.FormatDate(wo.Deadline.Time),
			TotalPrice:             wo.TotalPrice,
			ReadinessLeadDays:      row.ReadinessLeadDays,
			ReadinessDeadline:      platform.FormatDate(wo.ReadinessWeekStart.Time),
			AllowedTransitions:     allowedTransitions(wo.Status),
			SelfCancellable:        wo.Status == sqlcgen.WorkOrderStatusAccepted,
			// The accepting caller is a party, and a fresh order is 'accepted', a
			// status that still takes payment statements, so the caller may record
			// one right away (FR-041). It is not yet confirmed received, so it is
			// not reviewable, and with no statements there is no mismatch (FR-043,
			// FR-047).
			CanRecordPayment: true,
			CanReview:        false,
			AutoConfirmAt:    nil,
			Allocations:      allocations,
			StatusHistory: []statusEntry{
				{Status: string(wo.Status), At: now, Note: &noteStr},
			},
			Payments:        []paymentView{},
			PaymentMismatch: nil,
		}
		return nil
	})
	if err != nil {
		return workOrderView{}, err
	}
	return view, nil
}

// remainingCapacity is capacity - allocated, floored at zero and forced to zero
// when the week is marked full, matching the AvailabilityPeriod contract.
func remainingCapacity(p sqlcgen.AvailabilityPeriod) int32 {
	if p.MarkedFull {
		return 0
	}
	r := p.TotalCapacity - p.UsedCapacity
	if r < 0 {
		return 0
	}
	return r
}

// allowedTransitions returns the status moves that may follow the current one, so
// the client renders buttons from this array instead of re-implementing the
// machine (FR-039). The table is data-model.md section 7: the forward chain
// accepted -> production -> completed -> shipped -> confirmed, cancellation only
// before production, and mediation reachable from any active stage. Terminal
// states (confirmed, cancelled) have no outgoing move.
func allowedTransitions(status sqlcgen.WorkOrderStatus) []string {
	switch status {
	case sqlcgen.WorkOrderStatusAccepted:
		return []string{
			string(sqlcgen.WorkOrderStatusProduction),
			string(sqlcgen.WorkOrderStatusCancelled),
			string(sqlcgen.WorkOrderStatusInMediation),
		}
	case sqlcgen.WorkOrderStatusProduction:
		return []string{
			string(sqlcgen.WorkOrderStatusCompleted),
			string(sqlcgen.WorkOrderStatusInMediation),
		}
	case sqlcgen.WorkOrderStatusCompleted:
		return []string{
			string(sqlcgen.WorkOrderStatusShipped),
			string(sqlcgen.WorkOrderStatusInMediation),
		}
	case sqlcgen.WorkOrderStatusShipped:
		return []string{
			string(sqlcgen.WorkOrderStatusConfirmed),
			string(sqlcgen.WorkOrderStatusInMediation),
		}
	case sqlcgen.WorkOrderStatusInMediation:
		return []string{
			string(sqlcgen.WorkOrderStatusCancelled),
			string(sqlcgen.WorkOrderStatusConfirmed),
		}
	default:
		return []string{}
	}
}

// statusLabelID maps a work-order status to the Indonesian label the user sees.
// The INVALID_STATUS_TRANSITION detail and the ordered-sequence hint are composed
// from these labels, so the message reads in the interface language while the
// stored status stays the English machine value.
func statusLabelID(status sqlcgen.WorkOrderStatus) string {
	switch status {
	case sqlcgen.WorkOrderStatusAccepted:
		return "Diterima"
	case sqlcgen.WorkOrderStatusProduction:
		return "Produksi"
	case sqlcgen.WorkOrderStatusCompleted:
		return "Selesai"
	case sqlcgen.WorkOrderStatusShipped:
		return "Dikirim"
	case sqlcgen.WorkOrderStatusConfirmed:
		return "Dikonfirmasi"
	case sqlcgen.WorkOrderStatusCancelled:
		return "Dibatalkan"
	case sqlcgen.WorkOrderStatusInMediation:
		return "Dalam Mediasi"
	default:
		return string(status)
	}
}
