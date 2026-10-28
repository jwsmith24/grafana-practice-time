//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"
)

// T4.3-3 to T4.3-5
func TestModeA(t *testing.T) {
	d := labDuration(t)
	incidentSh(t, "clear")
	t.Cleanup(func() { incidentSh(t, "clear") })
	waitQuiet(t)

	start := time.Now()
	incidentSh(t, "A")
	expires := start.Add(d)

	// T4.3-3
	t.Run("error-rate panel shows it within 30s", func(t *testing.T) {
		var v float64
		for time.Since(start) <= 30*time.Second {
			if v = orders5xx(t); v > 0 {
				t.Logf("orders POST /orders 5xx = %.3f/s after %s", v, time.Since(start).Round(time.Second))
				return
			}
			time.Sleep(5 * time.Second)
		}
		t.Fatalf("orders POST /orders 5xx = %v 30s after the switch, want > 0", v)
	})

	// T4.3-4: the window starts at the switch, as an attendee's would.
	t.Run("errors isolate one item", func(t *testing.T) {
		secs := int(time.Since(start).Seconds()) + 1
		q := fmt.Sprintf(`sum by (item_id) (count_over_time({service="orders", level="ERROR"} | json [%ds]))`, secs)
		var res []lokiSample
		eventually(t, 15*time.Second, func() bool {
			res = lokiVector(t, q)
			return len(res) > 0
		})
		if len(res) != 1 || res[0].Metric["item_id"] != "SKU-1042" {
			t.Fatalf("%s: got %v, want one series with item_id SKU-1042", q, res)
		}
	})

	// T4.3-5: allow 2 s for the state poll and in-flight requests.
	t.Run("clears", func(t *testing.T) {
		from, to := expires.Add(2*time.Second), expires.Add(17*time.Second)
		time.Sleep(time.Until(to.Add(5 * time.Second)))
		if out := incidentSh(t, "status"); out != "none" {
			t.Fatalf("status after expiry: %q", out)
		}
		if n := countLines(lokiQueryRange(t, `{service=~"orders|inventory", level="ERROR"}`, from, to)); n != 0 {
			t.Fatalf("%d ERROR lines 2-17s after expiry", n)
		}
		if code := postOrder(t, "SKU-1042"); code != 201 {
			t.Fatalf("SKU-1042 order after expiry: status %d, want 201", code)
		}
	})
}
