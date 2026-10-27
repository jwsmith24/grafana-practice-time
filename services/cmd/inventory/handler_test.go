package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/catalog"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

const testToken = "test-token"

func serve(t *testing.T, h http.Handler, path string, hdr http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set(auth.Header, testToken)
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// T2.1-1
func TestHandler(t *testing.T) {
	h := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service), store{})

	rec := serve(t, h, "/inventory/SKU-1042", nil)
	if rec.Code != 200 {
		t.Fatalf("known item: status %d", rec.Code)
	}
	var got stockResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ItemID != "SKU-1042" || got.Stock != stockFor("SKU-1042") {
		t.Fatalf("got %+v", got)
	}
	again := serve(t, h, "/inventory/SKU-1042", nil)
	if again.Body.String() != rec.Body.String() {
		t.Fatal("stock is not deterministic")
	}

	if rec := serve(t, h, "/inventory/SKU-9999", nil); rec.Code != 404 {
		t.Fatalf("unknown item: status %d", rec.Code)
	}
	if rec := serve(t, h, "/healthz", nil); rec.Code != 200 {
		t.Fatalf("healthz: status %d", rec.Code)
	}
}

// T2.1-2
func TestLogSchema(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter(service, &buf)
	h := newHandler(logger, testToken, metrics.NewHTTP(service), store{})
	id := requestid.New()

	cases := []struct {
		name   string
		emit   func()
		kind   logschema.Kind
		status int
	}{
		{"startup", func() { logStartup(logger, "8080") }, logschema.Startup, 0},
		{"request ok", func() { serve(t, h, "/inventory/SKU-1001", http.Header{requestid.Header: {id}}) }, logschema.Request, 200},
		{"request 404", func() { serve(t, h, "/inventory/nope", nil) }, logschema.Request, 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf.Reset()
			c.emit()
			lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
			if len(lines) != 1 {
				t.Fatalf("want 1 line, got %d: %q", len(lines), buf.String())
			}
			if err := logschema.Validate([]byte(lines[0]), c.kind); err != nil {
				t.Fatalf("%v\nline: %s", err, lines[0])
			}
			if c.kind != logschema.Request {
				return
			}
			var l struct {
				Endpoint string `json:"endpoint"`
				Status   int    `json:"status"`
				ItemID   string `json:"item_id"`
			}
			_ = json.Unmarshal([]byte(lines[0]), &l)
			if l.Endpoint != "GET /inventory/{item_id}" || l.Status != c.status || l.ItemID == "" {
				t.Fatalf("got %+v", l)
			}
		})
	}

	t.Run("request id is propagated", func(t *testing.T) {
		buf.Reset()
		serve(t, h, "/inventory/SKU-1001", http.Header{requestid.Header: {id}})
		if !strings.Contains(buf.String(), `"request_id":"`+id+`"`) {
			t.Fatalf("request_id %s not logged: %s", id, buf.String())
		}
	})

	t.Run("healthz is not logged", func(t *testing.T) {
		buf.Reset()
		serve(t, h, "/healthz", nil)
		if buf.Len() != 0 {
			t.Fatalf("healthz logged: %s", buf.String())
		}
	})
}

// T2.1-5
func TestStartupLine(t *testing.T) {
	var buf bytes.Buffer
	logStartup(logging.NewWithWriter(service, &buf), "8080")
	var l struct {
		Event   string `json:"event"`
		Version string `json:"version"`
		Service string `json:"service"`
	}
	if err := json.Unmarshal(buf.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.Event != "startup" || l.Version != version || l.Service != service {
		t.Fatalf("got %+v", l)
	}
}

// T2.2-7 (inventory side)
func TestTokenRequired(t *testing.T) {
	h := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service), store{})
	for _, c := range []struct {
		name, path, token string
		want              int
	}{
		{"missing", "/inventory/SKU-1001", "", 401},
		{"bad", "/inventory/SKU-1001", "wrong", 401},
		{"good", "/inventory/SKU-1001", testToken, 200},
		{"healthz needs none", "/healthz", "", 200},
	} {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", c.path, nil)
			if c.token != "" {
				req.Header.Set(auth.Header, c.token)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d", rec.Code, c.want)
			}
		})
	}
	if auth.Valid("", "") {
		t.Fatal("an empty expected token must reject everything")
	}
}

