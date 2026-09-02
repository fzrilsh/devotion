package masterdata

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/notification"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Service serves the four public read endpoints, the authenticated item
// proposal endpoint, and drives the two seed subcommands. It holds the pool and
// the Clock; the Clock supplies created_at for catalog and proposal rows
// (Rule 5), never time.Now. auth gates POST /master/proposals; notif enqueues
// the decision notification the proposer receives (FR-061). Both are nil in the
// seed subcommands, which touch none of that surface. It carries no per-request
// state.
type Service struct {
	pool  *pgxpool.Pool
	clock platform.Clock
	auth  httpx.Authenticator
	notif *notification.Service
}

// New builds a Service over pool. clock is injected so a test drives catalog and
// proposal timestamps by advancing time rather than reading the wall clock. auth
// and notif may be nil for the seed subcommands, which register no routes.
func New(pool *pgxpool.Pool, clock platform.Clock, auth httpx.Authenticator, notif *notification.Service) *Service {
	return &Service{pool: pool, clock: clock, auth: auth, notif: notif}
}

// queries returns a Queries bound to the pool for a standalone statement.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// Register wires the four read routes plus the item proposal endpoint. The read
// routes carry security:[] in the contract, so they register as Public: covered
// without a role check and out of the router's uncovered set. POST
// /master/proposals has no security:[], so it is gated to the two business roles
// (FR-061: a user proposes an item when none fits; both subcontractors filling a
// listing and buyers searching pick from the same catalog, FR-022). Admins
// manage the catalog through the T068 surface, not this proposal path.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	r.Public("GET /api/master/products", s.handleProducts)
	r.Public("GET /api/master/machines", s.handleMachines)
	r.Public("GET /api/regions/provinces", s.handleProvinces)
	r.Public("GET /api/regions/cities", s.handleCities)

	gate := httpx.RequireRole(auth, httpx.RoleSubcontractor, httpx.RoleBuyer)
	r.Gated("POST /api/master/proposals", gate, s.handleCreateProposal)

	s.registerAdmin(r, auth)
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

// maxProposalBody caps the proposal request body. The payload is two short
// strings, so this stops a client from streaming an unbounded body into the
// decoder.
const maxProposalBody = 8 << 10

// handleCreateProposal records a user's item proposal (FR-061). The route is
// gated to the two business roles, so a missing Principal is an invariant break
// (500). A createProposal error is a validationError (422) or an invariant 500.
func (s *Service) handleCreateProposal(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in proposalInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.createProposal(r.Context(), acc.ID, in)
	if err != nil {
		writeProposalErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// writeProposalErr maps a proposal service error to its problem response. A
// validationError renders per field; everything else, including the profile
// invariant break, is a 500.
func writeProposalErr(w http.ResponseWriter, err error) {
	var verr *validationError
	if errors.As(err, &verr) {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	httpx.WriteInternal(w)
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields and oversized
// bodies. It returns false and writes a validation problem on failure, so a
// handler can `if !decodeJSON(...) { return }`.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxProposalBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Format permintaan tidak sah.")
		return false
	}
	return true
}

// principalAccount pulls the authenticated UserAccount off the request context.
// The route is gated, so a missing Principal or a wrong Account type is an
// invariant violation and becomes a 500. The bool is false when it already
// wrote the 500, so the handler returns early.
func principalAccount(w http.ResponseWriter, r *http.Request) (sqlcgen.UserAccount, bool) {
	p, ok := httpx.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	acc, ok := p.Account.(sqlcgen.UserAccount)
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	return acc, true
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

// textPtr maps a pgtype.Text back to an optional string: an invalid (NULL) value
// becomes nil, so an unset admin note surfaces as a null reason in the response.
func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}
