package search

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// regionLevel is the coverage scope of one search: exactly one of the three
// levels FR-063 defines. The handler validates the raw query enum into this.
type regionLevel string

const (
	regionCity     regionLevel = "city"
	regionProvince regionLevel = "province"
	regionNational regionLevel = "national"
)

// searchQuery holds the validated inputs of one search request. Optional filters
// are pointers so "unsupplied" (counts as satisfied, FR-023 C-4) is distinct
// from a zero value. deadline is already the raw wire date; the service rounds
// it to the Monday week boundary.
type searchQuery struct {
	productItem  *pgtype.UUID
	machineItem  *pgtype.UUID
	quantity     int32
	deadline     time.Time
	maxLead      *int32
	region       regionLevel
	cityCode     string
	provinceCode string
	cursor       *cursor
	size         int32
}

// criterion is one of the four hard criteria as reported to the buyer (FR-026):
// its name, whether it was met, and an optional Indonesian detail. A criterion
// whose filter was not supplied is reported met with a "tidak dinilai" detail so
// the buyer sees it was skipped rather than passed on merit.
type criterion struct {
	Name   string  `json:"name"`
	Met    bool    `json:"met"`
	Detail *string `json:"detail"`
}

// candidateView is one SearchCandidate. Field names follow the contract.
type candidateView struct {
	ListingID                  string      `json:"listing_id"`
	ProfileID                  string      `json:"profile_id"`
	BusinessName               string      `json:"business_name"`
	Score                      int32       `json:"score"`
	TotalCapacityUntilDeadline int64       `json:"total_capacity_until_deadline"`
	DistanceKm                 *float64    `json:"distance_km"`
	IdentityVerified           bool        `json:"identity_verified"`
	Criteria                   []criterion `json:"criteria"`
}

// pagination is the keyset page marker. next_cursor is opaque and nil when there
// is no next page (FR-080).
type pagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// relaxationView names the most restrictive filter and a concrete relaxation,
// present only when a search at the national level returned nothing (FR-028).
type relaxationView struct {
	MostRestrictive string `json:"most_restrictive"`
	Suggestion      string `json:"suggestion"`
}

// searchResult is the SearchResult body plus the search-owned view fields the
// contract leaves to this package: region_level (FR-063, the level actually
// used) and relaxation (FR-028, saran pelonggaran when empty at national).
type searchResult struct {
	Items       []candidateView `json:"items"`
	Pagination  pagination      `json:"pagination"`
	RegionLevel regionLevel     `json:"region_level"`
	Relaxation  *relaxationView `json:"relaxation,omitempty"`
}

// search runs one ranked page for the searcher account. It resolves the
// searcher's own profile (excluded from results, FR-081), builds the query
// params from the injected Clock's date (Rule 5, never time.Now), runs the read
// query, maps rows to the response, then extends the horizon of passing
// candidates in a separate transaction outside the read (FR-088).
func (s *Service) search(ctx context.Context, accountID pgtype.UUID, q searchQuery) (searchResult, error) {
	searcherProfile, err := s.queries().GetProfileIDByAccount(ctx, accountID)
	if err != nil && !isNoRows(err) {
		return searchResult{}, err
	}

	searchDate := platform.WeekStart(s.clock.Now())
	deadlineWeek := platform.WeekStart(q.deadline)

	params := sqlcgen.SearchCandidatesParams{
		SearchDate:      pgtype.Date{Time: searchDate, Valid: true},
		DeadlineWeek:    pgtype.Date{Time: deadlineWeek, Valid: true},
		Quantity:        q.quantity,
		SearcherProfile: searcherProfile,
		PageSize:        q.size + 1, // one extra row decides has_next
	}
	if q.productItem != nil {
		params.ProductItem = *q.productItem
	}
	if q.machineItem != nil {
		params.MachineItem = *q.machineItem
	}
	if q.maxLead != nil {
		params.MaxLead = pgtype.Int4{Int32: *q.maxLead, Valid: true}
	}
	switch q.region {
	case regionCity:
		params.CityCode = pgtype.Text{String: q.cityCode, Valid: true}
	case regionProvince:
		params.ProvinceCode = pgtype.Text{String: q.provinceCode, Valid: true}
	case regionNational:
		// no region filter
	}

	// The keyset cursor: the first page starts above every real row. Score tops
	// out at 4, remaining_capacity and lead are bounded by the schema, so a
	// sentinel above the maxima makes the strict "<" tuple comparison admit the
	// whole set. cursor_neg_lead is -lead, so its ceiling is 1 (above 0 == lead
	// 0). An empty business_name and the nil UUID sort last, but a first-page
	// name ceiling must exceed any real name; ￿ runes do that.
	if q.cursor == nil {
		params.CursorScore = 5
		params.CursorRemaining = 1 << 62
		params.CursorNegLead = 1
		params.CursorName = "￿"
		params.CursorListing = maxUUID()
	} else {
		params.CursorScore = q.cursor.Score
		params.CursorRemaining = q.cursor.Remaining
		params.CursorNegLead = -q.cursor.LeadDays
		params.CursorName = q.cursor.Name
		cl, _ := parseUUID(q.cursor.Listing)
		params.CursorListing = cl
	}

	rows, err := s.queries().SearchCandidates(ctx, params)
	if err != nil {
		return searchResult{}, err
	}

	hasNext := int32(len(rows)) > q.size
	if hasNext {
		rows = rows[:q.size]
	}

	items := make([]candidateView, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.viewOf(row, q))
	}

	res := searchResult{
		Items:       items,
		Pagination:  pagination{HasNext: hasNext},
		RegionLevel: q.region,
	}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		tok := encodeCursor(cursor{
			Score:     last.Score,
			Remaining: last.RemainingCapacity,
			LeadDays:  last.ReadinessLeadDays,
			Name:      last.BusinessName,
			Listing:   uuidString(last.ListingID),
		})
		res.Pagination.NextCursor = &tok
	}

	// FR-028: nothing at the highest level names the most restrictive filter and
	// a concrete relaxation. Only at national, the last level, and only the
	// first page (no cursor) so a mid-walk empty tail is not mistaken for empty.
	if len(items) == 0 && q.region == regionNational && q.cursor == nil {
		res.Relaxation = relaxationOf(q)
	}

	// FR-088: extend the calendar of passing candidates so a later booking finds
	// real periods, not just the optimistic uncreated count. This is a write, so
	// it runs in its own transaction outside the read query and never blocks the
	// response on failure of the read path.
	s.extendHorizons(ctx, rows, deadlineWeek)

	return res, nil
}

