//go:build integration

package integration

import (
	"testing"
	"time"
)

const redHerring = "config key ttl_seconds is deprecated, use cache_ttl"

// T2.4-2
func TestRedHerringRate(t *testing.T) {
	waitForUptime(t, "inventory", 65*time.Second)
	n := lokiInstant(t, `sum(count_over_time({service="inventory", level="WARN"} |= "ttl_seconds is deprecated" [1m]))`)
	if n < 28 || n > 32 {
		t.Fatalf("%v red herring lines in 1m, want 28-32", n)
	}
}

// T2.4-3 (lab AC6, baseline)
func TestRedHerringIsTopWarn(t *testing.T) {
	waitForUptime(t, "inventory", 30*time.Second)
	res := lokiVector(t, `topk(1, sum by (msg) (count_over_time({level="WARN"} | json [5m])))`)
	if len(res) != 1 {
		t.Fatalf("want 1 top WARN msg, got %v", res)
	}
	if got := res[0].Metric["msg"]; got != redHerring {
		t.Fatalf("top WARN msg = %q, want %q", got, redHerring)
	}
}

// T2.4-4
func TestHealthyBaseline(t *testing.T) {
	waitForUptime(t, "loadgen", 2*time.Minute)
	if res := lokiVector(t, `count_over_time({level="ERROR"}[2m])`); len(res) != 0 {
		t.Fatalf("ERROR lines at baseline: %v", res)
	}
	if n := countLines(lokiQuery(t, `{service="orders"} | json | status>=500`, 2*time.Minute)); n != 0 {
		t.Fatalf("%d orders 5xx lines at baseline", n)
	}
}

// T2.4-5
func TestRedHerringFilter(t *testing.T) {
	waitForLines(t, "inventory")
	if n := countLines(lokiQuery(t, `{service="inventory", level="WARN"} != "deprecated"`, 15*time.Minute)); n != 0 {
		t.Fatalf(`%d inventory WARN lines left after != "deprecated"`, n)
	}
}
