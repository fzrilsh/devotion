package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	web "github.com/fzrilsh/devotion/backend"
	"github.com/fzrilsh/devotion/backend/internal/account"
	"github.com/fzrilsh/devotion/backend/internal/admin"
	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/listing"
	"github.com/fzrilsh/devotion/backend/internal/masterdata"
	"github.com/fzrilsh/devotion/backend/internal/notification"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/config"
	"github.com/fzrilsh/devotion/backend/internal/platform/health"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/migrate"
	"github.com/fzrilsh/devotion/backend/internal/platform/observability"
	"github.com/fzrilsh/devotion/backend/internal/platform/ratelimit"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
	"github.com/fzrilsh/devotion/backend/internal/platform/session"
	"github.com/fzrilsh/devotion/backend/internal/platform/tlsconf"
	"github.com/fzrilsh/devotion/backend/internal/search"
)

// devPort is the plain-HTTP listen port outside production. Production always
// listens on 443 with TLS (research R-01/R-06); development derives no benefit
// from TLS, so it binds a fixed high port that docker-compose does not use.
const devPort = ":8080"

// shutdownTimeout bounds graceful shutdown so a hung connection cannot block a
// deploy rollover forever.
const shutdownTimeout = 15 * time.Second

// runServe boots the HTTP server: load config, connect the pool, run migrations,
// build the router, wire the domain services, start the scheduler goroutine,
// then listen and serve the embedded SPA with graceful shutdown on SIGTERM.
func runServe(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	log := httpx.NewLogger()

	// Sentry is the only external error sink; a nil DSN (development, or a
	// production that opted out) makes Init a no-op. Its BeforeSend scrubs by
	// allowlist so no request, user, or identity-document field can leak.
	flush, err := observability.Init(cfg.SentryDSN, string(cfg.AppEnv))
	if err != nil {
		return err
	}
	defer flush()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Run(ctx, cfg.DatabaseURL, log); err != nil {
		return err
	}

	clock := platform.SystemClock{}

	// Cookie Secure stays on unless the base URL is plain HTTP, which only
	// happens in local development. The exception is logged loudly so a
	// misconfigured production never disables Secure without a trace.
	secure := !strings.HasPrefix(cfg.AppBaseURL, "http://")
	if !secure {
		log.Warn("cookie sesi tanpa atribut Secure: APP_BASE_URL memakai http, hanya untuk pengembangan lokal",
			"base_url", cfg.AppBaseURL)
	}

	sessions := session.New(pool, clock, secure)
	limiter := ratelimit.New(pool, clock)

	router := httpx.NewRouter(log)
	acc := account.New(pool, clock, sessions, limiter, nil)
	acc.Register(router)
	ls := listing.New(pool, clock)
	ls.Register(router, acc)
	search.New(pool, clock, ls).Register(router, acc)

	// The WhatsApp manager runs the whatsmeow client as a goroutine inside this
	// same process (research R-08), with its session store on the same Postgres
	// database, so Gate I stays at two services. It is the concrete WhatsAppSender
	// wired into notification below; its admin route exposes only the link state,
	// never the service number (FR-082). A store failure at boot is fatal: without
	// it the WhatsApp channel could never deliver.
	wa, err := admin.New(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	go wa.Start(ctx)
	wa.Register(router, acc)

	// The email sender exists only when Mailjet credentials are configured
	// (always in production, optional in development). A nil sender fails the
	// email channel's attempt rather than dropping it silently.
	var email notification.EmailSender
	if cfg.MailjetAPIKey != "" && cfg.MailjetSecret != "" && cfg.MailFrom != "" {
		email = notification.NewMailjetSender(cfg.MailFrom, cfg.MailjetAPIKey, cfg.MailjetSecret)
	}
	notif := notification.New(pool, clock, acc, email, wa)
	notif.Register(router)

	// masterdata registers after notif because its proposal decision path
	// enqueues a notification to the proposer (FR-061); the read routes are
	// public, POST /master/proposals is gated to the two business roles.
	masterdata.New(pool, clock, acc, notif).Register(router, acc)

	// GET /health probes the database, the WhatsApp link, and free space on the
	// upload volume. The free-space floor is one file's worth: below it a new
	// upload would fail, so the instance is reported unhealthy. It sits outside
	// /api/ and is public (security:[] in the contract).
	health.New(pool, wa, clock, cfg.UploadPath, cfg.UploadFileLimitMB).Register(router)

	if uncovered := router.UncoveredAPIRoutes(); len(uncovered) > 0 {
		return errors.New("rute /api tanpa keputusan peran: " + strings.Join(uncovered, ", "))
	}

	// The scheduler runs as a goroutine inside this same process (research
	// R-07), not a second container, so Gate I stays at two services. It stops
	// when the serve context is cancelled.
	sched := scheduler.New(pool, clock, log)
	sched.Register(notif.DeliverJob())
	go sched.Start(ctx)

	dist, err := fs.Sub(web.FS, "webdist")
	if err != nil {
		return err
	}
	static, err := httpx.NewStatic(dist, router.Handler())
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           static,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if cfg.IsProduction() {
		tlsCfg, err := tlsconf.Load(cfg.TLSCertPath, cfg.TLSKeyPath, cfg.CFClientCAPath)
		if err != nil {
			return err
		}
		srv.Addr = ":443"
		srv.TLSConfig = tlsCfg
	} else {
		srv.Addr = devPort
	}

	return listenAndServe(ctx, srv, cfg.IsProduction(), log)
}

// listenAndServe starts the server and blocks until the context is cancelled
// (SIGINT/SIGTERM) or the server errors, then drains in-flight requests within
// shutdownTimeout. A production server serves TLS; the certificate and key are
// already in srv.TLSConfig, so ServeTLS is called with empty paths.
func listenAndServe(ctx context.Context, srv *http.Server, production bool, log *slog.Logger) error {
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server menyala", "addr", srv.Addr, "tls", production)
		var err error
		if production {
			err = srv.ListenAndServeTLS("", "")
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-sigCtx.Done():
		log.Info("sinyal berhenti diterima, mematikan server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
