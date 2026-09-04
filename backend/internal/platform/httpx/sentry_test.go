package httpx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

type sentryTestTransport struct {
	events chan *sentry.Event
}

func (t *sentryTestTransport) Configure(sentry.ClientOptions)        {}
func (t *sentryTestTransport) SendEvent(e *sentry.Event)             { t.events <- e }
func (t *sentryTestTransport) Flush(time.Duration) bool              { return true }
func (t *sentryTestTransport) FlushWithContext(context.Context) bool { return true }
func (t *sentryTestTransport) Close()                                {}

func initSentryForHTTPTest(t *testing.T) *sentryTestTransport {
	t.Helper()
	transport := &sentryTestTransport{events: make(chan *sentry.Event, 8)}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://key@sentry.invalid/1",
		Environment:      "test",
		AttachStacktrace: true,
		Transport:        transport,
	}); err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}
	t.Cleanup(func() {
		sentry.CurrentHub().BindClient(nil)
	})
	return transport
}

func waitSentryHTTPEvent(t *testing.T, transport *sentryTestTransport) *sentry.Event {
	t.Helper()
	select {
	case event := <-transport.events:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("tidak ada event Sentry dalam 2 detik")
		return nil
	}
}

// TestRecover_CapturesPanic_FR082 proves a recovered HTTP panic is reported to
// Sentry while the response remains a generic problem body.
func TestRecover_CapturesPanic_FR082(t *testing.T) {
	transport := initSentryForHTTPTest(t)
	rt := &Router{mux: http.NewServeMux(), log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	rt.HandleFunc("GET /api/boom", func(http.ResponseWriter, *http.Request) {
		panic("meledak")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/boom", nil)
	req.Header.Set(requestIDHeader, "req-sentry-panic")
	res := httptest.NewRecorder()
	rt.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, mau 500", res.Code)
	}
	if strings.Contains(res.Body.String(), "meledak") {
		t.Fatalf("panic bocor ke respons: %s", res.Body.String())
	}
	event := waitSentryHTTPEvent(t, transport)
	if event.Message != "meledak" {
		t.Fatalf("event message = %q, mau pesan panic", event.Message)
	}
	if event.Tags["request_id"] != "req-sentry-panic" {
		t.Fatalf("request_id tag = %q, mau req-sentry-panic", event.Tags["request_id"])
	}
}

// TestWriteInternal_CapturesCause_FR082 proves an unexpected handler error is
// reported before its details are hidden behind the generic 500 response.
func TestWriteInternal_CapturesCause_FR082(t *testing.T) {
	transport := initSentryForHTTPTest(t)
	res := httptest.NewRecorder()
	cause := errors.New("database unavailable")

	WriteInternal(res, cause)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, mau 500", res.Code)
	}
	event := waitSentryHTTPEvent(t, transport)
	if len(event.Exception) != 1 || event.Exception[0].Value != cause.Error() {
		t.Fatalf("exception = %+v, mau %q", event.Exception, cause.Error())
	}
}

// TestWriteInternal_CapturesRequestID_FR082 proves a 500 emitted from a normal
// request carries the same correlation ID as the request log.
func TestWriteInternal_CapturesRequestID_FR082(t *testing.T) {
	transport := initSentryForHTTPTest(t)
	rt := &Router{mux: http.NewServeMux(), log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	rt.HandleFunc("GET /api/failure", func(w http.ResponseWriter, r *http.Request) {
		WriteInternal(w, errors.New("database unavailable"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/failure", nil)
	req.Header.Set(requestIDHeader, "req-sentry-error")
	res := httptest.NewRecorder()
	rt.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, mau 500", res.Code)
	}
	event := waitSentryHTTPEvent(t, transport)
	if event.Tags["request_id"] != "req-sentry-error" {
		t.Fatalf("request_id tag = %q, mau req-sentry-error", event.Tags["request_id"])
	}
}
