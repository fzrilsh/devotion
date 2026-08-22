package masterdata

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Service serves the four public read endpoints and drives the two seed
// subcommands. It holds the pool and the Clock; the Clock supplies created_at
// for catalog rows (Rule 5), never time.Now. It carries no per-request state.
type Service struct {
	pool  *pgxpool.Pool
	clock platform.Clock
}

// New builds a Service over pool. clock is injected so a test drives catalog
// timestamps by advancing time rather than reading the wall clock.
func New(pool *pgxpool.Pool, clock platform.Clock) *Service {
	return &Service{pool: pool, clock: clock}
}

// queries returns a Queries bound to the pool for a standalone statement.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// Register wires the four read routes. All carry security:[] in the contract,
// so they register as Public: they are covered without a role check and stay
// out of the router's uncovered set. Region and catalog lists are reference
// data a signed-out visitor may read.
func (s *Service) Register(r *httpx.Router) {
	r.Public("GET /api/master/products", s.handleProducts)
	r.Public("GET /api/master/machines", s.handleMachines)
	r.Public("GET /api/regions/provinces", s.handleProvinces)
	r.Public("GET /api/regions/cities", s.handleCities)
}

// catalogItem is the CatalogItem response body. The field names follow the
// contract (item_id, kind), which differ from the DB columns (id, type), so the
// handler maps rather than serializing the row directly.
type catalogItem struct {
	ItemID string `json:"item_id"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// provinceItem is the Province response body: code and name.
type provinceItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// cityItem is the City response body: code, province_code, name.
type cityItem struct {
	Code         string `json:"code"`
	ProvinceCode string `json:"province_code"`
	Name         string `json:"name"`
}

func (s *Service) handleProducts(w http.ResponseWriter, r *http.Request) {
	s.writeCatalog(w, r, sqlcgen.ItemTypeProduct)
}

func (s *Service) handleMachines(w http.ResponseWriter, r *http.Request) {
	s.writeCatalog(w, r, sqlcgen.ItemTypeMachine)
}

// writeCatalog lists the active items of one type and writes them as the
// CatalogItem array. On a query error it returns a 500 problem rather than a
// partial list.
func (s *Service) writeCatalog(w http.ResponseWriter, r *http.Request, kind sqlcgen.ItemType) {
	rows, err := s.queries().ListActiveCatalogItems(r.Context(), kind)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	out := make([]catalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, catalogItem{
			ItemID: uuidString(row.ID),
			Kind:   string(row.Type),
			Name:   row.Name,
			Active: row.Active,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handleProvinces(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries().ListProvinces(r.Context())
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	out := make([]provinceItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, provinceItem{Code: row.Code, Name: row.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCities lists all cities, or the cities of one province when the
// province query parameter is present. An out-of-range or unknown province
// simply yields an empty array, matching a filter that no row satisfies.
func (s *Service) handleCities(w http.ResponseWriter, r *http.Request) {
	province := r.URL.Query().Get("province")
	var (
		rows []sqlcgen.City
		err  error
	)
	if province != "" {
		rows, err = s.queries().ListCitiesByProvince(r.Context(), province)
	} else {
		rows, err = s.queries().ListCities(r.Context())
	}
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	out := make([]cityItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, cityItem{Code: row.Code, ProvinceCode: row.ProvinceCode, Name: row.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// writeJSON encodes v as the 2xx body. Error bodies go through
// httpx.WriteProblem; this is only for the success payloads the handlers own.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b, _ := u.MarshalJSON()
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}

// tstz wraps a time as a pgtype.Timestamptz for the catalog created_at column.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
