package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
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
	h := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service))

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
	h := newHandler(logger, testToken, metrics.NewHTTP(service))
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
	h := newHandler(slog.New(slog.DiscardHandler), testToken, metrics.NewHTTP(service))
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
