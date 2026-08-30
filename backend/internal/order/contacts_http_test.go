package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// TestWorkOrderContacts_PartySeesCounterparty_FR092 proves each party to a formed
// order reads the other side's off-platform contact block (business name, email,
// WhatsApp), the prerequisite for the payment and coordination that happen
// directly between the parties (FR-040). The buyer sees the subcontractor's
// contacts tagged role=subcontractor, and the subcontractor sees the buyer's
// tagged role=buyer, so neither is shown its own side echoed back.
func TestWorkOrderContacts_PartySeesCounterparty_FR092(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "contacts_party")

	// Buyer reads: sees the subcontractor.
	buyerHandler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID)+"/contacts", nil)
	rec := httptest.NewRecorder()
	buyerHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Counterparty struct {
			Role         string `json:"role"`
			BusinessName string `json:"business_name"`
			Email        string `json:"email"`
			WhatsApp     string `json:"whatsapp"`
		} `json:"counterparty"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderContacts %q: %v", rec.Body.String(), err)
	}
	if body.Counterparty.Role != "subcontractor" {
		t.Fatalf("role %q, mau subcontractor; pembeli harus melihat kontak subkontraktor (FR-092)", body.Counterparty.Role)
	}
	if body.Counterparty.BusinessName == "" || body.Counterparty.Email == "" || body.Counterparty.WhatsApp == "" {
		t.Fatalf("kontak subkontraktor tidak lengkap: %+v (FR-092)", body.Counterparty)
	}

	// Subcontractor reads: sees the buyer.
	subHandler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)
	req = httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID)+"/contacts", nil)
	rec = httptest.NewRecorder()
	subHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderContacts %q: %v", rec.Body.String(), err)
	}
	if body.Counterparty.Role != "buyer" {
		t.Fatalf("role %q, mau buyer; subkontraktor harus melihat kontak pembeli (FR-092)", body.Counterparty.Role)
	}
	if body.Counterparty.BusinessName == "" || body.Counterparty.Email == "" || body.Counterparty.WhatsApp == "" {
		t.Fatalf("kontak pembeli tidak lengkap: %+v (FR-092)", body.Counterparty)
	}
}

// TestWorkOrderContacts_NonPartyGets404_FR092 proves a caller on neither side of
// the order gets the same 404 as a missing order: the contacts carry phone
// numbers and email, so a non-party must never learn the order exists, let alone
// read its parties' contacts.
func TestWorkOrderContacts_NonPartyGets404_FR092(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "contacts_nonparty")
	stranger := seedAcceptAccount(t, h.pool, "stranger_contacts@contoh.test", false)
	handler := woRouter(h, httpx.RoleBuyer, stranger)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID)+"/contacts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; pihak ketiga tidak boleh mengakses kontak (FR-092); body %s", rec.Code, rec.Body.String())
	}
}

// TestWorkOrderContacts_AdminGets404_FR092 proves the contacts endpoint does NOT
// admit an admin, unlike the detail read. The admin holds no business role and is
// never a transacting party; it reads the order through FR-045/FR-046, not
// through the parties' private off-platform contacts. So an admin is treated as a
// non-party here and sees a 404.
func TestWorkOrderContacts_AdminGets404_FR092(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "contacts_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_contacts@contoh.test", false)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID)+"/contacts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; admin bukan pihak dan tidak membaca kontak pribadi (FR-092); body %s", rec.Code, rec.Body.String())
	}
}
