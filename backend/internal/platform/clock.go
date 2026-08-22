// Package platform holds cross-cutting infrastructure shared by domain
// packages: the clock, configuration, and the like. Nothing here encodes
// product behavior.
package platform

import (
	"sync"
	"time"
)

// jakarta is the single time zone the application reasons about. Week
// boundaries and any displayed time are computed here, never from the host
// setting, so a misconfigured server cannot silently shift them.
var jakarta = mustLoadJakarta()

func mustLoadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic("platform: memuat zona waktu Asia/Jakarta: " + err.Error())
	}
	return loc
}

// Clock is injected into every service so time-dependent behavior (seven-day
// auto-confirm, window expiry) can be tested without waiting for real time.
type Clock interface {
	Now() time.Time
}

// SystemClock returns the current time localized to Asia/Jakarta.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().In(jakarta) }

// TestClock is a Clock whose time is set explicitly. It is safe for concurrent
// use: the scheduler ticker reads it from a goroutine, and without the mutex
// `go test -race` fails in confusing places.
type TestClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewTestClock returns a TestClock fixed at t, localized to Asia/Jakarta.
func NewTestClock(t time.Time) *TestClock {
	return &TestClock{now: t.In(jakarta)}
}

func (c *TestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Set replaces the current time.
func (c *TestClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t.In(jakarta)
}

// Advance moves the clock forward by d.
func (c *TestClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// WeekStart returns midnight of the Monday that begins t's week, in
// Asia/Jakarta. Capacity periods are stored as this date, and both scheduler
// layers use this one function so the week_start_is_monday constraint and the
// business logic never disagree on where a week begins.
func WeekStart(t time.Time) time.Time {
	t = t.In(jakarta)
	// time.Monday == 1, time.Sunday == 0; map to days since Monday.
	offset := (int(t.Weekday()) + 6) % 7
	y, m, d := t.Date()
	monday := time.Date(y, m, d-offset, 0, 0, 0, 0, jakarta)
	return monday
}
