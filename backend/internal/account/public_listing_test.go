package account

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/listing"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// silentNotifier satisfies listing.Notifier without recording anything. The
// public profile card is a read path, so no enqueue ever runs through it.
type silentNotifier struct{}

func (silentNotifier) Enqueue(_ context.Context, _ pgx.Tx, _ pgtype.UUID, _ sqlcgen.EventType, _, _ string, _ *string) error {
	return nil
}

// publicListingView mirrors the fields of the Listing shape the public profile
// card needs. It decodes the listing member of PublicProfile.
type publicListingView struct {
	ListingID         string `json:"listing_id"`
	ProfileID         string `json:"profile_id"`
	WeeklyCapacity    int32  `json:"weekly_capacity"`
	ReadinessLeadDays int32  `json:"readiness_lead_days"`
	Published         bool   `json:"published"`
	ProductItems      []struct {
		Name string `json:"name"`
	} `json:"product_items"`
	Machines []struct {
		MachineCount int32 `json:"machine_count"`
	} `json:"machines"`
}

// publicProfileWithListing decodes only what the capacity card assertions read.
type publicProfileWithListing struct {
	ProfileID string             `json:"profile_id"`
	Listing   *publicListingView `json:"listing"`
}

// wireListingViewer attaches a real listing service as the account service's
// viewer, the same wiring serve.go performs after both services exist.
func (h *harness) wireListingViewer() {
	h.svc.SetListingViewer(listing.New(h.pool, h.clock, silentNotifier{}))
}

// seedListing inserts a listing for profileID with one product and one machine,
// and returns its id. published controls visibility (FR-015).
func (h *harness) seedListing(t *testing.T, profileID pgtype.UUID, published bool) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)

	var productID, machineID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO catalog_item (type, name, active, created_at)
		 VALUES ('product', 'Kaos Oblong', true, $1) RETURNING id`,
		baseTime).Scan(&productID); err != nil {
		t.Fatalf("seed produk: %v", err)
	}
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO catalog_item (type, name, active, created_at)
		 VALUES ('machine', 'Mesin Obras', true, $1) RETURNING id`,
		baseTime).Scan(&machineID); err != nil {
		t.Fatalf("seed mesin: %v", err)
	}

	var listingID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published,
		     calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 500, 7, $2, $3, $4, $3, $3) RETURNING id`,
		profileID, published, baseTime, weekNow.AddDate(0, 0, 7*8)).Scan(&listingID); err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO listing_machine (listing_id, item_id, machine_count) VALUES ($1, $2, 4)`,
		listingID, machineID); err != nil {
		t.Fatalf("seed listing_machine: %v", err)
	}
	return listingID
}

// profileIDOf reads the business profile id of an account by its email.
func (h *harness) profileIDOf(t *testing.T, email string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT bp.id FROM business_profile bp JOIN user_account ua ON ua.id = bp.account_id
		 WHERE ua.email = $1`, email).Scan(&id); err != nil {
		t.Fatalf("ambil profil %q: %v", email, err)
	}
	return id
}

// getPublicProfile fetches a public profile anonymously and decodes it.
func (h *harness) getPublicProfile(t *testing.T, profileID pgtype.UUID) publicProfileWithListing {
	t.Helper()
	rec := h.do("GET", "/api/profile/"+uuidString(profileID), nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profile/{id}: status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var out publicProfileWithListing
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode profil publik: %v", err)
	}
	return out
}

// TestPublicProfile_ListingTerbit_KartuKapasitasTerisi_FR016 proves the public
// profile carries the profile's published listing, so the capacity card a
// searcher decides on (weekly capacity, lead time, products, machines) actually
// renders. Before this the field was hardcoded null and the card never appeared,
// leaving every search result linking to a profile that showed no capacity at
// all. FR-016.
func TestPublicProfile_ListingTerbit_KartuKapasitasTerisi_FR016(t *testing.T) {
	h := newHarness(t, "public_listing_terbit")
	h.wireListingViewer()
	h.registerAndLogin(t, "terbit@example.com", "+6281300013013", "rahasia123")

	profileID := h.profileIDOf(t, "terbit@example.com")
	listingID := h.seedListing(t, profileID, true)

	pub := h.getPublicProfile(t, profileID)
	if pub.Listing == nil {
		t.Fatal("listing null padahal profil punya listing terbit")
	}
	if pub.Listing.ListingID != uuidString(listingID) {
		t.Fatalf("listing_id = %q, mau %q", pub.Listing.ListingID, uuidString(listingID))
	}
	if pub.Listing.ProfileID != uuidString(profileID) {
		t.Fatalf("profile_id listing = %q, mau %q", pub.Listing.ProfileID, uuidString(profileID))
	}
	if pub.Listing.WeeklyCapacity != 500 {
		t.Fatalf("weekly_capacity = %d, mau 500", pub.Listing.WeeklyCapacity)
	}
	if pub.Listing.ReadinessLeadDays != 7 {
		t.Fatalf("readiness_lead_days = %d, mau 7", pub.Listing.ReadinessLeadDays)
	}
	if !pub.Listing.Published {
		t.Fatal("published mau true pada listing yang tampil publik")
	}
	if len(pub.Listing.ProductItems) != 1 || pub.Listing.ProductItems[0].Name != "Kaos Oblong" {
		t.Fatalf("product_items = %+v, mau satu produk Kaos Oblong", pub.Listing.ProductItems)
	}
	if len(pub.Listing.Machines) != 1 || pub.Listing.Machines[0].MachineCount != 4 {
		t.Fatalf("machines = %+v, mau satu mesin berjumlah 4", pub.Listing.Machines)
	}
}

// TestPublicProfile_ListingDisembunyikan_ListingNull_FR016 proves a listing whose
// owner turned visibility off (FR-015) is absent from the public profile, so
// hiding a listing hides its capacity everywhere, not only in search. FR-016.
func TestPublicProfile_ListingDisembunyikan_ListingNull_FR016(t *testing.T) {
	h := newHarness(t, "public_listing_sembunyi")
	h.wireListingViewer()
	h.registerAndLogin(t, "sembunyi@example.com", "+6281300014014", "rahasia123")

	profileID := h.profileIDOf(t, "sembunyi@example.com")
	h.seedListing(t, profileID, false)

	pub := h.getPublicProfile(t, profileID)
	if pub.Listing != nil {
		t.Fatalf("listing terisi padahal listing disembunyikan: %+v", pub.Listing)
	}
}

// TestPublicProfile_TanpaListing_ListingNull_FR016 proves a profile that never
// created a listing answers 200 with listing null rather than failing: a buyer
// account has a public profile too, and it simply has no capacity card. FR-016.
func TestPublicProfile_TanpaListing_ListingNull_FR016(t *testing.T) {
	h := newHarness(t, "public_listing_kosong")
	h.wireListingViewer()
	h.registerAndLogin(t, "kosong@example.com", "+6281300015015", "rahasia123")

	pub := h.getPublicProfile(t, h.profileIDOf(t, "kosong@example.com"))
	if pub.Listing != nil {
		t.Fatalf("listing terisi padahal profil belum punya listing: %+v", pub.Listing)
	}
}
