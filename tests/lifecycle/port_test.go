//go:build lifecycle

package lifecycle

import (
	"strings"
	"testing"
	"time"
)

// T5.1-4
func TestGrafanaPortMoves(t *testing.T) {
	compose(t, "down")
	t.Cleanup(func() {
		compose(t, "down")
		compose(t, "up", "-d", "--wait")
	})
	must(t, []string{"GRAFANA_PORT=3301"}, "docker", "compose", "up", "-d", "--wait")

	if !eventually(30*time.Second, time.Second, func() bool {
		code, _, err := grafanaGet(3301, "/api/health")
		return err == nil && code == 200
	}) {
		t.Fatal(":3301/api/health is not 200")
	}
	if _, _, err := grafanaGet(3300, "/api/health"); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf(":3300 should refuse connections, got err=%v", err)
	}
}
