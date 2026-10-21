//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// labDuration is the INCIDENT_DURATION the stack was started with.
func labDuration(t *testing.T) time.Duration {
	t.Helper()
	v := envOr("INCIDENT_DURATION", "3m")
	d, err := time.ParseDuration(v)
	if err != nil {
		t.Fatalf("INCIDENT_DURATION %q: %v", v, err)
	}
	return d
}

// incidentSh runs ./incident.sh with arg and returns its trimmed output.
func incidentSh(t *testing.T, arg string) string {
	t.Helper()
	cmd := exec.Command("../../incident.sh", arg)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("incident.sh %s: %v", arg, err)
	}
	return strings.TrimSpace(string(out))
}

// orders5xx returns the error-rate panel's latest value for
// {service="orders", endpoint="POST /orders"}, or 0 when there is no series.
func orders5xx(t *testing.T) float64 {
	t.Helper()
	for _, s := range panelQuery(t, dashboardPanels(t)["Error rate (5xx) by endpoint"], 5*time.Minute) {
		if s.Labels["service"] == "orders" && s.Labels["endpoint"] == "POST /orders" {
			v, _ := lastFinite(s.Values)
			return v
		}
	}
	return 0
}

// waitQuiet waits until earlier tests' 5xx have left the error-rate panel's
// 1m window, so a test that triggers a mode sees only its own errors.
func waitQuiet(t *testing.T) {
	t.Helper()
	if !eventually(t, 90*time.Second, func() bool { return orders5xx(t) == 0 }) {
		t.Fatalf("orders 5xx rate still %v after 90s", orders5xx(t))
	}
}

// postOrder places an order for item and returns the status code.
func postOrder(t *testing.T, item string) int {
	t.Helper()
	body := fmt.Sprintf(`{"item_id":%q,"quantity":1}`, item)
	resp, err := client.Post(ordersURL()+"/orders", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// T4.1-6
func TestSwitchEndToEnd(t *testing.T) {
	d := labDuration(t)
	incidentSh(t, "clear")
	t.Cleanup(func() { incidentSh(t, "clear") })

	start := time.Now()
	incidentSh(t, "A")
	if out := incidentSh(t, "status"); !strings.HasPrefix(out, "active: A") {
		t.Fatalf("status right after trigger: %q", out)
	}
	ok := eventually(t, d+10*time.Second, func() bool {
		return incidentSh(t, "status") == "none"
	})
	elapsed := time.Since(start)
	if !ok {
		t.Fatalf("still active %s after trigger (duration %s)", elapsed.Round(time.Second), d)
	}
	if elapsed < d || elapsed > d+5*time.Second {
		t.Fatalf("cleared after %s, want between %s and %s", elapsed.Round(100*time.Millisecond), d, d+5*time.Second)
	}
}

// T4.1-8
func TestSwitchLeavesNoTrace(t *testing.T) {
	incidentSh(t, "B")
	incidentSh(t, "clear")
	// Give Alloy time to ship anything that was written.
	time.Sleep(5 * time.Second)
	if n := countLines(lokiQuery(t, `{env="lab"} |~ "(?i)incident"`, 15*time.Minute)); n != 0 {
		t.Fatalf(`%d log lines match (?i)incident`, n)
	}
}
