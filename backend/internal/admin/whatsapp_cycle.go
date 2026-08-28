package admin

import (
	"context"
	"errors"
	"time"

	"go.mau.fi/whatsmeow"
)

// errNoClient is returned when the link is asked to send before a client exists.
// It only happens in tests and during a failed boot; a real serve always has one.
var errNoClient = errors.New("tautan WhatsApp belum siap")

// currentClient reads the live client under the read lock. rebuild swaps it, so
// no caller may cache the pointer across an arm.
func (m *Manager) currentClient() *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

// context returns the process context pairing cycles are anchored to. A cycle
// must outlive the request that armed it, so a request context is never used.
// The zero value falls back to Background so a Manager built by a test literal
// still works.
func (m *Manager) context() context.Context {
	if m.lifeCtx == nil {
		return context.Background()
	}
	return m.lifeCtx
}

// isPairing reports whether a cycle is live and still has a code on offer.
func (m *Manager) isPairing() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pairing
}

// beginCycle marks a new pairing cycle live and returns its generation and stop
// channel. The generation stamps every write the cycle's pump makes, so a pump
// left over from a superseded cycle cannot clobber the current code.
func (m *Manager) beginCycle() (uint64, <-chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gen++
	m.pairing = true
	m.qrCode = ""
	m.firstCode = make(chan struct{})
	m.stop = make(chan struct{})
	return m.gen, m.stop
}

// supersede retires the live cycle: its pump is told to stop and the status no
// longer counts as pairing. The pump's own writes are already fenced by gen, so
// this only releases it from the channel.
func (m *Manager) supersede() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.releaseWaiters()
	m.pairing = false
}

// endCycle clears the pairing flag when gen is still the live cycle. A stale
// pump calls this too, and the generation check keeps it from unmarking a newer
// cycle that has already started.
func (m *Manager) endCycle(gen uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gen != m.gen {
		return
	}
	m.releaseWaiters()
	m.pairing = false
}

// setCode publishes a code from cycle gen and clears the last error, since a
// fresh code means the link is waiting to be scanned rather than broken.
func (m *Manager) setCode(gen uint64, code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gen != m.gen {
		return
	}
	m.qrCode = code
	m.lastError = ""
	m.releaseWaiters()
}

// clearCode drops the code at the end of cycle gen, recording lastErr when the
// cycle ended badly. An empty lastErr leaves the previous error text alone.
func (m *Manager) clearCode(gen uint64, lastErr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gen != m.gen {
		return
	}
	m.qrCode = ""
	if lastErr != "" {
		m.lastError = lastErr
	}
	m.releaseWaiters()
}

// releaseWaiters wakes anything blocked in waitForCode. It must be called with
// mu held, and closing once is enough because the channel is replaced per cycle.
func (m *Manager) releaseWaiters() {
	if m.firstCode != nil {
		close(m.firstCode)
		m.firstCode = nil
	}
}

// waitForCode blocks until the cycle publishes its first code, ends, or qrWait
// elapses. Without it the page load that armed the cycle would return an empty
// QR and the admin would have to refresh again for no reason.
func (m *Manager) waitForCode() {
	m.mu.RLock()
	ch := m.firstCode
	m.mu.RUnlock()
	if ch == nil {
		return
	}
	timer := time.NewTimer(qrWait)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
	case <-m.context().Done():
	}
}
