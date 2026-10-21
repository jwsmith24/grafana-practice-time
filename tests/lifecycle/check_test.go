//go:build lifecycle

package lifecycle

import (
	"net"
	"strings"
	"testing"
	"time"
)

const facilitatorLine = "message the facilitator"

var passLines = []string{
	"PASS  Docker is running",
	"PASS  all 7 containers are running",
	"PASS  Grafana answers on port 3300",
	"PASS  Loki is ready",
	"PASS  Prometheus is ready",
	"PASS  Loki has lines from orders",
	"PASS  Loki has lines from inventory",
	"PASS  Loki has lines from loadgen",
	"PASS  Prometheus scrapes orders",
	"PASS  Prometheus scrapes inventory",
}

// T5.2-1, T5.2-5
func TestCheckHealthy(t *testing.T) {
	freshStack(t)
	out, code, took := checkSh(t)
	if code != 0 {
		t.Fatalf("exit %d on a healthy stack:\n%s", code, out)
	}
	for _, l := range passLines {
		if !strings.Contains(out, l) {
			t.Errorf("no %q line:\n%s", l, out)
		}
	}
	if took >= 30*time.Second {
		t.Errorf("took %s, want < 30s", took)
	}
	t.Logf("passed in %s", took.Round(10*time.Millisecond))
}

// T5.2-2: a stopped service's last lines stay in the 1m window for up to a
// minute, so each case keeps running check.sh until the failure shows.
func TestCheckFailures(t *testing.T) {
	freshStack(t)
	for _, c := range []struct {
		stop string
		want []string
	}{
		{"loki", []string{"FAIL  Loki is ready", "Loki isn't ready"}},
		{"prometheus", []string{"FAIL  Prometheus is ready", "Prometheus isn't ready"}},
		{"loadgen", []string{"FAIL  Loki has lines from loadgen", "no lines from loadgen"}},
		{"inventory", []string{
			"FAIL  Prometheus scrapes inventory", "Prometheus can't scrape inventory",
			"FAIL  Loki has lines from inventory", "no lines from inventory",
		}},
	} {
		t.Run("stop "+c.stop, func(t *testing.T) {
			compose(t, "stop", c.stop)
			defer compose(t, "up", "-d", "--wait")
			var out string
			var code int
			ok := eventually(3*time.Minute, 5*time.Second, func() bool {
				out, code, _ = checkSh(t)
				if code == 0 {
					return false
				}
				for _, w := range c.want {
					if !strings.Contains(out, w) {
						return false
					}
				}
				return true
			})
			if !ok {
				t.Fatalf("exit %d, want non-zero with %q:\n%s", code, c.want, out)
			}
			if !strings.Contains(out, "Not running: "+c.stop) {
				t.Errorf("does not name the stopped container:\n%s", out)
			}
			// T5.6-6: the failure output ends with the facilitator line.
			lines := strings.Split(strings.TrimSpace(out), "\n")
			if !strings.Contains(lines[len(lines)-1], facilitatorLine) {
				t.Errorf("last line %q, want the %q line", lines[len(lines)-1], facilitatorLine)
			}
		})
	}
}

// T5.2-3
func TestCheckPortConflict(t *testing.T) {
	compose(t, "down")
	l, err := net.Listen("tcp", ":3300")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		l.Close()
		compose(t, "down")
		compose(t, "up", "-d", "--wait")
	})
	if _, code := run(t, nil, "docker", "compose", "up", "-d"); code == 0 {
		t.Fatal("docker compose up succeeded with port 3300 taken")
	}
	out, code, _ := checkSh(t)
	if code == 0 {
		t.Fatalf("exit 0 with port 3300 taken:\n%s", out)
	}
	for _, w := range []string{"Another program is using port 3300", "GRAFANA_PORT"} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q:\n%s", w, out)
		}
	}
}
