package admin

import (
	"encoding/json"
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// whatsAppStatus is the WhatsAppStatus response body (openapi.yaml). Only
// connected is required; qr_code and last_error are nullable and omitted when
// empty. The service number is never a field here (FR-082): the type has no
// place to carry it, so it cannot leak by accident.
type whatsAppStatus struct {
	Connected bool    `json:"connected"`
	QRCode    *string `json:"qr_code"`
	LastError *string `json:"last_error"`
}

// Register wires the WhatsApp link routes behind an admin-only gate. Both are
// covered by RequireRole so they stay out of the router's uncovered set, and only
// an admin principal ever reaches a handler (FR-082 keeps the number out of the
// body regardless).
func (m *Manager) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleAdmin)
	r.Gated("GET /api/admin/whatsapp", gate, m.handleStatus)
	r.Gated("POST /api/admin/whatsapp/reconnect", gate, m.handleReconnect)
}

// handleStatus reports the live link state to an admin. Reading the status arms a
// pairing cycle when the link is unpaired and none is live: one GetQRChannel call
// only yields a finite batch of codes, so without this the page would be blank
// for every admin who opens it more than a few minutes after startup. A paired
// link is left alone.
func (m *Manager) handleStatus(w http.ResponseWriter, _ *http.Request) {
	m.EnsureQR(false)
	m.writeStatus(w)
}

// handleReconnect discards whatever cycle is live and starts a new one, or
// bounces the socket when the link is paired. It is the "sambung ulang" button
// (T024b): the admin presses it because the code on screen is stale or the link
// is stuck, and neither case should require server access.
func (m *Manager) handleReconnect(w http.ResponseWriter, _ *http.Request) {
	m.EnsureQR(true)
	m.writeStatus(w)
}

// writeStatus renders the guarded status, mapping empty strings to null to match
// the nullable contract fields.
func (m *Manager) writeStatus(w http.ResponseWriter) {
	st := m.Status()
	body := whatsAppStatus{Connected: st.Connected}
	if st.QRCode != "" {
		body.QRCode = &st.QRCode
	}
	if st.LastError != "" {
		body.LastError = &st.LastError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
