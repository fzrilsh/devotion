package order

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// paymentBody is the POST /payments request: which direction the party is
// stating (sent or received) plus the date they state and an optional free note.
// No money amount: the platform records only the statement, never the sum, and
// neither holds nor verifies funds (FR-040, FR-042).
type paymentBody struct {
	Direction string  `json:"direction"`
	Date      string  `json:"date"`
	Note      *string `json:"note"`
}

// registerPayment wires the payment-statement route. Either party to an order may
// record their own statement (FR-041), so the gate admits the buyer or the
// subcontractor role; the handler further guards that the caller is actually a
// party to this particular order.
func (s *Service) registerPayment(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer, httpx.RoleSubcontractor)
	r.Gated("POST /api/work-orders/{workOrderId}/payments", gate, s.handleCreatePayment)
}

// handleCreatePayment records one party's payment statement (FR-041). The route
// admits both business roles; the handler validates the direction, the date, and
// the optional note (<=500), then the service guards party membership and inserts
// the row. A party may state each direction at most once: the second attempt
// hits the one_statement_per_party_per_direction constraint and comes back as a
// readable PAYMENT_STATEMENT_EXISTS 409.
func (s *Service) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}
	var body paymentBody
	if !decodeJSON(w, r, &body) {
		return
	}

	direction, ok := parsePaymentDirection(body.Direction)
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Arah pembayaran tidak sah. Nilai yang diterima: sent, received.")
		return
	}
	date, err := platform.ParseDate(body.Date)
	if err != nil {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Tanggal pembayaran tidak sah. Format yang diterima: YYYY-MM-DD.")
		return
	}
	if body.Note != nil && len([]rune(*body.Note)) > 500 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Catatan pembayaran maksimal 500 karakter.")
		return
	}

	view, err := s.createPayment(r.Context(), acc.ID, id, direction, date, body.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// parsePaymentDirection accepts only the two directions a party may state: the
// money they say they sent, or the money they say they received.
func parsePaymentDirection(s string) (sqlcgen.PaymentDirection, bool) {
	switch sqlcgen.PaymentDirection(s) {
	case sqlcgen.PaymentDirectionSent:
		return sqlcgen.PaymentDirectionSent, true
	case sqlcgen.PaymentDirectionReceived:
		return sqlcgen.PaymentDirectionReceived, true
	default:
		return "", false
	}
}

// createPayment records a party's payment statement in one transaction. It locks
// the order, guards that the caller is one of its two parties (a non-party sees a
// 404, never a leak), resolves the caller's profile id (the statement is stored
// against the profile, not the account), inserts the row stamped from the Clock,
// then notifies the other party that a statement was recorded (FR-041). A repeat
// statement of the same direction by the same party violates the unique
// constraint and returns PAYMENT_STATEMENT_EXISTS. On success it reloads the full
// detail so the response carries the updated payments array.
func (s *Service) createPayment(ctx context.Context, accountID, workOrderID pgtype.UUID, direction sqlcgen.PaymentDirection, date time.Time, note *string) (workOrderView, error) {
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

		// The statement is stored against the party's business profile id, so the
		// two parties' statements stay distinguishable regardless of which account
		// device they came from.
		profileID, err := q.GetProfileIDByAccount(ctx, accountID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		var noteText pgtype.Text
		if note != nil {
			noteText = pgtype.Text{String: *note, Valid: true}
		}
		if _, err := q.InsertPaymentRecord(ctx, sqlcgen.InsertPaymentRecordParams{
			WorkOrderID: workOrderID,
			ProfileID:   profileID,
			Direction:   direction,
			Date:        pgdate(date),
			Note:        noteText,
			CreatedAt:   tstz(now),
		}); err != nil {
			if isPaymentStatementDuplicate(err) {
				return &conflictError{
					code: httpx.CodePaymentStatementExists,
					detail: "Anda sudah mencatat pernyataan dengan arah yang sama pada pesanan ini. " +
						"Setiap pihak mencatat tiap arah paling banyak sekali.",
				}
			}
			return err
		}

		// Notify the other party that a statement was recorded (FR-041). A payment
		// statement is its own FR-051 event; the notice goes to whichever account is
		// not the caller.
		other := view.BuyerAccount
		if accountID == view.BuyerAccount {
			other = view.SubcontractorAccount
		}
		link := "/work-orders/" + uuidString(workOrderID)
		if err := s.notifier.Enqueue(ctx, tx, other,
			sqlcgen.EventTypePaymentRecord,
			"Pernyataan pembayaran dicatat",
			"Pihak lain mencatat pernyataan pembayaran ("+paymentDirectionLabelID(direction)+") pada pesanan Anda.",
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
	return s.buildDetailView(ctx, row, accountID)
}

// isPaymentStatementDuplicate reports whether err is the unique-violation on the
// one_statement_per_party_per_direction constraint, the signal that this party
// already stated this direction on this order (FR-041).
func isPaymentStatementDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "one_statement_per_party_per_direction"
}

// paymentDirectionLabelID renders a payment direction in Indonesian for notices.
func paymentDirectionLabelID(d sqlcgen.PaymentDirection) string {
	switch d {
	case sqlcgen.PaymentDirectionSent:
		return "dana dikirim"
	case sqlcgen.PaymentDirectionReceived:
		return "dana diterima"
	default:
		return string(d)
	}
}
