package scheduler

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

var baseTime = time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// countingJob returns a Job that adds 1 to counter each time it runs, under the
// given lock key.
func countingJob(key int32, counter *atomic.Int64) Job {
	return Job{
		Name:    "counting",
		LockKey: key,
		Run: func(ctx context.Context, conn *pgxpool.Conn) error {
			counter.Add(1)
			return nil
		},
	}
}

// TestRunOne_TwoSchedulersSameKey_RunsOnce_R07 proves the advisory lock keeps a
// job from running doubled: two schedulers on the same database, both holding
// the same job key, do one pass each, but only one wins the lock, so the shared
// counter rises by exactly one. This is the deploy-rollover guarantee: the old
// container skips rather than fires a second notification.
func TestRunOne_TwoSchedulersSameKey_RunsOnce_R07(t *testing.T) {
	pool := testdb.New(t, "scheduler_lock")
	ctx := context.Background()
	clock := platform.NewTestClock(baseTime)

	// One job holds the lock so the other cannot win while it runs. Use two
	// pools so each scheduler acquires the lock on its own session, matching two
	// separate containers.
	var counter atomic.Int64
	const key = lockKeyBase + 1

	// Block the winning job until both schedulers have tried, so the loser meets
	// a held lock rather than a released one.
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	holdJob := Job{
		Name:    "hold",
		LockKey: key,
		Run: func(ctx context.Context, conn *pgxpool.Conn) error {
			counter.Add(1)
			started <- struct{}{}
			<-release
			return nil
		},
	}

	s1 := New(pool, clock, quietLogger())
	s1.Register(holdJob)

	pool2 := poolLike(t, pool)
	s2 := New(pool2, clock, quietLogger())
	s2.Register(countingJob(key, &counter))

	// Start s1's single job in the background; it takes the lock and blocks.
	done1 := make(chan struct{})
	go func() {
		s1.runOne(ctx, holdJob)
		close(done1)
	}()
	<-started // s1 holds the lock now.

	// s2 tries the same key while s1 holds it: it must skip, not block.
	s2done := make(chan struct{})
	go func() {
		s2.runOne(ctx, countingJob(key, &counter))
		close(s2done)
	}()
	select {
	case <-s2done:
		// good: it returned promptly without waiting.
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler kedua memblokir pada lock, mau melewati")
	}

	if got := counter.Load(); got != 1 {
		t.Fatalf("counter = %d selagi lock dipegang, mau 1 (job kedua dilewati)", got)
	}

	close(release)
	<-done1

	if got := counter.Load(); got != 1 {
		t.Fatalf("counter akhir = %d, mau 1", got)
	}
}

// TestRunOne_LockReleased_R07 proves the advisory lock is released after a job
// finishes: no scheduler lock remains in pg_locks, so a later tick can win it
// again. A lock leaked on a different connection would show here as a job that
// can never run again.
func TestRunOne_LockReleased_R07(t *testing.T) {
	pool := testdb.New(t, "scheduler_release")
	ctx := context.Background()
	clock := platform.NewTestClock(baseTime)

	var counter atomic.Int64
	const key = lockKeyBase + 2
	s := New(pool, clock, quietLogger())
	job := countingJob(key, &counter)
	s.Register(job)

	s.runOne(ctx, job)
	if counter.Load() != 1 {
		t.Fatalf("job tidak jalan sekali, counter = %d", counter.Load())
	}

	if held := advisoryLockHeld(t, pool, key); held {
		t.Fatal("advisory lock masih dipegang setelah job selesai, mau terlepas")
	}

	// A second pass wins the lock again, proving it was truly released.
	s.runOne(ctx, job)
	if counter.Load() != 2 {
		t.Fatalf("job kedua tidak jalan, counter = %d (lock bocor?)", counter.Load())
	}
}

// advisoryLockHeld reports whether any session holds this scheduler job's
// advisory lock. classid/objid in pg_locks hold the two lock ints for the
// two-argument advisory lock form.
func advisoryLockHeld(t *testing.T, pool *pgxpool.Pool, key int32) bool {
	t.Helper()
	var held bool
	err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM pg_locks
			WHERE locktype = 'advisory' AND classid = $1 AND objid = $2
		)`, advisoryLockClass, key).Scan(&held)
	if err != nil {
		t.Fatalf("query pg_locks: %v", err)
	}
	return held
}

// poolLike opens a second pool against the same schema as an existing pool, so
// two schedulers contend for the same advisory lock as two separate sessions
// would. It reuses the first pool's config (DSN and search_path included).
func poolLike(t *testing.T, p *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	cfg := p.Config().Copy()
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("pool kedua: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
