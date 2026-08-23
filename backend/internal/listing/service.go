package listing

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// profileID resolves the caller's profile from the authenticated account. A
// profile has existed since registration (T026), so a missing row is a genuine
// invariant break reported as errProfileMissing rather than a 404 the owner can
// act on. It takes a Queries so it works on the pool or inside a transaction.
func (s *Service) profileID(ctx context.Context, q *sqlcgen.Queries, accountID pgtype.UUID) (pgtype.UUID, error) {
	pid, err := q.GetProfileIDByAccount(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			return pgtype.UUID{}, errProfileMissing
		}
		return pgtype.UUID{}, err
	}
	return pid, nil
}

// getListing loads the caller's listing for GET /listing/me. No listing yet is
// a 404 the owner resolves by creating one.
func (s *Service) getListing(ctx context.Context, accountID pgtype.UUID) (listingView, error) {
	q := s.queries()
	profileID, err := s.profileID(ctx, q, accountID)
	if err != nil {
		return listingView{}, err
	}
	l, err := q.GetListingByProfile(ctx, profileID)
	if err != nil {
		if isNoRows(err) {
			return listingView{}, errListingNotFound
		}
		return listingView{}, err
	}
	return s.loadView(ctx, q, l)
}

// createListing publishes the profile's single listing (FR-010: it goes live
// immediately, no verification gate) and seeds a calendar reaching at least
// InitialHorizonWeeks ahead. Everything runs in one transaction so a listing is
// never left without its calendar or its items. Catalog items are checked
// before any insert, turning a wrong-type id into a 422 instead of the
// trigger's bare 500.
func (s *Service) createListing(ctx context.Context, accountID pgtype.UUID, in listingInput) (listingView, error) {
	if verr := s.validateListingInput(in); verr != nil {
		return listingView{}, verr
	}

	var view listingView
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		profileID, err := s.profileID(ctx, q, accountID)
		if err != nil {
			return err
		}

		if _, err := q.GetListingByProfile(ctx, profileID); err == nil {
			return errListingExists
		} else if !isNoRows(err) {
			return err
		}

		if verr := s.checkCatalogItems(ctx, q, in); verr != nil {
			return verr
		}

		now := s.clock.Now()
		weekNow := platform.WeekStart(now)
		l, err := q.CreateListing(ctx, sqlcgen.CreateListingParams{
			ProfileID:         profileID,
			WeeklyCapacity:    in.WeeklyCapacity,
			ReadinessLeadDays: in.ReadinessLeadDays,
			CalendarUpdatedAt: tstz(now),
			HorizonUntil:      pgdate(weekNow),
		})
		if err != nil {
			return err
		}

		if err := s.writeItems(ctx, q, l.ID, in); err != nil {
			return err
		}

		until := weekNow.AddDate(0, 0, 7*InitialHorizonWeeks)
		if _, err := s.EnsureHorizon(ctx, tx, l.ID, until); err != nil {
			return err
		}

		// EnsureHorizon raised horizon_until on the row, so the copy CreateListing
		// returned is stale. Re-read it before building the view.
		l, err = q.GetListingByID(ctx, l.ID)
		if err != nil {
			return err
		}

		view, err = s.loadView(ctx, q, l)
		return err
	})
	if err != nil {
		return listingView{}, err
	}
	return view, nil
}

// updateListing edits the owner-editable fields and propagates the new weekly
// capacity to future periods (FR-089). The listing row is locked first (fixed
// order: listing before availability_period), then any future period whose used
// capacity already exceeds the proposed value rejects the whole request with a
// 409 that names the week. Allocated periods keep their agreed number; only
// weeks with no active allocation take the new capacity. calendar_updated_at is
// untouched here: only the periods path advances it (FR-021).
func (s *Service) updateListing(ctx context.Context, accountID pgtype.UUID, in listingInput) (listingView, error) {
	if verr := s.validateListingInput(in); verr != nil {
		return listingView{}, verr
	}

	var view listingView
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		profileID, err := s.profileID(ctx, q, accountID)
		if err != nil {
			return err
		}
		l, err := q.LockListingByProfile(ctx, profileID)
		if err != nil {
			if isNoRows(err) {
				return errListingNotFound
			}
			return err
		}

		if verr := s.checkCatalogItems(ctx, q, in); verr != nil {
			return verr
		}

		now := s.clock.Now()
		weekNow := platform.WeekStart(now)

		over, err := q.FindFutureAllocatedPeriodOverCapacity(ctx, sqlcgen.FindFutureAllocatedPeriodOverCapacityParams{
			ListingID:    l.ID,
			UsedCapacity: in.WeeklyCapacity,
			WeekStart:    pgdate(weekNow),
		})
		if err != nil && !isNoRows(err) {
			return err
		}
		if err == nil {
			return &conflictError{
				code:   httpx.CodeCapacityAlreadyAllocated,
				detail: "Kapasitas tidak dapat diturunkan ke " + itoa32(in.WeeklyCapacity) + " potong. Minggu " + platform.FormatDateID(over.WeekStart.Time) + " sudah memakai " + itoa32(over.UsedCapacity) + " potong untuk pesanan berjalan.",
				week:   over.WeekStart.Time,
				used:   over.UsedCapacity,
				want:   in.WeeklyCapacity,
			}
		}

		if err := q.PropagateCapacityToFuturePeriods(ctx, sqlcgen.PropagateCapacityToFuturePeriodsParams{
			ListingID:     l.ID,
			TotalCapacity: in.WeeklyCapacity,
			WeekStart:     pgdate(weekNow),
			UpdatedAt:     tstz(now),
		}); err != nil {
			return err
		}

		l, err = q.UpdateListing(ctx, sqlcgen.UpdateListingParams{
			ID:                l.ID,
			WeeklyCapacity:    in.WeeklyCapacity,
			ReadinessLeadDays: in.ReadinessLeadDays,
			UpdatedAt:         tstz(now),
		})
		if err != nil {
			return err
		}

		if err := q.DeleteListingProducts(ctx, l.ID); err != nil {
			return err
		}
		if err := q.DeleteListingMachines(ctx, l.ID); err != nil {
			return err
		}
		if err := s.writeItems(ctx, q, l.ID, in); err != nil {
			return err
		}

		view, err = s.loadView(ctx, q, l)
		return err
	})
	if err != nil {
		return listingView{}, err
	}
	return view, nil
}

