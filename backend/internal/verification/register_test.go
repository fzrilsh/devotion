package verification

import (
	"log/slog"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// TestRegister_AllRoutesGated proves every /api route this service mounts carries
// a role decision. serve.go fails startup if UncoveredAPIRoutes is non-empty, so
// this catches a forgotten gate at test time rather than at boot.
func TestRegister_AllRoutesGated(t *testing.T) {
	r := httpx.NewRouter(slog.New(slog.NewTextHandler(discard{}, nil)))
	svc := &Service{}
	svc.Register(r, &mockAuth{})

	if uncovered := r.UncoveredAPIRoutes(); len(uncovered) > 0 {
		t.Fatalf("rute /api tanpa gerbang peran: %v", uncovered)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
