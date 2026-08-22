package masterdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// getJSONBody drives one GET through the router and decodes the JSON array body.
func getJSONBody[T any](t *testing.T, h http.Handler, path string) []T {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d, mau 200", path, rec.Code)
	}
	var out []T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("GET %s: urai body: %v", path, err)
	}
	return out
}

// TestReadEndpoints_ReturnSeededData_FR058 seeds a snapshot and the catalog,
// then reads all four public endpoints, checking the contract field names.
func TestReadEndpoints_ReturnSeededData_FR058(t *testing.T) {
	svc, r := newService(t, "read_endpoints")
	ctx := context.Background()
	if err := svc.SeedRegions(ctx, sampleData); err != nil {
		t.Fatal(err)
	}
	if err := svc.SeedMasterData(ctx); err != nil {
		t.Fatal(err)
	}
	h := r.Handler()

	provinces := getJSONBody[provinceItem](t, h, "/api/regions/provinces")
	if len(provinces) != 2 {
		t.Errorf("provinsi = %d, mau 2", len(provinces))
	}

	allCities := getJSONBody[cityItem](t, h, "/api/regions/cities")
	if len(allCities) != 3 {
		t.Errorf("kota = %d, mau 3", len(allCities))
	}

	jabar := getJSONBody[cityItem](t, h, "/api/regions/cities?province=32")
	if len(jabar) != 2 {
		t.Errorf("kota Jabar = %d, mau 2", len(jabar))
	}
	for _, c := range jabar {
		if c.ProvinceCode != "32" {
			t.Errorf("kota %s province_code = %q, mau 32", c.Code, c.ProvinceCode)
		}
	}

	products := getJSONBody[catalogItem](t, h, "/api/master/products")
	if len(products) != len(baselineProducts) {
		t.Errorf("produk = %d, mau %d", len(products), len(baselineProducts))
	}
	for _, p := range products {
		if p.Kind != "product" {
			t.Errorf("produk %s kind = %q, mau product", p.Name, p.Kind)
		}
		if p.ItemID == "" {
			t.Errorf("produk %s item_id kosong", p.Name)
		}
	}

	machines := getJSONBody[catalogItem](t, h, "/api/master/machines")
	if len(machines) != len(baselineMachines) {
		t.Errorf("mesin = %d, mau %d", len(machines), len(baselineMachines))
	}
}

// TestCities_UnknownProvince_EmptyArray_FR058 confirms an out-of-range province
// filter yields an empty array, not an error.
func TestCities_UnknownProvince_EmptyArray_FR058(t *testing.T) {
	svc, r := newService(t, "cities_unknown")
	if err := svc.SeedRegions(context.Background(), sampleData); err != nil {
		t.Fatal(err)
	}
	cities := getJSONBody[cityItem](t, r.Handler(), "/api/regions/cities?province=99")
	if len(cities) != 0 {
		t.Errorf("kota provinsi tak dikenal = %d, mau 0", len(cities))
	}
}