// viewOf maps one row to a candidate, building the four-criterion explanation.
// A filter that was not supplied is reported met with a "tidak dinilai" detail
// (FR-023 C-4, FR-026): the score already counts it satisfied, and the buyer is
// told it was skipped rather than judged.
func (s *Service) viewOf(row sqlcgen.SearchCandidatesRow, q searchQuery) candidateView {
	notEvaluated := "Tidak dinilai karena filter ini tidak diisi."
	crit := make([]criterion, 0, 4)

	crit = append(crit, matchCriterion("Produk cocok", row.ProductMatch == 1, q.productItem != nil, notEvaluated))
	crit = append(crit, matchCriterion("Mesin cocok", row.MachineMatch == 1, q.machineItem != nil, notEvaluated))
	crit = append(crit, matchCriterion("Waktu ancang-ancang memenuhi", row.LeadMatch == 1, q.maxLead != nil, notEvaluated))
	crit = append(crit, matchCriterion("Kapasitas mencukupi", row.CapacityEnough == 1, true, notEvaluated))

	return candidateView{
		ListingID:                  uuidString(row.ListingID),
		ProfileID:                  uuidString(row.ProfileID),
		BusinessName:               row.BusinessName,
		Score:                      row.Score,
		TotalCapacityUntilDeadline: row.RemainingCapacity,
		IdentityVerified:           row.Verified,
		Criteria:                   crit,
	}
}

// matchCriterion builds one criterion. When the filter was not supplied it is
// reported met with the "tidak dinilai" detail; otherwise met carries the row's
// result and detail stays nil.
func matchCriterion(name string, met, evaluated bool, notEvaluated string) criterion {
	if !evaluated {
		d := notEvaluated
		return criterion{Name: name, Met: true, Detail: &d}
	}
	return criterion{Name: name, Met: met}
}

// relaxationOf picks the most restrictive filter to relax and a concrete
// suggestion. Capacity and deadline are the levers that most often empty a
// national search, so quantity is named first, then the deadline, then the two
// item filters, then lead time.
func relaxationOf(q searchQuery) *relaxationView {
	switch {
	case q.maxLead != nil:
		return &relaxationView{
			MostRestrictive: "Waktu ancang-ancang maksimal",
			Suggestion:      "Naikkan batas waktu ancang-ancang atau hapus filter ini.",
		}
	case q.quantity > 0:
		return &relaxationView{
			MostRestrictive: "Jumlah dan tenggat",
			Suggestion:      "Kurangi jumlah pesanan atau mundurkan tenggat agar lebih banyak kapasitas terjangkau.",
		}
	default:
		return &relaxationView{
			MostRestrictive: "Filter produk atau mesin",
			Suggestion:      "Longgarkan pilihan produk atau mesin agar lebih banyak kandidat masuk.",
		}
	}
}

// extendHorizons raises the calendar horizon of each returned candidate up to
// the deadline week, in a single transaction outside the read query (FR-088).
// It is best-effort: a failure here does not fail the search, because the read
// already answered the buyer and the extension only pre-materializes weeks a
// future booking would create anyway.
func (s *Service) extendHorizons(ctx context.Context, rows []sqlcgen.SearchCandidatesRow, deadlineWeek time.Time) {
	if len(rows) == 0 {
		return
	}
	_ = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		for _, row := range rows {
			if row.HorizonUntil.Valid && !row.HorizonUntil.Time.Before(deadlineWeek) {
				continue
			}
			if _, err := s.horizon.EnsureHorizon(ctx, tx, row.ListingID, deadlineWeek); err != nil {
				return err
			}
		}
		return nil
	})
}

// maxUUID is the all-ones UUID, the ceiling of the keyset's final tiebreaker so
// a first page admits every real listing_id under the strict "<" comparison.
func maxUUID() pgtype.UUID {
	var u pgtype.UUID
	for i := range u.Bytes {
		u.Bytes[i] = 0xff
	}
	u.Valid = true
	return u
}
