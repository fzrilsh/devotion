// Package health serves GET /health, the liveness probe the container
// healthcheck and any external uptime monitor read. It checks the three
// dependencies a running serve process cannot work without: the database, the
// WhatsApp link, and free space on the upload volume. Any failing check makes
// the response 503 so a degraded instance is pulled out of rotation, and the
// body reports each dependency's state so an operator sees which one broke
// without shelling into the box.
//
// The response carries no secrets by construction: dependency states are a
// fixed enum ("ok" / "down"), never an error string, connection URL, or the
// WhatsApp service number (FR-082).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"syscall"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Pinger is the database dependency: a pool that can round-trip a ping. It is an
// interface so health does not import the pool package and the check is trivial
// to fake in a test.
type Pinger interface {
	Ping(ctx context.Context) error
}

// WhatsAppLink is the WhatsApp dependency, satisfied by *admin.Manager. Only the
// connected bit is read; the service number never crosses this boundary
// (FR-082).
type WhatsAppLink interface {
	Connected() bool
}

// Checker holds the dependencies GET /health probes. minFreeBytes is the floor
// below which the upload volume is reported down: past it a new upload would
// fail, so the instance is unhealthy even though it still answers.
type Checker struct {
	db          Pinger
	wa          WhatsAppLink
	clock       platform.Clock
	uploadPath  string
	minFreeByte uint64
}

// New builds a Checker. uploadPath is the directory uploads land in; minFreeMB
// is the free-space floor in megabytes.
func New(db Pinger, wa WhatsAppLink, clock platform.Clock, uploadPath string, minFreeMB int) *Checker {
	return &Checker{
		db:          db,
		wa:          wa,
		clock:       clock,
		uploadPath:  uploadPath,
		minFreeByte: uint64(minFreeMB) * 1024 * 1024,
	}
}

// state is a dependency's reported condition. The two values are the whole
// vocabulary: nothing here can carry an error message or a secret.
type state string

const (
	stateOK   state = "ok"
	stateDown state = "down"
)

// response is the /health body. status is "ok" only when every dependency is,
// matching the Health contract's status enum while adding the per-dependency
// detail T025 requires. checks holds one state per dependency.
type response struct {
	Status string           `json:"status"`
	Time   time.Time        `json:"time"`
	Checks map[string]state `json:"checks"`
}

// Register wires GET /health as a public route (security:[] in the contract).
// It sits outside /api/, so it is exempt from the uncovered-route check, but
// registering it through Public states the no-auth decision explicitly.
func (c *Checker) Register(r *httpx.Router) {
	r.Public("GET /health", c.handle)
}

// handle probes every dependency and reports 200 when all pass, 503 when any
// fails. Every check always runs so the body reflects the full state, not just
// the first failure.
func (c *Checker) handle(w http.ResponseWriter, r *http.Request) {
	checks := map[string]state{
		"database": c.checkDB(r.Context()),
		"whatsapp": c.checkWhatsApp(),
		"storage":  c.checkStorage(),
	}

	healthy := true
	for _, s := range checks {
		if s != stateOK {
			healthy = false
		}
	}

	body := response{Time: c.clock.Now(), Checks: checks}
	code := http.StatusOK
	if healthy {
		body.Status = "ok"
	} else {
		body.Status = "down"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// checkDB reports whether the pool can reach Postgres within a short deadline,
// so a hung database cannot hang the health probe itself.
func (c *Checker) checkDB(ctx context.Context) state {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := c.db.Ping(ctx); err != nil {
		return stateDown
	}
	return stateOK
}

// checkWhatsApp reports the link state. A down link does not stop the platform
// from serving, but it stops codes and notifications from going out, so it
// counts toward health.
func (c *Checker) checkWhatsApp() state {
	if c.wa != nil && c.wa.Connected() {
		return stateOK
	}
	return stateDown
}

// checkStorage reports down when free space on the upload volume falls below the
// floor, since a new upload would then fail. It reads the filesystem stats
// directly; no upload is written.
func (c *Checker) checkStorage() state {
	var st syscall.Statfs_t
	if err := syscall.Statfs(c.uploadPath, &st); err != nil {
		return stateDown
	}
	free := st.Bavail * uint64(st.Bsize)
	if free < c.minFreeByte {
		return stateDown
	}
	return stateOK
}
