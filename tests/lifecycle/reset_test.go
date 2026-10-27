//go:build lifecycle

package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// T5.3-1 to T5.3-3
func TestReset(t *testing.T) {
	freshStack(t)
	// Dirty the lab: switch to B and wait for its deploy line.
	must(t, nil, "./incident.sh", "B")
	if !eventually(30*time.Second, 2*time.Second, func() bool {
		return len(lokiLines(t, `{service="orders"} |= "1.4.3"`, time.Now().Add(-5*time.Minute), time.Now())) > 0
	}) {
		t.Fatal("no 1.4.3 deploy line before the reset")
	}

	reset := time.Now()
	compose(t, "down", "-v")
	compose(t, "up", "-d", "--wait")

	// T5.3-1
	if out, code, _ := checkSh(t); code != 0 {
		t.Fatalf("check.sh after reset: exit %d\n%s", code, out)
	}

	// T5.3-2
	if out := strings.TrimSpace(must(t, nil, "./incident.sh", "status")); out != "none" {
		t.Errorf("status after reset: %q", out)
	}
	deploys := lokiLines(t, `{service="orders"} | json | event="deploy"`, reset.Add(-time.Hour), time.Now())
	if len(deploys) == 0 {
		t.Error("no deploy line after reset")
	}
	for _, l := range deploys {
		var d struct{ Version string }
		_ = json.Unmarshal([]byte(l), &d)
		if d.Version != "1.4.2" {
			t.Errorf("deploy line after reset: %s", l)
		}
	}

	// T5.3-3: nothing from before the reset in Loki, Prometheus or the files.
	if n := len(lokiLines(t, `{env="lab"}`, reset.Add(-time.Hour), reset)); n != 0 {
		t.Errorf("Loki has %d lines from before the reset", n)
	}
	var tsdb struct {
		Data struct {
			HeadStats struct {
				MinTime int64 `json:"minTime"`
			} `json:"headStats"`
		} `json:"data"`
	}
	if !eventually(30*time.Second, 2*time.Second, func() bool {
		proxyGet(t, "prometheus", "/api/v1/status/tsdb", nil, &tsdb)
		return tsdb.Data.HeadStats.MinTime > 0
	}) {
		t.Fatal("Prometheus has no samples after the reset")
	}
	if first := time.UnixMilli(tsdb.Data.HeadStats.MinTime); first.Before(reset) {
		t.Errorf("Prometheus's oldest sample %s is before the reset at %s", first, reset)
	}
	dir := t.TempDir()
	for _, svc := range []string{"orders", "inventory", "loadgen"} {
		dst := filepath.Join(dir, svc+".log")
		compose(t, "cp", "orders:/var/log/lab/"+svc+".log", dst)
		b, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		first, _, _ := strings.Cut(string(b), "\n")
		if !strings.Contains(first, `"event":"startup"`) {
			t.Errorf("%s.log starts with %s, want the startup line", svc, first)
		}
	}
}
