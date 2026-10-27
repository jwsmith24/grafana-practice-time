package main

import (
	"context"
	"encoding/base64"
	"log/slog"
	"sync/atomic"
	"time"
)

// nextVersion is the release orders runs in mode B.
const nextVersion = "1.4.3"

// releaseCheckInterval is how often orders checks which release to run.
const releaseCheckInterval = 500 * time.Millisecond

// release tracks the version orders runs and logs a deploy event whenever
// it changes.
type release struct {
	logger  *slog.Logger
	mode    func() string
	version atomic.Value // string
}

func newRelease(logger *slog.Logger, mode func() string) *release {
	r := &release{logger: logger, mode: mode}
	r.version.Store(version)
	return r
}

// Version is the release currently running.
func (r *release) Version() string { return r.version.Load().(string) }

// sync switches to the release the lab mode calls for. The deploy line is
// written before the switch, so it precedes any request the new release
// serves.
func (r *release) sync() {
	want := version
	if r.mode != nil && r.mode() == "B" {
		want = nextVersion
	}
	if want == r.Version() {
		return
	}
	r.logger.Info("deploy completed", "event", "deploy", "version", want)
	r.version.Store(want)
}

// run calls sync on every tick until ctx is done.
func (r *release) run(ctx context.Context, tick <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			r.sync()
		}
	}
}

// tokenFor returns the service token as release v sends it.
func tokenFor(v, token string) string {
	if v == nextVersion {
		return base64.StdEncoding.EncodeToString([]byte(token))
	}
	return token
}
