//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type requestLine struct {
	RequestID  string  `json:"request_id"`
	Level      string  `json:"level"`
	Status     int     `json:"status"`
	DurationMS float64 `json:"duration_ms"`
}

func requestLines(t *testing.T, streams []lokiStream) []requestLine {
	t.Helper()
	var out []requestLine
	for _, s := range streams {
		for _, v := range s.Values {
			var l requestLine
			if err := json.Unmarshal([]byte(v[1]), &l); err != nil {
				t.Fatalf("line %q: %v", v[1], err)
			}
			out = append(out, l)
		}
	}
	return out
}

// T4.2-2 to T4.2-5
func TestModeA(t *testing.T) {
	d := labDuration(t)
	incidentSh(t, "clear")
	t.Cleanup(func() { incidentSh(t, "clear") })

	start := time.Now()
	incidentSh(t, "A")
	expires := start.Add(d)
	since := func() time.Duration { return time.Since(start) + 5*time.Second }

	// T4.2-2
	var timeouts []requestLine
	t.Run("orders logs deadline errors", func(t *testing.T) {
		const q = `{service="orders", level="ERROR"} |= "context deadline exceeded"`
		if !eventually(t, 20*time.Second, func() bool {
			timeouts = requestLines(t, lokiQuery(t, q, since()))
			return len(timeouts) > 0
		}) {
			t.Fatalf("no lines for %s", q)
		}
	})

	// T4.2-4
	t.Run("request id leads to a slow inventory line", func(t *testing.T) {
		if len(timeouts) == 0 {
			t.Skip("needs a deadline error from the previous step")
		}
		id := timeouts[0].RequestID
		if id == "" {
			t.Fatalf("deadline error has no request_id: %+v", timeouts[0])
		}
		q := fmt.Sprintf(`{service="inventory"} | json | request_id=%q`, id)
		var got []requestLine
		if !eventually(t, 15*time.Second, func() bool {
			got = requestLines(t, lokiQuery(t, q, since()))
			return len(got) > 0
		}) {
			t.Fatalf("no inventory line for request_id %s", id)
		}
		l := got[0]
		if l.Level != "INFO" || l.Status != 200 || l.DurationMS <= 2000 {
			t.Fatalf("inventory line for %s: %+v, want INFO, 200, duration_ms > 2000", id, l)
		}
	})

	// T4.2-3
	t.Run("p95 panel shows it within 30s", func(t *testing.T) {
		panel := dashboardPanels(t)["p95 latency by service"]
		var p95 float64
		for time.Since(start) <= 30*time.Second {
			for _, s := range panelQuery(t, panel, 5*time.Minute) {
				if s.Labels["service"] == "inventory" {
					p95, _ = lastFinite(s.Values)
				}
			}
			if p95 > 2 {
				t.Logf("p95(inventory) = %.2fs after %s", p95, time.Since(start).Round(time.Second))
				return
			}
			time.Sleep(5 * time.Second)
		}
		t.Fatalf("p95(inventory) = %.2fs 30s after the switch, want > 2s", p95)
	})

	// T4.2-5: lookups that started before expiry can take up to 4 s more
	// (plus the 1 s state poll), so judge the lines from expiry+6s to +10s.
	t.Run("clears", func(t *testing.T) {
		from, to := expires.Add(6*time.Second), expires.Add(10*time.Second)
		time.Sleep(time.Until(to.Add(5 * time.Second))) // let Alloy ship them
		if out := incidentSh(t, "status"); out != "none" {
			t.Fatalf("status after expiry: %q", out)
		}
		lines := requestLines(t, lokiQueryRange(t, `{service="inventory"} |= "duration_ms"`, from, to))
		if len(lines) == 0 {
			t.Fatal("no inventory request lines 6-10s after expiry")
		}
		for _, l := range lines {
			if l.DurationMS >= 2000 {
				t.Fatalf("inventory line 6-10s after expiry took %.0fms: %+v", l.DurationMS, l)
			}
		}
	})
}
