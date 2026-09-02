// Package admin owns the WhatsApp service link the platform uses to deliver
// codes and notifications. The whatsmeow client runs as a goroutine inside the
// serve process (research R-08), never a second container, so Gate I stays at
// two services. Its session lives in the same Postgres database through a second
// database/sql handle (whatsmeow demands one), budgeted from the five
// connections reserved beyond the pgx pool.
//
// The service phone number never leaves this package as output: it is not in
// the status response, the logs, or Sentry (FR-082). Manager exposes only
// whether the link is connected, a QR code to scan when it is not, and the last
// error text.
//
// Pairing is armed on demand, not once at boot. One GetQRChannel call yields a
// finite batch of codes; when the batch runs out whatsmeow closes the channel,
// drops its event handler, and disconnects. An admin who opens the page minutes
// after startup would then see no code at all, which is why reading the status
// re-arms a cycle and why there is an explicit reconnect route.
package admin

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" for whatsmeow's store

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"log/slog"
)

// qrWait bounds how long a status read blocks waiting for the first code of a
// cycle it just armed. WhatsApp answers the handshake in well under a second;
// the ceiling exists so a dead network cannot hold the request open.
const qrWait = 5 * time.Second

// Status is the observable state of the WhatsApp link. It carries no phone
// number by construction (FR-082): the fields are the link state, the pairing
// QR when unpaired, and the last error text.
type Status struct {
	Connected bool
	QRCode    string
	LastError string
}

// Manager owns the whatsmeow client and its lifecycle. It is safe for
// concurrent use: the HTTP handler reads status and arms pairing while the
// background goroutine mutates it. SendText satisfies notification.WhatsAppSender.
type Manager struct {
	log *slog.Logger

	db        *sql.DB
	container *sqlstore.Container

	// lifeCtx is the process context. Pairing cycles must outlive the request
	// that armed them, so GetQRChannel is never handed a request context.
	lifeCtx context.Context

	// armMu serializes lifecycle work (disconnect, rebuild, GetQRChannel,
	// Connect) so two admins cannot arm competing cycles. It is never held
	// while mu is held.
	armMu sync.Mutex

	mu        sync.RWMutex
	client    *whatsmeow.Client
	qrCode    string
	lastError string
	// gen counts pairing cycles. A pump from a superseded cycle carries an old
	// gen and its writes are dropped, so a stale code can never overwrite a
	// fresh one.
	gen     uint64
	pairing bool
	// firstCode is closed when the live cycle emits its first code, and stop is
	// closed when the cycle is superseded. Both let a waiting reader and a
	// stranded pump finish instead of blocking forever.
	firstCode chan struct{}
	stop      chan struct{}
}

