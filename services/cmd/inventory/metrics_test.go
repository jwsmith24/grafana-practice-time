package main

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// T3.1-1 (inventory)
func TestRequestMetrics(t *testing.T) {
	m := metrics.NewHTTP(service)
	h := newHandler(slog.New(slog.DiscardHandler), testToken, m, store{})
	serve(t, h, "/inventory/SKU-1001", nil)
	serve(t, h, "/inventory/nope", nil)
	serve(t, h, "/healthz", nil)

	want := `
# HELP http_requests_total HTTP requests served, by route template and status code.
# TYPE http_requests_total counter
http_requests_total{endpoint="GET /inventory/{item_id}",service="inventory",status="200"} 1
http_requests_total{endpoint="GET /inventory/{item_id}",service="inventory",status="404"} 1
`
	if err := testutil.GatherAndCompare(m.Registry, strings.NewReader(want), "http_requests_total"); err != nil {
		t.Fatal(err)
	}
	if n := testutil.CollectAndCount(m.Registry, "http_request_duration_seconds"); n != 1 {
		t.Fatalf("%d duration series, want 1", n)
	}
	if rec := serve(t, h, "/metrics", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "http_request_duration_seconds_bucket") {
		t.Fatalf("/metrics status %d", rec.Code)
	}
}

// T3.1-2
func TestEndpointIsRouteTemplate(t *testing.T) {
	m := metrics.NewHTTP(service)
	h := newHandler(slog.New(slog.DiscardHandler), testToken, m, store{})
	for _, id := range []string{"SKU-1001", "SKU-1002", "SKU-1003"} {
		serve(t, h, "/inventory/"+id, nil)
	}
	if n := testutil.CollectAndCount(m.Registry, "http_requests_total"); n != 1 {
		t.Fatalf("%d request series for 3 item IDs, want 1", n)
	}
	if n := testutil.CollectAndCount(m.Registry, "http_request_duration_seconds"); n != 1 {
		t.Fatalf("%d duration series for 3 item IDs, want 1", n)
	}
}
