//go:build integration

package integration

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// T6.4-3: every demo query returns results while mode B runs, each run in
// the script's order with its placeholders filled from the step before.
func TestDemoScript(t *testing.T) {
	blocks := regexp.MustCompile("(?ms)^```(logql|promql)\n(.*?)^```$").FindAllStringSubmatch(readMaterial(t, "demo-script.md"), -1)
	if len(blocks) == 0 {
		t.Fatal("no query blocks in the demo script")
	}
	start := triggerMode(t, "B")
	values := map[string]string{}
	for i, b := range blocks {
		lang, q := b[1], fill(t, strings.TrimSpace(b[2]), values)
		var ok bool
		switch {
		case lang == "promql":
			ok = eventually(t, 30*time.Second, func() bool { return promHasData(t, q) })
		case !strings.HasPrefix(q, "{"):
			ok = eventually(t, 30*time.Second, func() bool { return len(lokiVector(t, q)) > 0 })
		default:
			var lines []logLine
			ok = eventually(t, 30*time.Second, func() bool {
				lines = linesSince(t, q, start)
				return len(lines) > 0
			})
			// The next step's request_id comes from the newest deadline error.
			for _, l := range lines {
				if strings.Contains(l.str("error"), "context deadline exceeded") {
					values["request_id"] = l.str("request_id")
				}
			}
		}
		if !ok {
			t.Fatalf("step %d: no results for %s", i+1, q)
		}
		t.Logf("step %d: %s", i+1, q)
	}
}

// promHasData runs a PromQL query over the last 5 minutes through Grafana and
// reports whether any series has a value.
func promHasData(t *testing.T, expr string) bool {
	t.Helper()
	p := dashPanel{Title: "demo", Targets: []dashTarget{
		{RefID: "A", Datasource: map[string]string{"type": "prometheus", "uid": "prometheus"}, Expr: expr},
	}}
	for _, s := range panelQuery(t, p, 5*time.Minute) {
		if _, ok := lastFinite(s.Values); ok {
			return true
		}
	}
	return false
}
