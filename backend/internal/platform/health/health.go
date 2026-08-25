// Package health serves GET /api/health, the liveness probe the container
// healthcheck and any external uptime monitor read. It checks the three
// dependencies a running serve process cannot work without: the database, the
// WhatsApp link, and the upload volume. Any failing check makes the response
// 503 so a degraded instance is pulled out of rotation, and the body reports
// each dependency's state so an operator sees which one broke without shelling
// into the box.
//
// The response carries no secrets by construction: dependency states are fixed
// enums, never an error string, connection URL, or the WhatsApp service number
// (FR-082).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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

// nearFullRatio is the fraction of the total upload quota above which storage is
// reported near_full: still serving, but an operator should act before it fills.
const nearFullRatio = 0.9

// Checker holds the dependencies GET /health probes. limitBytes is the total
// upload quota; usage is compared against it to report ok, near_full, or full.
type Checker struct {
	db         Pinger
	wa         WhatsAppLink
	clock      platform.Clock
	uploadPath string
	version    string
	limitBytes int64
}

// New builds a Checker. uploadPath is the directory uploads land in; version is
// the build identifier echoed in the body; totalLimitMB is the total upload
// quota in megabytes.
func New(db Pinger, wa WhatsAppLink, clock platform.Clock, uploadPath, version string, totalLimitMB int) *Checker {
	return &Checker{
		db:         db,
		wa:         wa,
		clock:      clock,
		uploadPath: uploadPath,
		version:    version,
		limitBytes: int64(totalLimitMB) * 1024 * 1024,
	}
}

// storageState is the storage dependency block. It reports a coarse status plus
// the numbers behind it so an operator sees how close the volume is to full.
type storageState struct {
	Status string `json:"status"`
	UsedMB int64  `json:"used_mb"`
	LimitMB int64 `json:"limit_mb"`
}

// dependencies is the per-dependency detail block. Database and whatsapp are
// fixed enums; storage is an object.
type dependencies struct {
	Database string       `json:"database"`
	WhatsApp string       `json:"whatsapp"`
	Storage  storageState `json:"storage"`
}

// response is the /health body, matching the Health contract schema. status is
// "ok" only when every dependency is, "degraded" otherwise.
type response struct {
	Status       string       `json:"status"`
	Version      string       `json:"version"`
	Time         time.Time    `json:"time"`
	Dependencies dependencies `json:"dependencies"`
}

// Register wires GET /api/health as a public route (security:[] in the
// contract). The contract's servers url carries the /api prefix, so the /health
// path there resolves to /api/health; registering it under /api/ keeps it in the
// same namespace the SPA never claims. It is exempt from the uncovered-route
// check because Public states the no-auth decision explicitly rather than
// leaving the gate undeclared.
func (c *Checker) Register(r *httpx.Router) {
	r.Public("GET /api/health", c.handle)
}

// handle probes every dependency and reports 200 when all pass, 503 when any
// fails. Every check always runs so the body reflects the full state, not just
// the first failure.
func (c *Checker) handle(w http.ResponseWriter, r *http.Request) {
	deps := dependencies{
		Database: c.checkDB(r.Context()),
		WhatsApp: c.checkWhatsApp(),
		Storage:  c.checkStorage(),
	}

	healthy := deps.Database == "ok" &&
		deps.WhatsApp == "connected" &&
		deps.Storage.Status != "full"

	body := response{Version: c.version, Time: c.clock.Now(), Dependencies: deps}
	code := http.StatusOK
	if healthy {
		body.Status = "ok"
	} else {
		body.Status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// checkDB reports whether the pool can reach Postgres within a short deadline,
// so a hung database cannot hang the health probe itself.
func (c *Checker) checkDB(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := c.db.Ping(ctx); err != nil {
		return "fail"
	}
	return "ok"
}

// checkWhatsApp reports the link state. A down link does not stop the platform
// from serving, but it stops codes and notifications from going out, so it
// counts toward health.
func (c *Checker) checkWhatsApp() string {
	if c.wa != nil && c.wa.Connected() {
		return "connected"
	}
	return "disconnected"
}

// checkStorage reports how full the upload volume is against its quota. A path
// that cannot be read is reported full and drives 503, since a new upload would
// fail there just the same.
func (c *Checker) checkStorage() storageState {
	used, err := dirSize(c.uploadPath)
	if err != nil {
		return storageState{Status: "full", UsedMB: 0, LimitMB: c.limitBytes / (1024 * 1024)}
	}
	s := storageState{
		UsedMB:  used / (1024 * 1024),
		LimitMB: c.limitBytes / (1024 * 1024),
	}
	switch {
	case used >= c.limitBytes:
		s.Status = "full"
	case float64(used) >= float64(c.limitBytes)*nearFullRatio:
		s.Status = "near_full"
	default:
		s.Status = "ok"
	}
	return s
}

// dirSize sums the logical size of every regular file under root. It walks
// rather than reading filesystem free space so the number reflects the platform
// quota, not the whole disk the VPS shares with Postgres.
func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
