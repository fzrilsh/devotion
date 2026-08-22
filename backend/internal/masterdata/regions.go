package masterdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// wilayahBaseURL is the wilayah.id API root. Only two levels are fetched:
// provinces and their regencies/cities. The response is wrapped in a data
// object with English code and name fields (docs/master-data/README.md).
const wilayahBaseURL = "https://wilayah.id/api"

// regionsFileName is the normalized on-disk copy seed:regions reads by default.
// --refresh rewrites it from the network. Prinsip V forbids depending on the
// external source while serving a request, so the seed reads this copy, and it
// also keeps a demo alive when wilayah.id is down.
const regionsFileName = "regions.json"

// fetchTimeout bounds a single wilayah.id call so a hung network does not stall
// the whole refresh. The seed is an operator command, not a request path, so a
// generous but finite timeout is right.
const fetchTimeout = 30 * time.Second

// region is one code/name pair from the wilayah.id data array, shared by the
// province and city responses since both carry exactly these two fields.
type region struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// wilayahResponse is the data-wrapped envelope every wilayah.id endpoint
// returns. meta is ignored; only the data array is used.
type wilayahResponse struct {
	Data []region `json:"data"`
}

// RegionData is the normalized snapshot written to and read from regions.json.
// City codes are already dot-free here, so loading the file never has to
// normalize again and a hand-edited file cannot reintroduce a dotted code
// without the seed's constraints catching it.
type RegionData struct {
	Provinces []region `json:"provinces"`
	Cities    []region `json:"cities"`
}

// FetchRegions pulls the province list and every province's cities from
// wilayah.id, normalizing city codes as it goes. It returns the snapshot ready
// to write to disk or seed. A single failed call aborts the whole fetch: a
// partial region set would silently drop cities, and the operator must see the
// error rather than seed an incomplete list.
func FetchRegions(ctx context.Context) (RegionData, error) {
	client := &http.Client{Timeout: fetchTimeout}

	var provinces []region
	if err := getJSON(ctx, client, wilayahBaseURL+"/provinces.json", &provinces); err != nil {
		return RegionData{}, fmt.Errorf("ambil provinsi: %w", err)
	}
	sort.Slice(provinces, func(i, j int) bool { return provinces[i].Code < provinces[j].Code })

	var cities []region
	for _, p := range provinces {
		var cs []region
		url := fmt.Sprintf("%s/regencies/%s.json", wilayahBaseURL, p.Code)
		if err := getJSON(ctx, client, url, &cs); err != nil {
			return RegionData{}, fmt.Errorf("ambil kota provinsi %s: %w", p.Code, err)
		}
		for _, c := range cs {
			cities = append(cities, region{Code: NormalizeCityCode(c.Code), Name: c.Name})
		}
	}
	sort.Slice(cities, func(i, j int) bool { return cities[i].Code < cities[j].Code })

	return RegionData{Provinces: provinces, Cities: cities}, nil
}

// getJSON fetches url and decodes the data array into dst. It reads the wilayah
// data envelope, so dst receives just the region slice.
func getJSON(ctx context.Context, client *http.Client, url string, dst *[]region) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d dari %s", resp.StatusCode, url)
	}
	var env wilayahResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	*dst = env.Data
	return nil
}

// LoadRegions reads the normalized snapshot from dir/regions.json. It does not
// touch the network. A missing file is an error naming the path so the operator
// knows to run --refresh once.
func LoadRegions(dir string) (RegionData, error) {
	path := filepath.Join(dir, regionsFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		return RegionData{}, fmt.Errorf("baca %s (jalankan seed:regions --refresh sekali): %w", path, err)
	}
	var data RegionData
	if err := json.Unmarshal(b, &data); err != nil {
		return RegionData{}, fmt.Errorf("urai %s: %w", path, err)
	}
	return data, nil
}

// WriteRegions writes the normalized snapshot to dir/regions.json with a
// trailing newline, indented so the committed file diffs cleanly.
func WriteRegions(dir string, data RegionData) error {
	path := filepath.Join(dir, regionsFileName)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("tulis %s: %w", path, err)
	}
	return nil
}

// SeedRegions upserts every province and city into the database. It is
// idempotent on code: insert if absent, update the name if present, never
// delete, because business_profile references these rows and a delete would
// orphan real data. Cities are expected already normalized (dot-free); this
// normalizes again defensively so a hand-edited regions.json cannot slip a
// dotted code past the constraints.
func (s *Service) SeedRegions(ctx context.Context, data RegionData) error {
	q := s.queries()
	for _, p := range data.Provinces {
		if err := q.UpsertProvince(ctx, sqlcgen.UpsertProvinceParams{Code: p.Code, Name: p.Name}); err != nil {
			return fmt.Errorf("upsert provinsi %s: %w", p.Code, err)
		}
	}
	for _, c := range data.Cities {
		code := NormalizeCityCode(c.Code)
		if err := q.UpsertCity(ctx, sqlcgen.UpsertCityParams{
			Code:         code,
			ProvinceCode: code[:2],
			Name:         c.Name,
		}); err != nil {
			return fmt.Errorf("upsert kota %s: %w", code, err)
		}
	}
	return nil
}
