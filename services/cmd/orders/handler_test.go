package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

const testToken = "test-token"

// fakeInventory records the headers it receives and replies with status
// (and stock on 200) after delay.
type fakeInventory struct {
	status int
	stock  int
	delay  time.Duration

	mu      sync.Mutex
	headers []http.Header
}

func (f *fakeInventory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.headers = append(f.headers, r.Header.Clone())
	f.mu.Unlock()
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-r.Context().Done():
			return
		}
	}
	w.WriteHeader(f.status)
	if f.status == 200 {
		_ = json.NewEncoder(w).Encode(map[string]any{"item_id": r.PathValue("id"), "stock": f.stock})
	}
}

func (f *fakeInventory) lastHeader() http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.headers) == 0 {
		return nil
	}
	return f.headers[len(f.headers)-1]
}

type harness struct {
	h    http.Handler
	m    *metrics.HTTP
	inv  *fakeInventory
	logs *bytes.Buffer
}

func newHarness(t *testing.T, inv *fakeInventory, timeout time.Duration) *harness {
	t.Helper()
	srv := httptest.NewServer(inv)
	t.Cleanup(srv.Close)
	logs := &bytes.Buffer{}
	logger := logging.NewWithWriter(service, logs)
	m := metrics.NewHTTP(service)
	return &harness{h: newHandler(logger, newInventoryClient(srv.URL, testToken, timeout), m), m: m, inv: inv, logs: logs}
}

func (h *harness) do(method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	return rec
}

// logLine returns the single request line logged so far, decoded.
func (h *harness) logLine(t *testing.T) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(h.logs.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %s", len(lines), h.logs.String())
	}
	if err := logschema.Validate([]byte(lines[0]), logschema.Request); err != nil {
		t.Fatalf("%v\nline: %s", err, lines[0])
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &m)
	return m
}

const orderBody = `{"item_id":"SKU-1001","quantity":2}`

// T2.2-1
func TestCreateAndGetOrder(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)

	rec := h.do("POST", "/orders", orderBody, nil)
	if rec.Code != 201 {
		t.Fatalf("POST status %d: %s", rec.Code, rec.Body)
	}
	var o order
	if err := json.Unmarshal(rec.Body.Bytes(), &o); err != nil {
		t.Fatal(err)
	}
	if o.ID == "" || o.ItemID != "SKU-1001" || o.Quantity != 2 {
		t.Fatalf("created %+v", o)
	}

	rec = h.do("GET", "/orders/"+o.ID, "", nil)
	var got order
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || got != o {
		t.Fatalf("GET status %d, order %+v, want %+v", rec.Code, got, o)
	}
	if rec := h.do("GET", "/orders/ORD-999999", "", nil); rec.Code != 404 {
		t.Fatalf("missing order: status %d", rec.Code)
	}
	if rec := h.do("GET", "/healthz", "", nil); rec.Code != 200 {
		t.Fatalf("healthz: status %d", rec.Code)
	}

	low := newHarness(t, &fakeInventory{status: 200, stock: 1}, time.Second)
	if rec := low.do("POST", "/orders", orderBody, nil); rec.Code != 409 {
		t.Fatalf("insufficient stock: status %d", rec.Code)
	}
}

// T2.2-2
func TestRequestIDForwarded(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)
	id := requestid.New()
	h.do("POST", "/orders", orderBody, map[string]string{requestid.Header: id})

	if got := h.inv.lastHeader().Get(requestid.Header); got != id {
		t.Fatalf("inventory got X-Request-ID %q, want %q", got, id)
	}
	if got := h.logLine(t)["request_id"]; got != id {
		t.Fatalf("logged request_id %v, want %s", got, id)
	}
}

// T2.2-3
func TestRequestIDGenerated(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)
	h.do("POST", "/orders", orderBody, nil)

	logged, _ := h.logLine(t)["request_id"].(string) // logschema checks it is a UUID v4
	if fwd := h.inv.lastHeader().Get(requestid.Header); fwd != logged {
		t.Fatalf("forwarded %q, logged %q", fwd, logged)
	}
}

// T2.2-4
func TestInventory5xx(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 503}, time.Second)
	id := requestid.New()
	rec := h.do("POST", "/orders", orderBody, map[string]string{requestid.Header: id})
	if rec.Code != 502 {
		t.Fatalf("status %d, want 502", rec.Code)
	}
	l := h.logLine(t)
	if l["level"] != "ERROR" || l["item_id"] != "SKU-1001" || l["request_id"] != id || l["error"] == nil {
		t.Fatalf("log line %v", l)
	}
}

// T2.2-5
func TestInventoryTimeout(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10, delay: time.Second}, 50*time.Millisecond)
	rec := h.do("POST", "/orders", orderBody, nil)
	if rec.Code != 504 {
		t.Fatalf("status %d, want 504", rec.Code)
	}
	l := h.logLine(t)
	if l["level"] != "ERROR" || l["error"] != "context deadline exceeded" || l["item_id"] != "SKU-1001" {
		t.Fatalf("log line %v", l)
	}
}

// T2.2-5 (default)
func TestDefaultInventoryTimeout(t *testing.T) {
	t.Setenv("INVENTORY_TIMEOUT", "")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.inventoryTimeout != 2*time.Second || DefaultInventoryTimeout != 2*time.Second {
		t.Fatalf("default timeout %s", cfg.inventoryTimeout)
	}
	t.Setenv("INVENTORY_TIMEOUT", "750ms")
	if cfg, _ := loadConfig(); cfg.inventoryTimeout != 750*time.Millisecond {
		t.Fatalf("override ignored: %s", cfg.inventoryTimeout)
	}
}

// T2.2-7 (orders side)
func TestServiceTokenSent(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)
	h.do("POST", "/orders", orderBody, nil)
	if got := h.inv.lastHeader().Get(auth.Header); got != testToken {
		t.Fatalf("inventory got token %q, want %q", got, testToken)
	}
}

func TestStartupLine(t *testing.T) {
	var buf bytes.Buffer
	logStartup(logging.NewWithWriter(service, &buf), "8080")
	first, _, _ := strings.Cut(buf.String(), "\n")
	if err := logschema.Validate([]byte(first), logschema.Startup); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first, `"version":"`+version+`"`) {
		t.Fatalf("no version: %s", first)
	}
}

// T3.5-1
func TestDeployEvent(t *testing.T) {
	var buf bytes.Buffer
	logStartup(logging.NewWithWriter(service, &buf), "8080")
	var deploys []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		_ = json.Unmarshal([]byte(l), &m)
		if m["event"] == "deploy" {
			deploys = append(deploys, m)
		}
	}
	if len(deploys) != 1 {
		t.Fatalf("want 1 deploy line, got %d:\n%s", len(deploys), buf.String())
	}
	if deploys[0]["version"] != "1.4.2" || deploys[0]["level"] != "INFO" {
		t.Fatalf("deploy line %v, want version 1.4.2 at INFO", deploys[0])
	}
}
