// Package observability wires the error reporter. Sentry is the one external
// sink for panics and unexpected errors, and this package is the only place
// that talks to it, so the scrubbing rule lives in exactly one BeforeSend.
//
// The scrub is an allowlist, not a denylist: the event that leaves the process
// is rebuilt from a fixed set of fields known to be safe, and everything else
// is dropped. A denylist would have to name every field that must never leave,
// and it fails silently the moment the SDK or our own code adds a new one. The
// failure mode that allowlist guards against is identity-document data, session
// tokens, passwords, or the WhatsApp service number (FR-082) reaching a third
// party, so the safe default has to be "drop", not "forward".
package observability

import (
	"github.com/getsentry/sentry-go"
)

// Init starts Sentry when dsn is non-empty and returns a flush function to call
// on shutdown. An empty dsn (development, or a production that opted out) makes
// this a no-op with a flush that does nothing, so callers need no dsn check of
// their own. environment tags each event so production noise is separable from
// a staging run.
func Init(dsn, environment string) (flush func(), err error) {
	if dsn == "" {
		return func() {}, nil
	}
	err = sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: environment,
		BeforeSend:  scrub,
	})
	if err != nil {
		return func() {}, err
	}
	return func() { sentry.Flush(2 * 1e9) }, nil
}

// scrub rebuilds the outgoing event from an allowlist of fields that carry no
// user data. Anything not copied here never leaves the process: request bodies,
// headers, cookies, query strings, form fields, environment, and user identity
// are all dropped rather than filtered, so a new sensitive field added upstream
// is safe by default. The exception detail is our own error text, which the
// codebase writes and does not embed secrets in (config errors name variables
// not values, whatsmeow errors carry no number).
func scrub(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	safe := sentry.NewEvent()

	// Identity of the event itself, not of any user.
	safe.EventID = event.EventID
	safe.Timestamp = event.Timestamp
	safe.Level = event.Level
	safe.Platform = event.Platform
	safe.Environment = event.Environment
	safe.ServerName = event.ServerName
	safe.Release = event.Release
	safe.Logger = event.Logger

	// The message and the exception chain: our own text and stack frames. No
	// request, user, cookie, or header field is copied, so none can leak.
	safe.Message = event.Message
	safe.Exception = event.Exception
	safe.Threads = event.Threads

	// A fixed tag set, never the raw Extra/Contexts maps which may hold
	// request-derived values.
	safe.Tags = map[string]string{}
	if v, ok := event.Tags["request_id"]; ok {
		safe.Tags["request_id"] = v
	}

	return safe
}
