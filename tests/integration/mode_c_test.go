//go:build integration

package integration

import (
	"bytes"
	"strconv"
	"testing"
	"time"
)

// lokiEntry is one log line with its timestamp.
type lokiEntry struct {
	ts      time.Time
	service string
	line    string
}

// oldest returns the earliest entry in streams.
func oldest(t *testing.T, streams []lokiStream) (lokiEntry, bool) {
	t.Helper()
	var first lokiEntry
	found := false
	for _, s := range streams {
		for _, v := range s.Values {
			ns, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				t.Fatalf("timestamp %q: %v", v[0], err)
			}
			ts := time.Unix(0, ns)
			if !found || ts.Before(first.ts) {
				first, found = lokiEntry{ts: ts, service: s.Stream["service"], line: v[1]}, true
			}
		}
	}
	return first, found
}

// deployAt returns when orders logged a deploy of version since start.
func deployAt(t *testing.T, version string, start time.Time) (time.Time, bool) {
	t.Helper()
	q := `{service="orders"} | json | event="deploy" | version="` + version + `"`
	e, ok := oldest(t, lokiQueryRange(t, q, start, time.Now()))
	return e.ts, ok
}

// T4.4-4 to T4.4-6
func TestModeC(t *testing.T) {
	d := labDuration(t)
	incidentSh(t, "clear")
	t.Cleanup(func() { incidentSh(t, "clear") })
	waitQuiet(t)

	start := time.Now()
	incidentSh(t, "C")
	expires := start.Add(d)

	// T4.4-4
	t.Run("deploy annotation and error rate within 30s", func(t *testing.T) {
		anno := deployAnnotation(t)
		var seen bool
		var v float64
		for time.Since(start) <= 30*time.Second {
			seen = seen || bytes.Contains(panelRaw(t, anno, 5*time.Minute), []byte(`\"version\":\"1.4.3\"`))
			if v = orders5xx(t); seen && v > 0 {
				t.Logf("annotation and 5xx = %.3f/s after %s", v, time.Since(start).Round(time.Second))
				return
			}
			time.Sleep(5 * time.Second)
		}
		t.Fatalf("30s after the switch: 1.4.3 annotation %v, orders 5xx %v", seen, v)
	})

	// T4.4-5
	t.Run("first error after the deploy leads to inventory", func(t *testing.T) {
		deployed, ok := deployAt(t, "1.4.3", start.Add(-time.Second))
		if !ok {
			t.Fatal("no 1.4.3 deploy line")
		}
		first, ok := oldest(t, lokiQueryRange(t, `{level="ERROR"}`, deployed, time.Now()))
		if !ok {
			t.Fatal("no ERROR line after the deploy")
		}
		if first.service != "orders" || !bytes.Contains([]byte(first.line), []byte("inventory rejected credentials")) {
			t.Fatalf("first ERROR after the deploy is %s: %s", first.service, first.line)
		}
		warn, ok := oldest(t, lokiQueryRange(t, `{service="inventory", level="WARN"} |= "auth token rejected"`, deployed, time.Now()))
		if !ok {
			t.Fatal("no auth token rejected WARN after the deploy")
		}
		if gap := warn.ts.Sub(deployed); gap < 0 || gap > 5*time.Second {
			t.Fatalf("auth token rejected starts %s after the deploy, want 0-5s", gap)
		}
	})

	// T4.4-6
	t.Run("clears with a rollback", func(t *testing.T) {
		time.Sleep(time.Until(expires.Add(2 * time.Second)))
		if out := incidentSh(t, "status"); out != "none" {
			t.Fatalf("status after expiry: %q", out)
		}
		if !eventually(t, 10*time.Second, func() bool { return postOrder(t, "SKU-1001") == 201 }) {
			t.Fatal("orders not returning 201 10s after expiry")
		}
		bad, _ := deployAt(t, "1.4.3", start.Add(-time.Second))
		var back time.Time
		if !eventually(t, 10*time.Second, func() bool {
			var ok bool
			back, ok = deployAt(t, "1.4.2", bad)
			return ok
		}) {
			t.Fatal("no 1.4.2 deploy line after the 1.4.3 one")
		}
		t.Logf("1.4.2 rollback %s after 1.4.3", back.Sub(bad).Round(time.Millisecond))
	})
}
