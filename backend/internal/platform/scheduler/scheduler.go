// Package scheduler is the in-process layer 2 of the two-layer scheduler
// (research.md R-07). It is a single time.Ticker in a goroutine that serve
// starts, not a second OS process, cron entry, or container, so the two-service
// rule (CLAUDE.md rule 1, Gate I) holds. Every registered job runs on each tick
// wrapped in pg_try_advisory_lock: during a deploy rollover two containers
// briefly overlap, and the old one must skip a job rather than queue behind the
// new one, which would fire the same notification twice.
//
// The deadline arithmetic these jobs act on lives in internal/order, shared with
// the compute-on-read layer, so both layers agree on every boundary instant.
// Job registration is empty here; T023 (notification) registers the real jobs.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// TickInterval is the fixed cadence of the ticker. Five minutes is fine grained
// enough for day-scale deadlines (FR-068 is a 7-day window) while keeping the
// wake-ups cheap on a 2GB box.
const TickInterval = 5 * time.Minute

// advisoryLockClass namespaces every scheduler lock, kept distinct from the
// migrate runner's class so the two never collide. Each job supplies its own
// LockKey as the second lock int; the pair (class, key) identifies one job.
const advisoryLockClass int32 = 54701302

// Job LockKey constants live in one block so the whole set of jobs is visible at
// once and two jobs cannot silently share a key. They are literal, never derived
// from a name, so a rename cannot shift a lock and let two instances run a job
// at once. Real jobs are registered by T023; the block starts empty apart from
// the base so the const type is fixed.
const (
	// lockKeyBase is not a job; it anchors the int32 const type for LockKey so
	// job keys are declared as, e.g., LockKeyAutoConfirm = lockKeyBase + 1.
	lockKeyBase int32 = 0
)

// Job is one unit of scheduled work. Run receives the pinned connection that
// already holds the job's advisory lock, so a job never has to reason about the
// lock itself. It must return promptly; a slow job delays every later job on the
// same tick.
type Job struct {
	// Name labels the job in logs.
	Name string
	// LockKey is the job's advisory-lock object id, unique across jobs.
	LockKey int32
	// Run does the work. ctx is the tick context; conn is the same connection
	// the lock was taken on, so queries share the session that holds it.
	Run func(ctx context.Context, conn *pgxpool.Conn) error
}

// Scheduler owns the pool, clock, logger, and the registered jobs. It carries no
// per-tick state, so one instance runs for the process lifetime.
type Scheduler struct {
	pool  *pgxpool.Pool
	clock platform.Clock
	log   *slog.Logger
	jobs  []Job
}

// New builds a Scheduler over pool. clock is injected so a test drives ticks by
// advancing time rather than sleeping; log carries the request-id-free process
// logger.
func New(pool *pgxpool.Pool, clock platform.Clock, log *slog.Logger) *Scheduler {
	return &Scheduler{pool: pool, clock: clock, log: log}
}

// Register adds a job to the set run on each tick. It is called at startup,
// before Start, so the slice needs no locking.
func (s *Scheduler) Register(j Job) {
	s.jobs = append(s.jobs, j)
}

// Start runs the ticker until ctx is cancelled. It runs one pass immediately so
// a just-started process does not wait a full interval before the first check,
// then one pass per tick. It blocks, so serve launches it in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info("penjadwal menyala", "interval", TickInterval.String(), "jobs", len(s.jobs))
	s.runAll(ctx)
	ticker := time.NewTicker(TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.log.Info("penjadwal berhenti")
			return
		case <-ticker.C:
			s.runAll(ctx)
		}
	}
}

// runAll runs every registered job once, isolating each so one failure does not
// stop the rest.
func (s *Scheduler) runAll(ctx context.Context) {
	for _, j := range s.jobs {
		s.runOne(ctx, j)
	}
}

// runOne acquires a pool connection, tries the job's advisory lock on it, and
// runs the job only if the lock was won. The lock is taken and released on the
// same connection: a session-scoped advisory lock released on a different
// connection is never released, and the leak shows only as a job silently
// skipped forever. Release uses context.Background() so a cancelled tick still
// unlocks.
func (s *Scheduler) runOne(ctx context.Context, j Job) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		s.log.Error("penjadwal: ambil koneksi", "job", j.Name, "error", err)
		return
	}
	defer conn.Release()

	var acquired bool
	if err := conn.QueryRow(ctx,
		"SELECT pg_try_advisory_lock($1, $2)", advisoryLockClass, j.LockKey).Scan(&acquired); err != nil {
		s.log.Error("penjadwal: pg_try_advisory_lock", "job", j.Name, "error", err)
		return
	}
	if !acquired {
		// Another instance holds it (deploy rollover); skip, do not queue.
		s.log.Debug("penjadwal: job dilewati, lock dipegang proses lain", "job", j.Name)
		return
	}
	defer func() {
		if _, err := conn.Exec(context.Background(),
			"SELECT pg_advisory_unlock($1, $2)", advisoryLockClass, j.LockKey); err != nil {
			s.log.Error("penjadwal: melepas advisory lock", "job", j.Name, "error", err)
		}
	}()

	if err := j.Run(ctx, conn); err != nil {
		s.log.Error("penjadwal: job gagal", "job", j.Name, "error", err)
	}
}
