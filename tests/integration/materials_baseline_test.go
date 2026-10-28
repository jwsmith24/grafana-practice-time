//go:build integration

package integration

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// This file sorts before materials_test.go, whose mode B chain leaves WARN
// lines other than the red herring.

// T6.2-3: the warm-ups and the stretch question, answered at baseline with
// cheat-sheet queries only.
func TestWorksheetBaseline(t *testing.T) {
	waitForUptime(t, "inventory", 65*time.Second)
	selector := template(t, `{service="<service>", level="<level>"}`)

	t.Run("1 labels and fields", func(t *testing.T) {
		lines := eventuallyLines(t, fill(t, selector, map[string]string{"service": "orders", "level": "INFO"}), time.Now().Add(-time.Minute))
		for _, l := range lines {
			keys := make([]string, 0, len(l.labels))
			for k := range l.labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			if strings.Join(keys, ",") != "env,level,service" {
				t.Fatalf("stream labels %v, want env, level, service", l.labels)
			}
		}
		var request logLine
		for _, l := range lines {
			if l.str("msg") == "request" {
				request = l
			}
		}
		for _, f := range []string{"request_id", "endpoint", "status", "duration_ms", "order_id"} {
			if _, ok := request.fields[f]; !ok {
				t.Errorf("orders request line has no %s: %v", f, request.fields)
			}
		}
	})

	t.Run("2 follow one request", func(t *testing.T) {
		id := recentRequestID(t)
		q := fill(t, template(t, "request_id="), map[string]string{"request_id": id})
		lines := eventuallyLines(t, q, time.Now().Add(-2*time.Minute))
		var services []string
		for _, l := range lines {
			services = append(services, l.labels["service"])
		}
		sort.Strings(services)
		if strings.Join(services, ",") != "inventory,orders" {
			t.Fatalf("request %s: lines from %v, want one each from inventory and orders", id, services)
		}
	})

	t.Run("6 the noisy warning", func(t *testing.T) {
		top := lokiVector(t, fill(t, template(t, "topk("), map[string]string{"level": "WARN", "range": "1m"}))
		sort.Slice(top, func(i, j int) bool { return top[i].float(t) > top[j].float(t) })
		if len(top) == 0 || top[0].Metric["msg"] != redHerring {
			t.Fatalf("most frequent WARN messages: %v, want %q first", top, redHerring)
		}
		if n := top[0].float(t); n < 28 || n > 32 {
			t.Errorf("%v a minute, want about 30", n)
		}
		lines := eventuallyLines(t, fill(t, selector, map[string]string{"service": "inventory", "level": "WARN"}), time.Now().Add(-time.Minute))
		if lines[0].str("msg") != redHerring {
			t.Errorf("inventory WARN line %q, want %q", lines[0].str("msg"), redHerring)
		}
		if n := countLines(lokiQuery(t, template(t, `!= "deprecated"`), 2*time.Minute)); n != 0 {
			t.Fatalf("%d WARN lines left after hiding the red herring", n)
		}
	})
}