// New opens the whatsmeow session store on the same Postgres database through a
// second database/sql handle (the "pgx" stdlib driver), upgrades the store
// schema, and builds the client over the first stored device or a fresh one. It
// does not connect: call Start to bring the link up. databaseURL is the same DSN
// the pool uses; the extra handle is capped small because it competes with the
// pool against max_connections (see docs/dependencies.md). ctx must be the
// process context, since pairing cycles are anchored to it.
func New(ctx context.Context, databaseURL string, log *slog.Logger) (*Manager, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	// This handle draws from the five connections reserved beyond the pgx pool
	// (max_connections=20, pool=15). One connection is enough for the session
	// store's occasional reads and writes.
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	container := sqlstore.NewWithDB(sqlDB, "pgx", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	m := &Manager{log: log, db: sqlDB, container: container, lifeCtx: ctx}
	m.client = whatsmeow.NewClient(device, waLog.Noop)
	m.client.AddEventHandler(m.onEvent)
	return m, nil
}

// Start brings the link up and blocks until ctx is cancelled, then disconnects
// and closes the store handle. A paired device is connected here; an unpaired one
// is left alone, because a pairing cycle armed at boot expires unseen and burns
// pairing attempts nobody is watching. The admin page arms the cycle instead. It
// is meant to run in its own goroutine started by serve.
func (m *Manager) Start(ctx context.Context) {
	if cli := m.currentClient(); cli != nil && cli.Store.ID != nil {
		if err := cli.Connect(); err != nil {
			m.setError(err)
		}
	}

	<-ctx.Done()
	if cli := m.currentClient(); cli != nil {
		cli.Disconnect()
	}
	_ = m.db.Close()
}

// EnsureQR arms a pairing cycle when the link is unpaired and no cycle is live,
// then waits briefly for the first code so a single page load shows something to
// scan. A paired link returns at once. force discards any live cycle, which is
// what the reconnect route wants: the admin pressed the button because whatever
// is on screen is not working.
func (m *Manager) EnsureQR(force bool) {
	if m.arm(force) {
		m.waitForCode()
	}
}

// arm moves the link toward a usable state and reports whether it started a new
// pairing cycle. A paired device is reconnected rather than re-paired, since
// wiping a working session to show a QR would be the worse failure.
func (m *Manager) arm(force bool) bool {
	m.armMu.Lock()
	defer m.armMu.Unlock()

	cli := m.currentClient()
	if cli == nil {
		return false
	}

	// A device that still holds its JID is paired. It needs a socket, not a
	// code; whatsmeow reconnects on its own, and force bounces it by hand.
	if cli.Store.ID != nil && !cli.Store.Deleted {
		if force || !cli.IsConnected() {
			cli.Disconnect()
			if err := cli.Connect(); err != nil {
				m.setError(err)
			}
		}
		return false
	}

	if m.isPairing() && !force {
		return false // codes are still being emitted, the latest one is in the status
	}

	m.supersede()
	cli.Disconnect() // GetQRChannel refuses a client that is already connected

	// Every cycle gets a fresh client. The retired cycle's whatsmeow-internal
	// handler is still registered on the old one and would, on the next QR
	// event, start emitting into a channel nobody reads and then disconnect the
	// socket out from under the new cycle. Dropping the client drops the
	// handler. An unpaired device holds nothing in the database (NewDevice saves
	// only on successful pairing), and a logged out one was already wiped by
	// whatsmeow, so nothing is lost either way.
	cli = m.rebuild()

	qrChan, err := cli.GetQRChannel(m.context())
	if err != nil {
		m.setError(err)
		return false
	}
	gen, stop := m.beginCycle()
	go m.pump(gen, stop, qrChan)

	if err := cli.Connect(); err != nil {
		m.endCycle(gen)
		m.setError(err)
		return false
	}
	return true
}

// rebuild replaces the client with one over a brand new device and returns it.
// The caller has already disconnected the old client, which is then dropped
// whole so its event handlers, including a retired cycle's QR handler, go with
// it.
func (m *Manager) rebuild() *whatsmeow.Client {
	device := m.container.NewDevice()
	cli := whatsmeow.NewClient(device, waLog.Noop)
	cli.AddEventHandler(m.onEvent)

	m.mu.Lock()
	m.client = cli
	m.mu.Unlock()
	return cli
}

// pump copies QR codes into the status as whatsmeow emits them, and clears the
// code on success. Writes carry the cycle's gen so a superseded pump cannot
// overwrite a newer code. The service number never appears here; only the opaque
// QR string does.
func (m *Manager) pump(gen uint64, stop <-chan struct{}, qrChan <-chan whatsmeow.QRChannelItem) {
	defer m.endCycle(gen)
	for {
		select {
		case <-stop:
			return
		case item, ok := <-qrChan:
			if !ok {
				return
			}
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				m.setCode(gen, item.Code)
			case whatsmeow.QRChannelSuccess.Event:
				m.clearCode(gen, "")
				return
			case whatsmeow.QRChannelTimeout.Event:
				// The batch expired unscanned. Reading the status arms a new
				// cycle, so this is not worth reporting as a failure.
				m.clearCode(gen, "")
				return
			default: // pairing errors: record and stop waiting for this cycle
				m.clearCode(gen, "pemasangan WhatsApp gagal, coba sambungkan ulang")
				return
			}
		}
	}
}

// onEvent tracks the link's paired state. LoggedOut invalidates the session, so
// the QR must be shown again; Connected clears any stale QR and error. The event
// payloads carry no number worth logging, and none is logged.
func (m *Manager) onEvent(evt any) {
	switch evt.(type) {
	case *events.Connected:
		m.mu.Lock()
		m.qrCode = ""
		m.lastError = ""
		m.mu.Unlock()
	case *events.LoggedOut:
		m.mu.Lock()
		m.qrCode = ""
		m.lastError = "sesi WhatsApp keluar, pindai ulang kode QR"
		m.mu.Unlock()
	}
}

// Status reports the current link state without ever exposing the service
// number (FR-082). Connected is the live client state; QRCode and LastError are
// the guarded fields.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connected := m.client != nil && m.client.IsConnected() && m.client.IsLoggedIn()
	return Status{
		Connected: connected,
		QRCode:    m.qrCode,
		LastError: m.lastError,
	}
}

// Connected reports whether the link is up, for the /health probe. It reads the
// same live client state Status does but exposes only the bit, never the QR or
// error text, and never the service number (FR-082).
func (m *Manager) Connected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client != nil && m.client.IsConnected() && m.client.IsLoggedIn()
}

// SendText delivers one WhatsApp text to phone, satisfying
// notification.WhatsAppSender. phone is the recipient's number in bare digits
// (62...); it is turned into a JID on the default user server. An unpaired or
// disconnected link returns an error so the notification channel records a
// failed attempt rather than a silent drop.
func (m *Manager) SendText(ctx context.Context, phone, body string) error {
	cli := m.currentClient()
	if cli == nil {
		return errNoClient
	}
	jid := types.NewJID(phone, types.DefaultUserServer)
	msg := &waE2E.Message{Conversation: proto.String(body)}
	_, err := cli.SendMessage(ctx, jid, msg)
	return err
}

// setError records the last error text without the service number. whatsmeow
// errors do not embed the number, so the message is safe to store.
func (m *Manager) setError(err error) {
	m.mu.Lock()
	m.lastError = err.Error()
	m.mu.Unlock()
}
