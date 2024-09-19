package lease

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
)

// Document represents a Firestore document representing a lease.
type Document struct {
	ExpireAt time.Time
	User     string
	Reason   string
}

// Manager handles a lease document and manages the lease state.
type Manager struct {
	logger          *logging.Logger
	guaranteedUntil time.Time

	enabled     atomic.Bool
	expireTimer *time.Timer
}

// NewManager creates a new lease watcher.
//   - guaranteedUntil is the time until which the lease is guaranteed to be active
//   - if guaranteedUntil is in the past, the lease is disabled immediately
//   - if guaranteedUntil is in the future, the lease is enabled until that time
//   - to handle changes to the lease, WatchLease must be called
//   - start the lease manager in a goroutine to watch the lease until the context is canceled
func NewManager(ctx context.Context, logger *logging.Logger, guaranteedUntil time.Time, docRef *firestore.DocumentRef) *Manager {
	lw := &Manager{
		logger:          logger,
		guaranteedUntil: guaranteedUntil,

		enabled: atomic.Bool{},
	}

	if guaranteedUntil.After(time.Now().UTC()) {
		lw.expireAfter(guaranteedUntil)
	}

	go lw.watchLeaseWithRetry(ctx, docRef)

	return lw
}

// watchLeaseWithRetry watches a lease document for changes and updates the lease state.
//   - runs until the context is canceled
//   - retries every 5 seconds if the lease watcher fails
func (m *Manager) watchLeaseWithRetry(ctx context.Context, docRef *firestore.DocumentRef) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		err := m.watchLease(ctx, docRef)
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			return
		default:
			fmt.Fprintln(os.Stderr, "Failed to watch lease:", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C: // retry
		}
	}
}

// watchLease watches a lease document for changes and updates the lease state.
func (m *Manager) watchLease(ctx context.Context, docRef *firestore.DocumentRef) error {
	fmt.Fprintln(os.Stderr, "===  WATCH LEASE", docRef.Path)
	iter := docRef.Snapshots(ctx)
	defer iter.Stop()
	for {
		snapshot, err := iter.Next()
		switch {
		case err == io.EOF,
			err == context.DeadlineExceeded,
			err == context.Canceled:
			return nil
		case err != nil:
			fmt.Fprintln(os.Stderr, "Failed to get snapshot:", err)
			return err
		}

		// if the snapshot does not yet exist, espire after the guaranteedUntil time
		// for leases that are deleted after the guaranteedUntil time, this will disable the lease immediately
		if !snapshot.Exists() {
			m.expireAfter(m.guaranteedUntil)
			continue
		}

		var lease Document
		if err := snapshot.DataTo(&lease); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to parse lease:", err)
			continue
		}

		m.expireAfter(lease.ExpireAt)
		if lease.ExpireAt.After(m.guaranteedUntil) {
			fmt.Fprintf(os.Stderr, "=== LEASE EXTENDED, expires in %s | user=%q reason=%q\n", time.Until(lease.ExpireAt).Round(time.Millisecond*100), lease.User, lease.Reason)
		}
	}
}

func (m *Manager) enable() {
	m.enabled.Store(true)
}

func (m *Manager) disable() {
	m.enabled.Store(false)
}

// expireAfter sets a new lease expiration time, resetting the lease timer
//   - respects the guaranteedUntil time, even if the lease is shorter
func (m *Manager) expireAfter(expire time.Time) {
	// ensure guaranteedUntil is always respected, even if the lease is shorter
	if expire.Before(m.guaranteedUntil) {
		expire = m.guaranteedUntil
	}

	// cancel the previous timer if it was unfired
	if m.expireTimer != nil && !m.expireTimer.Stop() {
		select {
		case <-m.expireTimer.C:
		default:
		}
	}

	// already expired, disable immediately
	if expire.Before(time.Now().UTC()) {
		fmt.Fprintln(os.Stderr, "=== LEASE EXPIRED")
		m.disable()
		return
	}

	// enable and set a new timer
	m.enable()

	m.expireTimer = time.AfterFunc(time.Until(expire), func() {
		fmt.Fprintln(os.Stderr, "=== LEASE EXPIRED")
		m.disable()
	})
}

// Write writes a log message directly to the logger if the lease is active
//   - if the lease is not active, the message is discarded
func (m *Manager) Write(p []byte) (n int, err error) {
	if m.enabled.Load() {
		return m.logger.StandardLogger(logging.Info).Writer().Write(p)
	}
	return len(p), nil
}

// StdoutWriter returns an io.Writer that writes to both stdout and the logger.
//   - it writes to stdout only when the lease is enabled or the initial lease time has not yet expired
//   - logs are all written as INFO level
func (m *Manager) StdoutWriter() io.Writer {
	log := m.logger.StandardLogger(logging.Info).Writer()
	return &toggleableWriter{
		leaser:   m,
		upstream: io.MultiWriter(os.Stdout, log),
		fallback: os.Stdout,
	}
}

// StderrWriter returns an io.Writer that writes to both stderr and the logger.
//   - it always writes all messages to stderr and the logger, regardless of the lease state
//   - logs are all written as ERROR level
func (m *Manager) StderrWriter() io.Writer {
	log := m.logger.StandardLogger(logging.Error).Writer()
	return io.MultiWriter(os.Stderr, log)
}

// SlogLogger returns a slog.Logger that writes to both stdout and the logger.
//   - always logs to stdout
//   - logs to the logger only when the lease is enabled or the initial lease time has not yet expired
func (m *Manager) SlogLogger() *slog.Logger {
	return slog.New(&slogger{
		logger:       m.logger,
		lw:           m,
		stdoutLogger: slog.NewTextHandler(os.Stdout, nil),
	})
}

// toggleableWriter is an io.Writer that writes to an upstream writer when the lease is enabled.
type toggleableWriter struct {
	leaser   *Manager
	upstream io.Writer
	fallback io.Writer
}

// Write writes to the upstream writer if the lease is enabled.
// otherwise, it writes to the fallback writer if it is set.
// otherwise, it discards the message.
func (tw *toggleableWriter) Write(p []byte) (n int, err error) {
	if tw.leaser.enabled.Load() {
		return tw.upstream.Write(p)
	}
	if tw.fallback != nil {
		return tw.fallback.Write(p)
	}
	return len(p), nil
}
