//go:build integration

package integration

import "testing"

// T4.4-7 (lab AC6): this file sorts last, so the window covers the whole
// run: baseline, then A, B and C, then cleared. The nightly run uses the 3m
// default for each.
func TestRedHerringSession(t *testing.T) {
	q := `topk(1, sum by (msg) (count_over_time({level="WARN"} | json [1h])))`
	res := lokiVector(t, q)
	if len(res) != 1 {
		t.Fatalf("want 1 top WARN msg, got %v", res)
	}
	if got := res[0].Metric["msg"]; got != redHerring {
		t.Fatalf("top WARN msg over the session = %q, want %q (all: %v)", got, redHerring, lokiVector(t, `sum by (msg) (count_over_time({level="WARN"} | json [1h]))`))
	}
}
