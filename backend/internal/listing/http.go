package listing

import (
	"errors"
	"net/http"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// dateLayout is the wire format for week_start on the periods endpoints, a bare
// calendar date. The time of day is meaningless for a Monday week boundary.
const dateLayout = "2006-01-02"

// catalogItem is the CatalogItem response body. Field names follow the contract
// (item_id, kind), which differ from the DB columns (id, type). It is copied
// locally rather than imported from masterdata to keep the packages decoupled.
type catalogItem struct {
	ItemID string `json:"item_id"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// machineView is a listing's machine link: the catalog item plus its unit count.
type machineView struct {
	Item         catalogItem `json:"item"`
	MachineCount int32       `json:"machine_count"`
}

// listingView is the Listing response body, mapping contract names by hand.
// calendar_updated_at is nullable (null until the owner first edits the
// calendar), so it is a pointer.
type listingView struct {
	ListingID         string         `json:"listing_id"`
	ProfileID         string         `json:"profile_id"`
	WeeklyCapacity    int32          `json:"weekly_capacity"`
	ReadinessLeadDays int32          `json:"readiness_lead_days"`
	Published         bool           `json:"published"`
	CalendarUpdatedAt *time.Time     `json:"calendar_updated_at"`
	HorizonUntil      string         `json:"horizon_until"`
	ProductItems      []catalogItem  `json:"product_items"`
	Machines          []machineView  `json:"machines"`
}

// periodView is the AvailabilityPeriod response body. remaining is derived in
// Go: max(0, capacity - allocated), forced to zero when the week is marked full.
type periodView struct {
	WeekStart  string `json:"week_start"`
	Capacity   int32  `json:"capacity"`
	Allocated  int64  `json:"allocated"`
	Remaining  int64  `json:"remaining"`
	MarkedFull bool   `json:"marked_full"`
}

// machineInput is one element of the ListingRequest.machines array.
type machineInput struct {
	ItemID       string `json:"item_id"`
	MachineCount int32  `json:"machine_count"`
}

// listingInput is the ListingRequest body for POST and PUT /listing/me.
type listingInput struct {
	WeeklyCapacity    int32          `json:"weekly_capacity"`
	ReadinessLeadDays int32          `json:"readiness_lead_days"`
	ProductItemIDs    []string       `json:"product_item_ids"`
	Machines          []machineInput `json:"machines"`
}

// visibilityInput is the body of PUT /listing/me/visibility.
type visibilityInput struct {
	Published bool `json:"published"`
}

// periodInput is one PeriodUpdateItem: a week and its new capacity/full flag.
type periodInput struct {
	WeekStart  string `json:"week_start"`
	Capacity   int32  `json:"capacity"`
	MarkedFull bool   `json:"marked_full"`
}

// periodsInput is the PUT /listing/me/periods body, wrapping the batch under a
// periods key as the contract requires.
type periodsInput struct {
	Periods []periodInput `json:"periods"`
}

// newListingView assembles the response from the row and its item links.
func newListingView(l sqlcgen.CapacityListing, products []sqlcgen.ListListingProductsRow, machines []sqlcgen.ListListingMachinesRow) listingView {
	v := listingView{
		ListingID:         uuidString(l.ID),
		ProfileID:         uuidString(l.ProfileID),
		WeeklyCapacity:    l.WeeklyCapacity,
		ReadinessLeadDays: l.ReadinessLeadDays,
		Published:         l.Published,
		HorizonUntil:      l.HorizonUntil.Time.Format(dateLayout),
		ProductItems:      make([]catalogItem, 0, len(products)),
		Machines:          make([]machineView, 0, len(machines)),
	}
	if l.CalendarUpdatedAt.Valid {
		t := l.CalendarUpdatedAt.Time
		v.CalendarUpdatedAt = &t
	}
	for _, p := range products {
		v.ProductItems = append(v.ProductItems, catalogItem{
			ItemID: uuidString(p.ID),
			Kind:   string(p.Type),
			Name:   p.Name,
			Active: p.Active,
		})
	}
	for _, m := range machines {
		v.Machines = append(v.Machines, machineView{
			Item: catalogItem{
				ItemID: uuidString(m.ID),
				Kind:   string(m.Type),
				Name:   m.Name,
				Active: m.Active,
			},
			MachineCount: m.MachineCount,
		})
	}
	return v
}

// periodViews maps the range rows to the response, deriving remaining.
func periodViews(rows []sqlcgen.ListPeriodsInRangeRow) []periodView {
	out := make([]periodView, 0, len(rows))
	for _, r := range rows {
		allocated := allocatedInt64(r.Allocated)
		remaining := max(int64(r.TotalCapacity)-allocated, 0)
		if r.MarkedFull {
			remaining = 0
		}
		out = append(out, periodView{
			WeekStart:  r.WeekStart.Time.Format(dateLayout),
			Capacity:   r.TotalCapacity,
			Allocated:  allocated,
			Remaining:  remaining,
			MarkedFull: r.MarkedFull,
		})
	}
	return out
}

// Register wires the six subcontractor-only listing routes. Every route is
// Gated behind RoleSubcontractor, so each stays out of the router's uncovered
// set and a wrong role is rejected before the handler runs. The Register(r,
// auth) shape mirrors admin.Register, so Service holds no Authenticator.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleSubcontractor)
	r.Gated("GET /api/listing/me", gate, s.handleGetListing)
	r.Gated("POST /api/listing/me", gate, s.handleCreateListing)
	r.Gated("PUT /api/listing/me", gate, s.handleUpdateListing)
	r.Gated("PUT /api/listing/me/visibility", gate, s.handleSetVisibility)
	r.Gated("GET /api/listing/me/periods", gate, s.handleGetPeriods)
	r.Gated("PUT /api/listing/me/periods", gate, s.handleUpdatePeriods)
}

func (s *Service) handleGetListing(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	view, err := s.getListing(r.Context(), acc.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Service) handleCreateListing(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in listingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.createListing(r.Context(), acc.ID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Service) handleUpdateListing(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in listingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.updateListing(r.Context(), acc.ID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Service) handleSetVisibility(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in visibilityInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.setVisibility(r.Context(), acc.ID, in.Published)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Service) handleGetPeriods(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	from, ok := parseDateQuery(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseDateQuery(w, r, "to")
	if !ok {
		return
	}
	views, err := s.listPeriods(r.Context(), acc.ID, from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Service) handleUpdatePeriods(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in periodsInput
	if !decodeJSON(w, r, &in) {
		return
	}
	views, err := s.updatePeriods(r.Context(), acc.ID, in.Periods)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

// parseDateQuery reads an optional YYYY-MM-DD query parameter. Absent is a nil
// pointer (use the default bound); present but malformed is a 422.
func parseDateQuery(w http.ResponseWriter, r *http.Request, key string) (*time.Time, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, true
	}
	t, err := platform.ParseDate(raw)
	if err != nil {
		httpx.WriteValidation(w, "Parameter tanggal tidak sah.", []httpx.FieldError{
			{Field: key, Message: "Tanggal harus berformat YYYY-MM-DD."},
		})
		return nil, false
	}
	return &t, true
}

// validateListingInput enforces the ListingRequest field rules (all 422 with a
// field name): capacity >= 1, lead days 0..365, at least one product and one
// machine, every id a valid UUID, machine_count >= 1, no duplicate ids across
// either array.
func (s *Service) validateListingInput(in listingInput) *validationError {
	var fields []httpx.FieldError

	if in.WeeklyCapacity < 1 {
		fields = append(fields, httpx.FieldError{Field: "weekly_capacity", Message: "Kapasitas mingguan minimal 1 potong."})
	}
	if in.ReadinessLeadDays < 0 || in.ReadinessLeadDays > 365 {
		fields = append(fields, httpx.FieldError{Field: "readiness_lead_days", Message: "Waktu ancang-ancang harus antara 0 dan 365 hari."})
	}

	if len(in.ProductItemIDs) == 0 {
		fields = append(fields, httpx.FieldError{Field: "product_item_ids", Message: "Pilih minimal satu produk."})
	} else {
		seen := make(map[string]struct{}, len(in.ProductItemIDs))
		for _, id := range in.ProductItemIDs {
			if _, ok := parseUUID(id); !ok {
				fields = append(fields, httpx.FieldError{Field: "product_item_ids", Message: "Ada id produk yang tidak sah."})
				break
			}
			if _, dup := seen[id]; dup {
				fields = append(fields, httpx.FieldError{Field: "product_item_ids", Message: "Ada produk yang terpilih lebih dari sekali."})
				break
			}
			seen[id] = struct{}{}
		}
	}

	if len(in.Machines) == 0 {
		fields = append(fields, httpx.FieldError{Field: "machines", Message: "Pilih minimal satu mesin."})
	} else {
		seen := make(map[string]struct{}, len(in.Machines))
		for _, m := range in.Machines {
			if _, ok := parseUUID(m.ItemID); !ok {
				fields = append(fields, httpx.FieldError{Field: "machines", Message: "Ada id mesin yang tidak sah."})
				break
			}
			if m.MachineCount < 1 {
				fields = append(fields, httpx.FieldError{Field: "machines", Message: "Jumlah mesin minimal 1 unit."})
				break
			}
			if _, dup := seen[m.ItemID]; dup {
				fields = append(fields, httpx.FieldError{Field: "machines", Message: "Ada mesin yang terpilih lebih dari sekali."})
				break
			}
			seen[m.ItemID] = struct{}{}
		}
	}

	if len(fields) > 0 {
		return &validationError{fields: fields}
	}
	return nil
}

// parsedPeriod is a validated period edit with its week parsed to a time.
type parsedPeriod struct {
	week       time.Time
	capacity   int32
	markedFull bool
}

// validatePeriodInput checks the whole batch before any write: 1..MaxPeriodBatch
// elements, each week_start a Monday parsed as YYYY-MM-DD, no earlier than the
// current week and no more than 26 weeks ahead, capacity >= 0, no duplicate
// weeks. It returns the parsed elements so the caller writes them without
// re-parsing.
func (s *Service) validatePeriodInput(items []periodInput, weekNow time.Time) ([]parsedPeriod, *validationError) {
	var fields []httpx.FieldError

	if len(items) == 0 {
		fields = append(fields, httpx.FieldError{Field: "periods", Message: "Sertakan minimal satu periode."})
		return nil, &validationError{fields: fields}
	}
	if len(items) > MaxPeriodBatch {
		fields = append(fields, httpx.FieldError{Field: "periods", Message: "Maksimal 26 periode per permintaan."})
		return nil, &validationError{fields: fields}
	}

	maxWeek := weekNow.AddDate(0, 0, 7*MaxPeriodBatch)
	seen := make(map[string]struct{}, len(items))
	parsed := make([]parsedPeriod, 0, len(items))

	for _, it := range items {
		t, err := platform.ParseDate(it.WeekStart)
		if err != nil {
			fields = append(fields, httpx.FieldError{Field: "week_start", Message: "Tanggal harus berformat YYYY-MM-DD."})
			continue
		}
		if platform.WeekStart(t) != t || t.Weekday() != time.Monday {
			fields = append(fields, httpx.FieldError{Field: "week_start", Message: "Awal minggu harus hari Senin."})
			continue
		}
		if t.Before(weekNow) {
			fields = append(fields, httpx.FieldError{Field: "week_start", Message: "Minggu tidak boleh di masa lampau."})
			continue
		}
		if t.After(maxWeek) {
			fields = append(fields, httpx.FieldError{Field: "week_start", Message: "Minggu tidak boleh lebih dari 26 minggu ke depan."})
			continue
		}
		if it.Capacity < 0 {
			fields = append(fields, httpx.FieldError{Field: "capacity", Message: "Kapasitas tidak boleh negatif."})
			continue
		}
		if _, dup := seen[it.WeekStart]; dup {
			fields = append(fields, httpx.FieldError{Field: "week_start", Message: "Ada minggu yang muncul lebih dari sekali."})
			continue
		}
		seen[it.WeekStart] = struct{}{}
		parsed = append(parsed, parsedPeriod{week: t, capacity: it.Capacity, markedFull: it.MarkedFull})
	}

	if len(fields) > 0 {
		return nil, &validationError{fields: fields}
	}
	return parsed, nil
}

// writeErr maps a service error to its problem response. Sentinels get their
// contract code and Indonesian detail; a conflictError carries its own; a
// validationError renders per field; anything else is a 500.
func writeErr(w http.ResponseWriter, err error) {
	var verr *validationError
	if errors.As(err, &verr) {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	var cerr *conflictError
	if errors.As(err, &cerr) {
		httpx.WriteProblem(w, cerr.code, cerr.detail)
		return
	}
	switch {
	case errors.Is(err, errListingNotFound):
		httpx.WriteProblem(w, httpx.CodeListingNotFound, "Belum ada listing kapasitas. Buat listing lebih dulu.")
	case errors.Is(err, errListingExists):
		httpx.WriteProblem(w, httpx.CodeListingAlreadyExists, "Profil ini sudah punya listing. Ubah listing yang ada, bukan membuat baru.")
	case errors.Is(err, errPeriodOutsideCal):
		httpx.WriteInternal(w)
	default:
		httpx.WriteInternal(w)
	}
}

