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
package admin

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" for whatsmeow's store

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"log/slog"
)

// Status is the observable state of the WhatsApp link. It carries no phone
// number by construction (FR-082): the fields are the link state, the pairing
// QR when unpaired, and the last error text.
type Status struct {
	Connected bool
	QRCode    string
	LastError string
}

// Manager owns the whatsmeow client and its lifecycle. It is safe for
// concurrent use: the HTTP handler reads status while the background goroutine
// mutates it. SendText satisfies notification.WhatsAppSender.
type Manager struct {
	log *slog.Logger

	db        *sql.DB
	container *sqlstore.Container
	client    *whatsmeow.Client

	mu        sync.RWMutex
	qrCode    string
	lastError string
}

// New opens the whatsmeow session store on the same Postgres database through a
// second database/sql handle (the "pgx" stdlib driver), upgrades the store
// schema, and builds the client over the first stored device or a fresh one. It
// does not connect: call Start to bring the link up and drive pairing. databaseURL
// is the same DSN the pool uses; the extra handle is capped small because it
// competes with the pool against max_connections (see docs/dependencies.md).
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

	m := &Manager{log: log, db: sqlDB, container: container}
	m.client = whatsmeow.NewClient(device, waLog.Noop)
	m.client.AddEventHandler(m.onEvent)
	return m, nil
}

// Start brings the link up and blocks until ctx is cancelled, then disconnects
// and closes the store handle. When the device is unpaired it drains the QR
// channel into the status so the admin page can render a code; once paired it
// simply connects. It is meant to run in its own goroutine started by serve.
func (m *Manager) Start(ctx context.Context) {
	if m.client.Store.ID == nil {
		qrChan, err := m.client.GetQRChannel(ctx)
		if err != nil {
			m.setError(err)
		} else {
			go m.pump(qrChan)
		}
	}
	if err := m.client.Connect(); err != nil {
		m.setError(err)
	}

	<-ctx.Done()
	m.client.Disconnect()
	_ = m.db.Close()
}

// pump copies QR codes into the status as whatsmeow emits them, and clears the
// code on success. The service number never appears here; only the opaque QR
// string does.
func (m *Manager) pump(qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			m.mu.Lock()
			m.qrCode = item.Code
			m.lastError = ""
			m.mu.Unlock()
		case whatsmeow.QRChannelSuccess.Event:
			m.mu.Lock()
			m.qrCode = ""
			m.mu.Unlock()
		default: // error, timeout: record and stop waiting for this cycle
			m.mu.Lock()
			m.qrCode = ""
			m.lastError = "pemasangan WhatsApp gagal, coba sambungkan ulang"
			m.mu.Unlock()
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

// SendText delivers one WhatsApp text to phone, satisfying
// notification.WhatsAppSender. phone is the recipient's number in bare digits
// (62...); it is turned into a JID on the default user server. An unpaired or
// disconnected link returns an error so the notification channel records a
// failed attempt rather than a silent drop.
func (m *Manager) SendText(ctx context.Context, phone, body string) error {
	jid := types.NewJID(phone, types.DefaultUserServer)
	msg := &waE2E.Message{Conversation: proto.String(body)}
	_, err := m.client.SendMessage(ctx, jid, msg)
	return err
}

// setError records the last error text without the service number. whatsmeow
// errors do not embed the number, so the message is safe to store.
func (m *Manager) setError(err error) {
	m.mu.Lock()
	m.lastError = err.Error()
	m.mu.Unlock()
}

// ensure Manager satisfies the store device pointer expectation at compile time.
var _ = store.Device{}
