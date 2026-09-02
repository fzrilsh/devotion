package masterdata

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// quietLogger discards router output during tests.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestNormalizeCityCode_StripsDot_FR058 pins the single silent-failure guard:
// the wilayah.id source dots city codes (32.73), and the city_code_format and
// city_belongs_to_province constraints reject anything but the dot-free form.
func TestNormalizeCityCode_StripsDot_FR058(t *testing.T) {
	cases := map[string]string{
		"32.73":  "3273",
		"3273":   "3273",
		" 32.73": "3273",
		"11.01":  "1101",
	}
	for in, want := range cases {
		if got := NormalizeCityCode(in); got != want {
			t.Errorf("NormalizeCityCode(%q) = %q, mau %q", in, want, got)
		}
	}
}

// sampleData is a small hand-built snapshot: one province (Jawa Barat, 32) with
// two cities, plus another province, exercising the seed and read paths without
// touching the network.
var sampleData = RegionData{
	Provinces: []region{
		{Code: "32", Name: "Jawa Barat"},
		{Code: "11", Name: "Aceh"},
	},
	Cities: []region{
		{Code: "3273", Name: "Kota Bandung"},
		{Code: "3204", Name: "Kabupaten Bandung"},
		{Code: "1101", Name: "Kabupaten Aceh Selatan"},
	},
}

func newService(t *testing.T, name string) (*Service, *httpx.Router) {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	svc := New(pool, clock, nil, nil)
	r := httpx.NewRouter(quietLogger())
	svc.Register(r, &mockAuth{})
	return svc, r
}

// TestSeedRegions_Idempotent_FR058 seeds twice and asserts the province/city
// counts are unchanged and no city row has a province mismatch (the normalize
// guard). A second run must not duplicate or delete.
func TestSeedRegions_Idempotent_FR058(t *testing.T) {
	svc, _ := newService(t, "regions_seed")
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := svc.SeedRegions(ctx, sampleData); err != nil {
			t.Fatalf("SeedRegions run %d: %v", i, err)
		}
	}

	q := svc.queries()
	prov, err := q.CountProvinces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if prov != 2 {
		t.Errorf("provinsi = %d, mau 2", prov)
	}
	cities, err := q.CountCities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cities != 3 {
		t.Errorf("kota = %d, mau 3", cities)
	}

	var mismatch int64
	if err := svc.pool.QueryRow(ctx,
		"SELECT count(*)::bigint FROM city WHERE left(code,2) <> province_code").Scan(&mismatch); err != nil {
		t.Fatal(err)
	}
	if mismatch != 0 {
		t.Errorf("baris kota dengan province_code salah = %d, mau 0", mismatch)
	}
}

// TestSeedMasterData_Idempotent_FR058 seeds the catalog twice and asserts the
// per-type counts equal the baseline lengths.
func TestSeedMasterData_Idempotent_FR058(t *testing.T) {
	svc, _ := newService(t, "catalog_seed")
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := svc.SeedMasterData(ctx); err != nil {
			t.Fatalf("SeedMasterData run %d: %v", i, err)
		}
	}

	q := svc.queries()
	products, err := q.CountCatalogByType(ctx, "product")
	if err != nil {
		t.Fatal(err)
	}
	if int(products) != len(baselineProducts) {
		t.Errorf("produk = %d, mau %d", products, len(baselineProducts))
	}
	machines, err := q.CountCatalogByType(ctx, "machine")
	if err != nil {
		t.Fatal(err)
	}
	if int(machines) != len(baselineMachines) {
		t.Errorf("mesin = %d, mau %d", machines, len(baselineMachines))
	}
}
