package main

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// rejectWindow is the shortest gap between two rejection warnings for one
// caller.
const rejectWindow = 5 * time.Second

// rejectLog warns about rejected service tokens, at most once per window per
// caller, with the number of rejections since its last warning.
type rejectLog struct {
	logger *slog.Logger
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	callers map[string]*rejects
}

type rejects struct {
	last  time.Time
	count int
}

func newRejectLog(logger *slog.Logger) *rejectLog {
	return &rejectLog{logger: logger, window: rejectWindow, now: time.Now, callers: map[string]*rejects{}}
}

func (l *rejectLog) note(r *http.Request) {
	caller, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		caller = r.RemoteAddr
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	c := l.callers[caller]
	if c == nil {
		c = &rejects{}
		l.callers[caller] = c
	}
	c.count++
	now := l.now()
	if !c.last.IsZero() && now.Sub(c.last) < l.window {
		return
	}
	l.logger.Warn("auth token rejected", "caller", caller, "rejected", c.count, "window_s", l.window.Seconds())
	c.last, c.count = now, 0
}
