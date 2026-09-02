package admin

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
)

// waitPairingCleared polls the pairing flag so a test does not depend on the
// pump goroutine's scheduling. It fails the test rather than blocking forever.
func waitPairingCleared(t *testing.T, m *Manager) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !m.isPairing() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("siklus pemasangan tidak pernah berakhir")
}

// TestPump_TimeoutEndsCycleSoNextArmCanStart proves the batch running out leaves
// the manager ready to arm again. This is the defect behind a blank QR after a
// few minutes of uptime: whatsmeow emits QRChannelTimeout and closes the channel
// once its finite batch of codes expires, and if the manager stayed marked as
// pairing no later read would ever start a new cycle. FR-002, FR-052.
func TestPump_TimeoutEndsCycleSoNextArmCanStart_FR002_FR052(t *testing.T) {
	m := &Manager{log: quietLogger(), lifeCtx: context.Background()}
	ch := make(chan whatsmeow.QRChannelItem, 2)
	gen, stop := m.beginCycle()
	if !m.isPairing() {
		t.Fatal("siklus mau ditandai berjalan setelah beginCycle")
	}

	go m.pump(gen, stop, ch)
	ch <- whatsmeow.QRChannelItem{Event: whatsmeow.QRChannelEventCode, Code: "kode-pertama"}
	waitForStatus(t, m, "kode-pertama")

	ch <- whatsmeow.QRChannelTimeout
	waitPairingCleared(t, m)
	if got := m.Status().QRCode; got != "" {
		t.Fatalf("qr_code = %q, mau kosong setelah batch habis", got)
	}
	if got := m.Status().LastError; got != "" {
		t.Fatalf("last_error = %q, mau kosong: batch habis bukan kegagalan", got)
	}
}

// TestPump_StaleCycleCannotOverwriteFreshCode proves the generation fence. Two
// admins pressing reconnect at once leave an older pump alive for a moment; its
// writes must be dropped, or the page would show a code from a cycle whose
// socket is already gone and scanning it would fail. FR-002, FR-052.
func TestPump_StaleCycleCannotOverwriteFreshCode_FR002_FR052(t *testing.T) {
	m := &Manager{log: quietLogger(), lifeCtx: context.Background()}

	oldCh := make(chan whatsmeow.QRChannelItem, 2)
	oldGen, oldStop := m.beginCycle()
	go m.pump(oldGen, oldStop, oldCh)
	oldCh <- whatsmeow.QRChannelItem{Event: whatsmeow.QRChannelEventCode, Code: "kode-lama"}
	waitForStatus(t, m, "kode-lama")

	m.supersede()
	newCh := make(chan whatsmeow.QRChannelItem, 2)
	newGen, newStop := m.beginCycle()
	go m.pump(newGen, newStop, newCh)
	newCh <- whatsmeow.QRChannelItem{Event: whatsmeow.QRChannelEventCode, Code: "kode-baru"}
	waitForStatus(t, m, "kode-baru")

	// The retired pump is released by stop, but a code that raced past the close
	// must still be dropped on the generation check.
	m.setCode(oldGen, "kode-lama-lagi")
	if got := m.Status().QRCode; got != "kode-baru" {
		t.Fatalf("qr_code = %q, mau kode-baru: siklus lama tidak boleh menimpa", got)
	}
	m.clearCode(oldGen, "galat lama")
	if got := m.Status().LastError; got != "" {
		t.Fatalf("last_error = %q, mau kosong: galat siklus lama tidak boleh muncul", got)
	}
}

// TestWaitForCode_ReleasedByFirstCode proves a status read that arms a cycle
// blocks only until the first code arrives, so one page load is enough to show
// something scannable instead of an empty body the admin must refresh away.
// FR-002, FR-052.
func TestWaitForCode_ReleasedByFirstCode_FR002_FR052(t *testing.T) {
	m := &Manager{log: quietLogger(), lifeCtx: context.Background()}
	gen, _ := m.beginCycle()

	done := make(chan struct{})
	go func() {
		m.waitForCode()
		close(done)
	}()

	time.Sleep(5 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("waitForCode selesai sebelum ada kode")
	default:
	}

	m.setCode(gen, "kode-segar")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waitForCode tidak dilepas oleh kode pertama")
	}
	if got := m.Status().QRCode; got != "kode-segar" {
		t.Fatalf("qr_code = %q, mau kode-segar", got)
	}
}

// TestWaitForCode_ReleasedWhenCycleEnds proves a cycle that dies before emitting
// anything does not hold the request open for the full qrWait ceiling. FR-052.
func TestWaitForCode_ReleasedWhenCycleEnds_FR052(t *testing.T) {
	m := &Manager{log: quietLogger(), lifeCtx: context.Background()}
	gen, _ := m.beginCycle()

	done := make(chan struct{})
	go func() {
		m.waitForCode()
		close(done)
	}()

	time.Sleep(5 * time.Millisecond)
	m.endCycle(gen)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("waitForCode tidak dilepas saat siklus berakhir")
	}
}

// waitForStatus polls until the published QR equals want, so tests never sleep
// on a guess about goroutine scheduling.
func waitForStatus(t *testing.T, m *Manager, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status().QRCode == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("qr_code tidak pernah menjadi %q, terakhir %q", want, m.Status().QRCode)
}
