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

// reputationView is the Reputation schema as carried on a search candidate. It
// is computed at read time from SearchReputation, never materialized to a column
// (data-model.md section 19). completion_rate is nil until enough_data is true
// (FR-073), so the client never shows a raw percentage below the threshold.
type reputationView struct {
	EnoughData     bool     `json:"enough_data"`
	CompletionRate *int     `json:"completion_rate"`
	AverageRating  *float64 `json:"average_rating"`
	ReviewCount    int      `json:"review_count"`
}

// pageReputation bundles a profile's read-time reputation block with its
// completed-jobs count, which sits on the candidate itself (FR-048) rather than
// inside the Reputation schema.
type pageReputation struct {
	Reputation    reputationView
	CompletedJobs int64
}

// candidateView is one SearchCandidate. Field names follow the contract. The
// reputation block, completed_jobs, and stale_calendar are informative and do
// not influence the score (FR-034); stale_calendar and distance never reorder.
type candidateView struct {
	ListingID                  string         `json:"listing_id"`
	ProfileID                  string         `json:"profile_id"`
	BusinessName               string         `json:"business_name"`
	Score                      int32          `json:"score"`
	CityCode                   *string        `json:"city_code"`
	CityName                   *string        `json:"city_name"`
	MachineTypes               []string       `json:"machine_types"`
	WeeklyCapacity             int32          `json:"weekly_capacity"`
	ReadinessWeek              string         `json:"readiness_week"`
	ReadinessLeadDays          int32          `json:"readiness_lead_days"`
	TotalCapacityUntilDeadline int64          `json:"total_capacity_until_deadline"`
	CompletedJobs              int64          `json:"completed_jobs"`
	Reputation                 reputationView `json:"reputation"`
	StaleCalendar              bool           `json:"stale_calendar"`
	DistanceKm                 *float64       `json:"distance_km"`
	IdentityVerified           bool           `json:"identity_verified"`
	Criteria                   []criterion    `json:"criteria"`
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

	// Reputation is read at query time over the page's profiles, never stored as
	// a column (data-model.md section 19). One query for the whole page avoids an
	// N+1; a failure here does not fail the search, the block just stays empty.
	repByProfile := s.reputationForPage(ctx, rows)

	// staleBefore is the cutoff for FR-021: a calendar not touched in seven days
	// is marked stale. Derived from the injected Clock, never time.Now (Rule 5).
	staleBefore := s.clock.Now().AddDate(0, 0, -7)

	items := make([]candidateView, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.viewOf(row, q, repByProfile[uuidString(row.ProfileID)], staleBefore))
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

// viewOf maps one row to a candidate, building the four-criterion explanation
// and the informative attributes (FR-027): city, machine types, weekly and
// remaining capacity, readiness, reputation, and the stale-calendar marker
// (FR-021). rep is the read-time reputation for this profile, already thresholded
// by FR-073. A filter that was not supplied is reported met with a "tidak
// dinilai" detail (FR-023 C-4, FR-026): the score already counts it satisfied,
// and the buyer is told it was skipped rather than judged.
func (s *Service) viewOf(row sqlcgen.SearchCandidatesRow, q searchQuery, rep pageReputation, staleBefore time.Time) candidateView {
	notEvaluated := "Tidak dinilai karena filter ini tidak diisi."
	crit := make([]criterion, 0, 4)

	crit = append(crit, matchCriterion("Produk cocok", row.ProductMatch == 1, q.productItem != nil, notEvaluated))
	crit = append(crit, matchCriterion("Mesin cocok", row.MachineMatch == 1, q.machineItem != nil, notEvaluated))
	crit = append(crit, matchCriterion("Waktu ancang-ancang memenuhi", row.LeadMatch == 1, q.maxLead != nil, notEvaluated))
	crit = append(crit, matchCriterion("Kapasitas mencukupi", row.CapacityEnough == 1, true, notEvaluated))

	machineTypes := row.MachineTypes
	if machineTypes == nil {
		machineTypes = []string{}
	}

	// FR-021: a calendar not updated within seven days is flagged stale. The
	// marker is informative and never reorders results.
	stale := row.CalendarUpdatedAt.Valid && row.CalendarUpdatedAt.Time.Before(staleBefore)

	return candidateView{
		ListingID:                  uuidString(row.ListingID),
		ProfileID:                  uuidString(row.ProfileID),
		BusinessName:               row.BusinessName,
		Score:                      row.Score,
		CityCode:                   strPtrText(row.CityCode),
		CityName:                   pgTextPtr(row.CityName),
		MachineTypes:               machineTypes,
		WeeklyCapacity:             row.WeeklyCapacity,
		ReadinessWeek:              dateString(row.ReadinessWeek),
		ReadinessLeadDays:          row.ReadinessLeadDays,
		TotalCapacityUntilDeadline: row.RemainingCapacity,
		CompletedJobs:              rep.CompletedJobs,
		Reputation:                 rep.Reputation,
		StaleCalendar:              stale,
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

// reputationForPage computes the read-time reputation for every profile on one
// page in a single query (data-model.md section 19, FR-071), keyed by profile id
// text. It applies the FR-073 threshold here in the service: a completion
// percentage is only filled once the divisor reaches three agreed orders,
// otherwise enough_data stays false and completion_rate is nil, mirroring the
// public Reputation schema. A query error yields an empty map so the search
// still answers with empty reputation blocks rather than failing.
func (s *Service) reputationForPage(ctx context.Context, rows []sqlcgen.SearchCandidatesRow) map[string]pageReputation {
	out := make(map[string]pageReputation, len(rows))
	if len(rows) == 0 {
		return out
	}
	ids := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProfileID)
	}
	reps, err := s.queries().SearchReputation(ctx, ids)
	if err != nil {
		return out
	}
	for _, rep := range reps {
		body := reputationView{
			ReviewCount: int(rep.ReviewCount),
		}
		if rep.ReviewCount > 0 {
			body.AverageRating = floatFromNumeric(rep.AverageRating)
		}
		// FR-073: hold the completion percentage until at least three agreed
		// orders (the divisor). Below the threshold the client shows a "belum
		// cukup data" note, not a raw ratio, so completion_rate stays nil.
		if rep.CompletionDivisor >= 3 {
			body.EnoughData = true
			pct := int((rep.CompletionCompleted*100 + rep.CompletionDivisor/2) / rep.CompletionDivisor)
			body.CompletionRate = &pct
		}
		out[uuidString(rep.ProfileID)] = pageReputation{
			Reputation:    body,
			CompletedJobs: rep.CompletedJobs,
		}
	}
	return out
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
