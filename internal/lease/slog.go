package lease

import (
	"context"
	"log/slog"
	"slices"

	"cloud.google.com/go/logging"
)

// slogger is a slog.Handler that writes to both stdout and the logger when enabled.
type slogger struct {
	logger       *logging.Logger
	lw           *Manager
	stdoutLogger slog.Handler
	attrs        []slog.Attr
	groups       []string
}

// Enabled returns true if the lease is enabled
//   - Always returns true to ensure logs are always written to stdout
func (s *slogger) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle writes a log record to both stdout and the logger when enabled.
func (s *slogger) Handle(ctx context.Context, r slog.Record) error {
	// build labels map
	r.AddAttrs(s.attrs...)
	labels := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		labels[a.Key] = a.Value.String()
		return true
	})

	// always log to stdout
	if err := s.stdoutLogger.Handle(ctx, r); err != nil {
		return err
	}

	// skip shipping to logger if lease is disabled and level is below ERROR
	if !s.lw.enabled.Load() && r.Level < slog.LevelError {
		return nil
	}

	s.logger.Log(logging.Entry{
		Timestamp: r.Time,
		Severity:  getSeverity(r.Level),
		Payload:   r.Message,
		Labels:    labels,
	})

	return nil
}

// WithAttrs returns a new handler with additional attributes.
func (s *slogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *s
	c.attrs = slices.Clone(s.attrs)
	c.attrs = append(c.attrs, attrs...)
	return &c
}

// WithGroup returns a new handler with an additional group.
func (s *slogger) WithGroup(g string) slog.Handler {
	c := *s
	c.groups = slices.Clone(s.groups)
	c.groups = append(c.groups, g)
	return &c
}

// getSeverity converts a slog.Level to a logging.Severity.
func getSeverity(l slog.Level) logging.Severity {
	switch l {
	case slog.LevelDebug:
		return logging.Debug
	case slog.LevelInfo:
		return logging.Info
	case slog.LevelWarn:
		return logging.Warning
	case slog.LevelError:
		return logging.Error
	default:
		return logging.Default
	}
}
