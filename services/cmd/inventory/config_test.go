package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
)

// T2.4-1
func TestDeprecatedConfigWarning(t *testing.T) {
	if configReloadInterval != 2*time.Second {
		t.Fatalf("reload interval %s, want 2s", configReloadInterval)
	}
	var buf bytes.Buffer
	logger := logging.NewWithWriter(service, &buf)
	tick := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		watchConfig(ctx, logger, settings, tick)
		close(done)
	}()
	// 10 s of a 2 s ticker.
	for i := range 5 {
		tick <- time.Unix(int64(2*(i+1)), 0)
	}
	cancel()
	<-done

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("want 5 lines, got %d:\n%s", len(lines), buf.String())
	}
	for _, l := range lines {
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Fatal(err)
		}
		var v struct{ Level, Msg string }
		_ = json.Unmarshal([]byte(l), &v)
		if v.Level != "WARN" || v.Msg != "config key ttl_seconds is deprecated, use cache_ttl" {
			t.Fatalf("got level=%q msg=%q", v.Level, v.Msg)
		}
	}
}
