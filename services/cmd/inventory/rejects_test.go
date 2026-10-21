package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
)

type rejectLine struct {
	Level    string  `json:"level"`
	Msg      string  `json:"msg"`
	Caller   string  `json:"caller"`
	Rejected int     `json:"rejected"`
	WindowS  float64 `json:"window_s"`
}

func rejectLines(t *testing.T, buf *bytes.Buffer) []rejectLine {
	t.Helper()
	var out []rejectLine
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if l == "" || !strings.Contains(l, "auth token rejected") {
			continue
		}
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Fatalf("%v\nline: %s", err, l)
		}
		var r rejectLine
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// T4.4-2
func TestRejectedTokenWarning(t *testing.T) {
	t.Run("handler returns 401 and warns", func(t *testing.T) {
		var buf bytes.Buffer
		h := newHandler(logging.NewWithWriter(service, &buf), testToken, metrics.NewHTTP(service), store{})
		req := httptest.NewRequest("GET", "/inventory/SKU-1001", nil)
		req.Header.Set(auth.Header, "wrong")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Fatalf("status %d, want 401", rec.Code)
		}
		got := rejectLines(t, &buf)
		if len(got) != 1 || got[0].Level != "WARN" || got[0].Msg != "auth token rejected" || got[0].Caller == "" || got[0].Rejected != 1 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("at most one warning per window per caller", func(t *testing.T) {
		var buf bytes.Buffer
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		l := newRejectLog(logging.NewWithWriter(service, &buf))
		l.now = func() time.Time { return now }
		from := func(addr string) {
			req := httptest.NewRequest("GET", "/inventory/SKU-1001", nil)
			req.RemoteAddr = addr
			l.note(req)
		}

		from("10.0.0.2:4000") // first rejection warns at once
		for i := 0; i < 24; i++ {
			now = now.Add(200 * time.Millisecond)
			from("10.0.0.2:4001")
		}
		// 4.8 s later: still one warning.
		if got := rejectLines(t, &buf); len(got) != 1 {
			t.Fatalf("%d warnings within the window, want 1: %+v", len(got), got)
		}
		from("10.0.0.3:5000") // another caller has its own window
		now = now.Add(200 * time.Millisecond)
		from("10.0.0.2:4002") // 5.0 s after the first warning

		got := rejectLines(t, &buf)
		if len(got) != 3 {
			t.Fatalf("got %d warnings, want 3: %+v", len(got), got)
		}
		if got[0].Caller != "10.0.0.2" || got[0].Rejected != 1 || got[0].WindowS != 5 {
			t.Fatalf("first warning %+v", got[0])
		}
		if got[1].Caller != "10.0.0.3" || got[1].Rejected != 1 {
			t.Fatalf("second caller %+v", got[1])
		}
		if got[2].Caller != "10.0.0.2" || got[2].Rejected != 25 {
			t.Fatalf("after the window %+v, want 25 rejected", got[2])
		}
	})
}
