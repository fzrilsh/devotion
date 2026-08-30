package search

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/listing"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// noopNotifier satisfies listing.Notifier without a queue; the search suite only
// borrows the real listing service as a horizon extender, never its reminder job.
type noopNotifier struct{}

func (noopNotifier) Enqueue(_ context.Context, _ pgx.Tx, _ pgtype.UUID, _ sqlcgen.EventType, _, _ string, _ *string) error {
	return nil
}

// useRealHorizon rewires the harness Service to use the real listing.Service as
// its horizon extender, so SC-021 can prove the passing candidate's periods are
// actually materialized past the initial horizon (FR-088), not just recorded.
func useRealHorizon(t *testing.T, h *harness) {
	t.Helper()
	ext := listing.New(h.pool, h.clock, noopNotifier{})
	h.svc = New(h.pool, h.clock, ext)
	r := httpx.NewRouter(quietLogger())
	h.svc.Register(r, h.auth)
	h.handler = r.Handler()
}

// itemByListing returns the candidate whose listing_id matches, failing the test
// when the candidate is absent from the page.
func itemByListing(t *testing.T, res resultResp, listingID pgtype.UUID) candidateResp {
	t.Helper()
	want := uuidString(listingID)
	for _, it := range res.Items {
		if it.ListingID == want {
			return it
		}
	}
	t.Fatalf("listing %s tidak ada di hasil", want)
	return candidateResp{}
}

// criterionByName returns the named criterion of a candidate, failing when the
// candidate carries no criterion by that name. The return type matches the
// anonymous element type of candidateResp.Criteria.
func criterionByName(t *testing.T, c candidateResp, name string) struct {
	Name   string  `json:"name"`
	Met    bool    `json:"met"`
	Detail *string `json:"detail"`
} {
	t.Helper()
	for _, cr := range c.Criteria {
		if cr.Name == name {
			return cr
		}
	}
	t.Fatalf("kriteria %q tidak ada pada kandidat %s", name, c.ListingID)
	return c.Criteria[0]
}

// orderIDs projects a result page to its listing_id order, for comparing two
// runs of the same query.
func orderIDs(res resultResp) []string {
	ids := make([]string, 0, len(res.Items))
	for _, it := range res.Items {
		ids = append(ids, it.ListingID)
	}
	return ids
}

// profileOfListing reads the business_profile id that owns a listing, so a test
// can mutate the owner's verification or reviews.
func profileOfListing(t *testing.T, h *harness, listingID pgtype.UUID) pgtype.UUID {
	t.Helper()
	var prof pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT profile_id FROM capacity_listing WHERE id = $1`, listingID).Scan(&prof); err != nil {
		t.Fatalf("baca profile listing: %v", err)
	}
	return prof
}
