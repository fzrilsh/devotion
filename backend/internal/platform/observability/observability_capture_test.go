package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// fakeTransport records events the client hands over, proving a capture actually
// travels the full pipeline (capture, scrub, transport) without touching the
// network.
type fakeTransport struct {
	events chan *sentry.Event
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{events: make(chan *sentry.Event, 8)}
}

func (t *fakeTransport) Configure(sentry.ClientOptions)        {}
func (t *fakeTransport) SendEvent(e *sentry.Event)             { t.events <- e }
func (t *fakeTransport) Flush(time.Duration) bool              { return true }
func (t *fakeTransport) FlushWithContext(context.Context) bool { return true }
func (t *fakeTransport) Close()                                {}

// initFakeClient binds a real Sentry client to the fake transport. Cleanup
// restores the SDK's no-op client so tests do not share events or transports.
func initFakeClient(t *testing.T, tr *fakeTransport) {
	t.Helper()
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://key@sentry.invalid/1",
		Environment:      "test",
		AttachStacktrace: true,
		BeforeSend:       scrub,
		Transport:        tr,
	}); err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}
	t.Cleanup(func() {
		sentry.CurrentHub().BindClient(nil)
	})
}

// waitEvent pulls one event or fails after a short timeout, so a missing send
// fails the test instead of hanging it.
func waitEvent(t *testing.T, tr *fakeTransport) *sentry.Event {
	t.Helper()
	select {
	case e := <-tr.events:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("tidak ada event yang sampai ke transport dalam 2 detik")
		return nil
	}
}

// TestCaptureException_SendsToInitializedClient proves the fix for the dead
// integration: before this, nothing in the codebase ever called Capture, so a
// valid DSN still produced an empty project. CaptureException must hand the
// error through scrub to the transport with the exception intact.
func TestCaptureException_SendsToInitializedClient_FR082(t *testing.T) {
	tr := newFakeTransport()
	initFakeClient(t, tr)

	CaptureException(errors.New("gagal membaca alokasi"))

	e := waitEvent(t, tr)
	if len(e.Exception) != 1 {
		t.Fatalf("Exception len = %d, mau 1", len(e.Exception))
	}
	if e.Exception[0].Value != "gagal membaca alokasi" {
		t.Fatalf("Exception[0].Value = %q, mau pesan galat asli", e.Exception[0].Value)
	}
	if e.Environment != "test" {
		t.Fatalf("Environment = %q, mau test", e.Environment)
	}
}

// TestCaptureException_NoOpWithNoDSN proves callers can report errors when the
// client is disabled without producing a network event or panicking.
func TestCaptureException_NoOpWithNoDSN(t *testing.T) {
	tr := newFakeTransport()
	initFakeClient(t, tr)
	if err := sentry.Init(sentry.ClientOptions{}); err != nil {
		t.Fatalf("disable Sentry: %v", err)
	}

	CaptureException(errors.New("tidak boleh terkirim"))
	CapturePanic("ledakan")

	select {
	case e := <-tr.events:
		t.Fatalf("event bocor saat client mati: %+v", e)
	default:
	}
}

// TestCapturePanic_CarriesValueAndStack proves a recovered string panic reaches
// Sentry with its message and a stack attached by the SDK.
func TestCapturePanic_CarriesValueAndStack(t *testing.T) {
	tr := newFakeTransport()
	initFakeClient(t, tr)

	CapturePanic("dipakai sebagai nilai panic")

	e := waitEvent(t, tr)
	if e.Message != "dipakai sebagai nilai panic" {
		t.Fatalf("Message = %q, mau pesan panic", e.Message)
	}
	if len(e.Threads) != 1 || e.Threads[0].Stacktrace == nil || len(e.Threads[0].Stacktrace.Frames) == 0 {
		t.Fatal("Stacktrace kosong, mau frame dari AttachStacktrace")
	}
}

// TestCapturePanic_ScrubStillApplies proves the panic path goes through the same
// BeforeSend allowlist: fields the recoverer did not set cannot smuggle user
// data out (FR-082).
func TestCapturePanic_ScrubStillApplies_FR082(t *testing.T) {
	tr := newFakeTransport()
	initFakeClient(t, tr)

	CapturePanic("ledakan")

	e := waitEvent(t, tr)
	if e.Request != nil || e.User.ID != "" || len(e.Extra) != 0 {
		t.Fatalf("field di luar allowlist lolos: request=%v user=%+v extra=%v",
			e.Request != nil, e.User, e.Extra)
	}
	if e.Message != "ledakan" {
		t.Fatalf("Message = %q, mau ledakan", e.Message)
	}
}
