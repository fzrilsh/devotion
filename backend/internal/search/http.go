package search

import (
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Register wires the single buyer-only search route. It is Gated behind
// RoleBuyer, so it stays out of the router's uncovered set and a wrong role is
// rejected before the handler runs. The Register(r, auth) shape mirrors the
// listing and admin services, so Service holds no Authenticator.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer)
	r.Gated("GET /api/search", gate, s.handleSearch)
}

// handleSearch validates the query parameters, runs the ranked page, and writes
// the SearchResult body. A bad parameter is a 422 with the offending field; a
// query failure is a 500.
func (s *Service) handleSearch(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	q, verr := parseSearchQuery(r)
	if verr != nil {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	res, err := s.search(r.Context(), acc.ID, q)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// parseSearchQuery validates the raw query string into a searchQuery. It follows
// the /search contract: product_item_id required uuid, machine_item_id optional
// uuid, quantity >= 1, deadline required date, max_lead_days >= 0 default unset,
// region_level enum with city_code/province_code required for their level,
// cursor opaque, size 1..50 default 20. Every failure names its field so the
// buyer sees which parameter to fix.
func parseSearchQuery(r *http.Request) (searchQuery, *validationError) {
	qv := r.URL.Query()
	var fields []httpx.FieldError
	var q searchQuery

	if raw := qv.Get("product_item_id"); raw == "" {
		fields = append(fields, httpx.FieldError{Field: "product_item_id", Message: "Pilih produk yang dicari."})
	} else if u, ok := parseUUID(raw); !ok {
		fields = append(fields, httpx.FieldError{Field: "product_item_id", Message: "Id produk tidak sah."})
	} else {
		q.productItem = &u
	}

	if raw := qv.Get("machine_item_id"); raw != "" {
		if u, ok := parseUUID(raw); !ok {
			fields = append(fields, httpx.FieldError{Field: "machine_item_id", Message: "Id mesin tidak sah."})
		} else {
			q.machineItem = &u
		}
	}

	if raw := qv.Get("quantity"); raw == "" {
		fields = append(fields, httpx.FieldError{Field: "quantity", Message: "Isi jumlah yang dibutuhkan."})
	} else if n, ok := atoiDefault(raw, 0); !ok || n < 1 {
		fields = append(fields, httpx.FieldError{Field: "quantity", Message: "Jumlah minimal 1 potong."})
	} else {
		q.quantity = int32(n)
	}

	if raw := qv.Get("deadline"); raw == "" {
		fields = append(fields, httpx.FieldError{Field: "deadline", Message: "Isi tenggat pesanan."})
	} else if t, err := platform.ParseDate(raw); err != nil {
		fields = append(fields, httpx.FieldError{Field: "deadline", Message: "Tenggat harus berformat YYYY-MM-DD."})
	} else {
		q.deadline = t
	}

	if raw := qv.Get("max_lead_days"); raw != "" {
		if n, ok := atoiDefault(raw, 0); !ok || n < 0 {
			fields = append(fields, httpx.FieldError{Field: "max_lead_days", Message: "Waktu ancang-ancang maksimal tidak boleh negatif."})
		} else {
			v := int32(n)
			q.maxLead = &v
		}
	}

	switch region := regionLevel(qv.Get("region_level")); region {
	case regionCity:
		q.region = regionCity
		if code := qv.Get("city_code"); !matchDigits(code, 4) {
			fields = append(fields, httpx.FieldError{Field: "city_code", Message: "Kode kota wajib empat digit untuk cakupan kota."})
		} else {
			q.cityCode = code
		}
	case regionProvince:
		q.region = regionProvince
		if code := qv.Get("province_code"); !matchDigits(code, 2) {
			fields = append(fields, httpx.FieldError{Field: "province_code", Message: "Kode provinsi wajib dua digit untuk cakupan provinsi."})
		} else {
			q.provinceCode = code
		}
	case regionNational:
		q.region = regionNational
	default:
		fields = append(fields, httpx.FieldError{Field: "region_level", Message: "Cakupan harus city, province, atau national."})
	}

	if raw := qv.Get("cursor"); raw != "" {
		c, ok := decodeCursor(raw)
		if !ok {
			fields = append(fields, httpx.FieldError{Field: "cursor", Message: "Kursor paginasi tidak sah."})
		} else {
			q.cursor = &c
		}
	}

	if n, ok := atoiDefault(qv.Get("size"), 20); !ok || n < 1 || n > 50 {
		fields = append(fields, httpx.FieldError{Field: "size", Message: "Ukuran halaman harus antara 1 dan 50."})
	} else {
		q.size = int32(n)
	}

	if len(fields) > 0 {
		return searchQuery{}, &validationError{fields: fields}
	}
	return q, nil
}

// matchDigits reports whether s is exactly n ASCII digits, the shape the region
// codes take (city four, province two). It avoids a regexp dependency for a
// check this small.
func matchDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
