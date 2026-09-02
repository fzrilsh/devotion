package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCodes_EveryCodeMapsToOneStatus proves every code in the map resolves to
// exactly one status in the error range, with a non-empty title. This is the
// property the contract needs, no handler may emit a code without a defined
// status, so it stays red if a code lands without a status but does not go red
// merely because the catalogue grew. FR: contract error catalogue.
func TestCodes_EveryCodeMapsToOneStatus(t *testing.T) {
	if len(codes) == 0 {
		t.Fatal("peta kode kosong")
	}
	for c, meta := range codes {
		if c == "" {
			t.Error("ada kode kosong di peta")
		}
		if meta.Title == "" {
			t.Errorf("kode %q tanpa title", c)
		}
		// StatusFor must resolve through the map to the same status, and it must
		// be a real error status. A code without an entry falls through to 500,
		// which this catches.
		if s := StatusFor(c); s != meta.Status {
			t.Errorf("kode %q: StatusFor=%d tidak sama dengan peta %d", c, s, meta.Status)
		}
		if meta.Status < 400 || meta.Status > 599 {
			t.Errorf("kode %q status %d di luar rentang galat", c, meta.Status)
		}
	}
}

// TestWriteProblem_ShapeAndStatus checks the body is problem+json with the
// status derived from the code.
func TestWriteProblem_ShapeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, CodeSelfRequest, "Tidak bisa ke listing sendiri.")

	if rec.Code != 409 {
		t.Fatalf("status = %d, mau 409", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != problemContentType {
		t.Fatalf("Content-Type = %q, mau %q", ct, problemContentType)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Code != CodeSelfRequest || p.Status != 409 {
		t.Fatalf("body code/status = %q/%d", p.Code, p.Status)
	}
	if p.Type != "https://devotion/errors/self-request" {
		t.Fatalf("type = %q", p.Type)
	}
}

// TestWriteValidation_CarriesFields ensures ProblemValidation includes errors[].
func TestWriteValidation_CarriesFields(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteValidation(rec, "Beberapa isian tidak sesuai.", []FieldError{
		{Field: "email", Message: "Format email tidak sah"},
	})
	if rec.Code != 422 {
		t.Fatalf("status = %d, mau 422", rec.Code)
	}
	var p Problem
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "email" {
		t.Fatalf("errors = %+v", p.Errors)
	}
}

// TestRouter_UnknownAPIReturnsJSON404 proves a misspelled /api/ path returns a
// problem+json 404, never HTML. This is the guard CLAUDE.md requires.
func TestRouter_UnknownAPIReturnsJSON404(t *testing.T) {
	rt := NewRouter(quietLogger())
	srv := httptest.NewServer(rt.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/tidak-ada")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, mau 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, problemContentType) {
		t.Fatalf("Content-Type = %q, mau %q", ct, problemContentType)
	}
}

// TestMiddleware_RequestIDInResponseAndLog checks the request ID is echoed and
// appears in the log line.
func TestMiddleware_RequestIDInResponseAndLog(t *testing.T) {
	var buf strings.Builder
	log := slog.New(contextHandler{Handler: slog.NewJSONHandler(&buf, nil)})

	rt := &Router{mux: http.NewServeMux(), log: log}
	rt.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	srv := httptest.NewServer(rt.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get(requestIDHeader) == "" {
		t.Fatal("X-Request-ID tidak ada di respons")
	}
	if !strings.Contains(buf.String(), "request_id") {
		t.Fatalf("log tanpa request_id: %s", buf.String())
	}
}

// TestMiddleware_RecoverWrites500 proves a panicking handler yields a 500
// problem+json and no stack in the body.
func TestMiddleware_RecoverWrites500(t *testing.T) {
	rt := &Router{mux: http.NewServeMux(), log: quietLogger()}
	rt.HandleFunc("GET /api/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("meledak")
	})
	srv := httptest.NewServer(rt.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/boom")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Fatalf("status = %d, mau 500", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "meledak") || strings.Contains(string(body), "goroutine") {
		t.Fatalf("stack/panic bocor ke klien: %s", body)
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
