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

// Register wires GET /api/admin/whatsapp behind an admin-only gate. The route is
// covered by RequireRole so it stays out of the router's uncovered set, and only
// an admin principal ever reaches the handler (FR-082 keeps the number out of the
// body regardless).
func (m *Manager) Register(r *httpx.Router, auth httpx.Authenticator) {
	r.Gated("GET /api/admin/whatsapp", httpx.RequireRole(auth, httpx.RoleAdmin), m.handleStatus)
}

// handleStatus reports the live link state to an admin. It reads the guarded
// status and maps empty strings to null, matching the nullable contract fields.
func (m *Manager) handleStatus(w http.ResponseWriter, _ *http.Request) {
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
