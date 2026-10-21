package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// T3.1-1 (orders)
func TestRequestMetrics(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)
	h.do("POST", "/orders", orderBody, nil)
	h.do("GET", "/orders/ORD-999999", "", nil)
	h.do("GET", "/healthz", "", nil)

	want := `
# HELP http_requests_total HTTP requests served, by route template and status code.
# TYPE http_requests_total counter
http_requests_total{endpoint="GET /orders/{id}",service="orders",status="404"} 1
http_requests_total{endpoint="POST /orders",service="orders",status="201"} 1
`
	if err := testutil.GatherAndCompare(h.m.Registry, strings.NewReader(want), "http_requests_total"); err != nil {
		t.Fatal(err)
	}
	if n := testutil.CollectAndCount(h.m.Registry, "http_request_duration_seconds"); n != 2 {
		t.Fatalf("%d duration series, want 2", n)
	}
}

// T3.1-3
func TestUpstreamErrorMetrics(t *testing.T) {
	cases := []struct {
		reason string
		inv    *fakeInventory
		want   int
	}{
		{"timeout", &fakeInventory{status: 200, delay: time.Second}, 504},
		{"server_error", &fakeInventory{status: 500}, 502},
		{"unauthorized", &fakeInventory{status: 401}, 502},
	}
	for _, c := range cases {
		t.Run(c.reason, func(t *testing.T) {
			h := newHarness(t, c.inv, 50*time.Millisecond)
			if rec := h.do("POST", "/orders", orderBody, nil); rec.Code != c.want {
				t.Fatalf("status %d, want %d", rec.Code, c.want)
			}
			if err := testutil.GatherAndCompare(h.m.Registry, strings.NewReader(upstreamWant(c.reason)), "inventory_upstream_errors_total"); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("unknown item is not an upstream error", func(t *testing.T) {
		h := newHarness(t, &fakeInventory{status: 404}, time.Second)
		if rec := h.do("POST", "/orders", orderBody, nil); rec.Code != 404 {
			t.Fatalf("status %d, want 404", rec.Code)
		}
		if err := testutil.GatherAndCompare(h.m.Registry, strings.NewReader(upstreamWant("")), "inventory_upstream_errors_total"); err != nil {
			t.Fatal(err)
		}
	})
}

// upstreamWant is the expected exposition with reason at 1 and the rest at 0.
func upstreamWant(reason string) string {
	var b strings.Builder
	b.WriteString("# HELP inventory_upstream_errors_total Failed calls from orders to inventory, by reason.\n# TYPE inventory_upstream_errors_total counter\n")
	for _, r := range upstreamReasons {
		n := 0
		if r == reason {
			n = 1
		}
		fmt.Fprintf(&b, "inventory_upstream_errors_total{reason=%q} %d\n", r, n)
	}
	return b.String()
}
