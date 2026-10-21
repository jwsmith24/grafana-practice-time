// Package logging builds the JSON slog logger shared by the lab services.
//
// Every line carries time (RFC 3339, millisecond precision, UTC), level
// (lowercase), msg, and service. Output goes to stdout and to
// <dir>/<service>.log, which is truncated when the logger is created.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TimeFormat is RFC 3339 with milliseconds, always in UTC.
const TimeFormat = "2006-01-02T15:04:05.000Z07:00"

// New returns a logger for service writing to stdout and dir/<service>.log.
// The returned closer closes the log file.
func New(service, dir string, stdout io.Writer) (*slog.Logger, io.Closer, error) {
	path := filepath.Join(dir, service+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}
	return NewWithWriter(service, io.MultiWriter(stdout, f)), f, nil
}

// NewWithWriter returns a logger for service writing to w.
func NewWithWriter(service string, w io.Writer) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	})
	return slog.New(h).With("service", service)
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String(slog.TimeKey, t.UTC().Format(TimeFormat))
		}
	case slog.LevelKey:
		if l, ok := a.Value.Any().(slog.Level); ok {
			return slog.String(slog.LevelKey, strings.ToLower(l.String()))
		}
	}
	return a
}