// T4.2-1
func TestSlowLookups(t *testing.T) {
	t.Run("delay bounds", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			if d := lookupDelay("A"); d < 2*time.Second || d >= 4*time.Second {
				t.Fatalf("delay %s outside [2s, 4s)", d)
			}
		}
		for _, mode := range []string{"", "B", "C"} {
			if d := lookupDelay(mode); d != 0 {
				t.Fatalf("mode %q: delay %s, want 0", mode, d)
			}
		}
	})

	t.Run("slow but successful", func(t *testing.T) {
		var buf bytes.Buffer
		var slept []time.Duration
		st := store{
			mode:  func() string { return "A" },
			sleep: func(d time.Duration) { slept = append(slept, d); time.Sleep(d) },
		}
		h := newHandler(logging.NewWithWriter(service, &buf), testToken, metrics.NewHTTP(service), st)
		id := requestid.New()
		rec := serve(t, h, "/inventory/SKU-1001", http.Header{requestid.Header: {id}})
		if rec.Code != 200 {
			t.Fatalf("status %d, want 200", rec.Code)
		}
		if len(slept) != 1 || slept[0] < 2*time.Second || slept[0] >= 4*time.Second {
			t.Fatalf("slept %v, want one delay in [2s, 4s)", slept)
		}
		if err := logschema.Validate(bytes.TrimSpace(buf.Bytes()), logschema.Request); err != nil {
			t.Fatalf("%v\nline: %s", err, buf.String())
		}
		var l struct {
			Level      string  `json:"level"`
			Status     int     `json:"status"`
			RequestID  string  `json:"request_id"`
			DurationMS float64 `json:"duration_ms"`
		}
		if err := json.Unmarshal(buf.Bytes(), &l); err != nil {
			t.Fatal(err)
		}
		if l.Level != "INFO" || l.Status != 200 || l.RequestID != id || l.DurationMS <= 2000 {
			t.Fatalf("got %+v", l)
		}
	})

	t.Run("no delay outside A", func(t *testing.T) {
		st := store{
			mode:  func() string { return "" },
			sleep: func(d time.Duration) { t.Fatalf("slept %s", d) },
		}
		h := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service), st)
		if rec := serve(t, h, "/inventory/SKU-1001", nil); rec.Code != 200 {
			t.Fatalf("status %d", rec.Code)
		}
	})
}

// T4.3-1
func TestUnparsableRecord(t *testing.T) {
	var buf bytes.Buffer
	st := store{mode: func() string { return "C" }, sleep: func(time.Duration) {}}
	h := newHandler(logging.NewWithWriter(service, &buf), testToken, metrics.NewHTTP(service), st)

	rec := serve(t, h, "/inventory/SKU-1042", nil)
	if rec.Code != 500 {
		t.Fatalf("SKU-1042: status %d, want 500", rec.Code)
	}
	if err := logschema.Validate(bytes.TrimSpace(buf.Bytes()), logschema.Request); err != nil {
		t.Fatalf("%v\nline: %s", err, buf.String())
	}
	var l struct {
		Level  string `json:"level"`
		Status int    `json:"status"`
		ItemID string `json:"item_id"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(buf.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.Level != "ERROR" || l.Status != 500 || l.ItemID != "SKU-1042" || !strings.Contains(l.Error, "parse") {
		t.Fatalf("got %+v", l)
	}

	for _, id := range catalog.Items() {
		if id == "SKU-1042" {
			continue
		}
		if rec := serve(t, h, "/inventory/"+id, nil); rec.Code != 200 {
			t.Fatalf("%s: status %d, want 200", id, rec.Code)
		}
	}

	plain := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service), store{})
	if rec := serve(t, plain, "/inventory/SKU-1042", nil); rec.Code != 200 {
		t.Fatalf("SKU-1042 outside C: status %d, want 200", rec.Code)
	}
}