// setVisibility flips published for PUT /listing/me/visibility (FR-015). A
// hidden listing keeps its calendar and allocations; it merely drops out of
// search until re-enabled.
func (s *Service) setVisibility(ctx context.Context, accountID pgtype.UUID, published bool) (listingView, error) {
	q := s.queries()
	profileID, err := s.profileID(ctx, q, accountID)
	if err != nil {
		return listingView{}, err
	}
	l, err := q.GetListingByProfile(ctx, profileID)
	if err != nil {
		if isNoRows(err) {
			return listingView{}, errListingNotFound
		}
		return listingView{}, err
	}
	l, err = q.SetListingPublished(ctx, sqlcgen.SetListingPublishedParams{
		ID:        l.ID,
		Published: published,
		UpdatedAt: tstz(s.clock.Now()),
	})
	if err != nil {
		return listingView{}, err
	}
	return s.loadView(ctx, q, l)
}

// writeItems inserts the product and machine links of a listing. Callers run it
// inside a transaction after checkCatalogItems has confirmed every id.
func (s *Service) writeItems(ctx context.Context, q *sqlcgen.Queries, listingID pgtype.UUID, in listingInput) error {
	for _, id := range in.ProductItemIDs {
		u, _ := parseUUID(id)
		if err := q.InsertListingProduct(ctx, sqlcgen.InsertListingProductParams{
			ListingID: listingID,
			ItemID:    u,
		}); err != nil {
			return err
		}
	}
	for _, m := range in.Machines {
		u, _ := parseUUID(m.ItemID)
		if err := q.InsertListingMachine(ctx, sqlcgen.InsertListingMachineParams{
			ListingID:    listingID,
			ItemID:       u,
			MachineCount: m.MachineCount,
		}); err != nil {
			return err
		}
	}
	return nil
}

// checkCatalogItems confirms every product id is an active product and every
// machine id an active machine, before any insert. Without it, a machine id
// sent as a product would trip trg_reject_wrong_product_item and surface as a
// 500; here a shortfall is a 422 naming the offending field.
func (s *Service) checkCatalogItems(ctx context.Context, q *sqlcgen.Queries, in listingInput) error {
	var fields []httpx.FieldError

	productIDs := uniqueUUIDs(in.ProductItemIDs)
	if len(productIDs) > 0 {
		n, err := q.CountActiveCatalogItemsOfType(ctx, sqlcgen.CountActiveCatalogItemsOfTypeParams{
			Type:    sqlcgen.ItemTypeProduct,
			Column2: productIDs,
		})
		if err != nil {
			return err
		}
		if int(n) != len(productIDs) {
			fields = append(fields, httpx.FieldError{
				Field:   "product_item_ids",
				Message: "Ada produk yang tidak dikenal atau tidak aktif.",
			})
		}
	}

	machineStrings := make([]string, len(in.Machines))
	for i, m := range in.Machines {
		machineStrings[i] = m.ItemID
	}
	machineIDs := uniqueUUIDs(machineStrings)
	if len(machineIDs) > 0 {
		n, err := q.CountActiveCatalogItemsOfType(ctx, sqlcgen.CountActiveCatalogItemsOfTypeParams{
			Type:    sqlcgen.ItemTypeMachine,
			Column2: machineIDs,
		})
		if err != nil {
			return err
		}
		if int(n) != len(machineIDs) {
			fields = append(fields, httpx.FieldError{
				Field:   "machines",
				Message: "Ada mesin yang tidak dikenal atau tidak aktif.",
			})
		}
	}

	if len(fields) > 0 {
		return &validationError{fields: fields}
	}
	return nil
}

// loadView assembles the full listing response from the row plus its product
// and machine links.
func (s *Service) loadView(ctx context.Context, q *sqlcgen.Queries, l sqlcgen.CapacityListing) (listingView, error) {
	products, err := q.ListListingProducts(ctx, l.ID)
	if err != nil {
		return listingView{}, err
	}
	machines, err := q.ListListingMachines(ctx, l.ID)
	if err != nil {
		return listingView{}, err
	}
	return newListingView(l, products, machines), nil
}
