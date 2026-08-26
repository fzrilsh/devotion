package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// disputeBody is the POST /disputes request: the reporting party's account of
// what went wrong, which admin reads when mediating (FR-046). report_body is
// 10..2000 characters.
type disputeBody struct {
	ReportBody string `json:"report_body"`
}

// registerDispute wires the dispute-reporting route. Either party to an order
// may report a dispute (FR-046), so the gate admits the buyer or the
// subcontractor role; the handler further guards that the caller is actually a
// party to this particular order.
func (s *Service) registerDispute(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer, httpx.RoleSubcontractor)
	r.Gated("POST /api/work-orders/{workOrderId}/disputes", gate, s.handleCreateDispute)
}

// handleCreateDispute opens a dispute on a work order (FR-046, FR-070). The
// route admits both business roles; the handler validates the report body
// (10..2000), then the service guards party membership and inserts the dispute.
// At most one dispute may be open per order: a second report while one is still
// unresolved comes back as a readable DISPUTE_ALREADY_OPEN 409.
func (s *Service) handleCreateDispute(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}
	var body disputeBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if n := len([]rune(body.ReportBody)); n < 10 || n > 2000 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Laporan sengketa antara 10 dan 2000 karakter.")
		return
	}

	view, err := s.createDispute(r.Context(), acc.ID, id, body.ReportBody)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// createDispute records a party's dispute report in one transaction (FR-046,
// FR-070). It locks the order, guards that the caller is one of its two parties
// (a non-party sees a 404, never a leak), resolves the caller's profile id (the
// dispute is stored against the reporting profile), and inserts the dispute row.
// The work order keeps its current status: the open dispute row itself stops the
// 7-day auto-confirm count (ListShippedDueForAutoConfirm excludes any order with
// an unresolved dispute, FR-070), so there is no need to move the order off
// 'shipped' here. Admin moves it to in_mediation when picking up the case (T071).
// A second report while one dispute is still open violates the idx_one_open_dispute
// partial unique index and returns DISPUTE_ALREADY_OPEN. confirm_warn_sent_at is
// deliberately left untouched (data-model.md). The other party is notified. On
// success it reloads the full detail so the response carries the current status.
func (s *Service) createDispute(ctx context.Context, accountID, workOrderID pgtype.UUID, reportBody string) (workOrderView, error) {
	now := s.clock.Now()
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		if _, err := q.LockWorkOrderForStatusChange(ctx, workOrderID); err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		// Resolve both parties' accounts for the party guard and the notice.
		view, err := q.GetWorkOrderForView(ctx, workOrderID)
		if err != nil {
			return err
		}
		if view.BuyerAccount != accountID && view.SubcontractorAccount != accountID {
			return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
		}

		// reporter_id stores the reporting party's business profile id, so admin
		// can see which side opened the dispute when mediating (FR-046).
		profileID, err := q.GetProfileIDByAccount(ctx, accountID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		if _, err := q.InsertDispute(ctx, sqlcgen.InsertDisputeParams{
			WorkOrderID: workOrderID,
			ReporterID:  profileID,
			ReportBody:  reportBody,
			CreatedAt:   tstz(now),
		}); err != nil {
			if isDisputeAlreadyOpen(err) {
				return &conflictError{
					code: httpx.CodeDisputeAlreadyOpen,
					detail: "Sengketa pada pesanan ini masih terbuka. " +
						"Tunggu admin menyelesaikannya sebelum melaporkan lagi.",
				}
			}
			return err
		}

		// Notify the other party that a dispute was opened (FR-051 "sengketa"). The
		// reporter is one of the two accounts; the notice goes to whichever is not
		// the caller.
		other := view.BuyerAccount
		if accountID == view.BuyerAccount {
			other = view.SubcontractorAccount
		}
		link := "/work-orders/" + uuidString(workOrderID)
		if err := s.notifier.Enqueue(ctx, tx, other,
			sqlcgen.EventTypeOrderStatusChanged,
			"Sengketa dilaporkan",
			"Pihak lain melaporkan sengketa pada pesanan Anda. Admin akan menengahi.",
			&link); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return workOrderView{}, err
	}

	row, err := s.queries().GetWorkOrderForView(ctx, workOrderID)
	if err != nil {
		return workOrderView{}, err
	}
	return s.buildDetailView(ctx, row)
}

// isDisputeAlreadyOpen reports whether err is the unique-violation on the
// idx_one_open_dispute partial index, the signal that this order already has an
// unresolved dispute (FR-046).
func isDisputeAlreadyOpen(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "idx_one_open_dispute"
}
