package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// deployVersions returns the version of every deploy line in buf.
func deployVersions(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if l == "" {
			continue
		}
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Fatalf("%v\nline: %s", err, l)
		}
		var m map[string]any
		_ = json.Unmarshal([]byte(l), &m)
		if m["event"] == "deploy" {
			if m["level"] != "INFO" {
				t.Fatalf("deploy line not INFO: %s", l)
			}
			out = append(out, m["version"].(string))
		}
	}
	return out
}

// T4.4-1
func TestReleaseSwitch(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10}, time.Second)
	var logs bytes.Buffer
	mode := ""
	rel := newRelease(logging.NewWithWriter(service, &logs), func() string { return mode })
	h.client.token = func() string { return tokenFor(rel.Version(), testToken) }

	rel.sync()
	if got := deployVersions(t, &logs); len(got) != 0 {
		t.Fatalf("deploy lines with no mode: %v", got)
	}
	h.do("POST", "/orders", orderBody, nil)
	if got := h.inv.lastHeader().Get("X-Service-Token"); got != testToken {
		t.Fatalf("token %q before the switch", got)
	}

	mode = "C"
	rel.sync()
	rel.sync()
	if got := deployVersions(t, &logs); len(got) != 1 || got[0] != "1.4.3" {
		t.Fatalf("deploy lines %v, want [1.4.3]", got)
	}
	h.do("POST", "/orders", orderBody, nil)
	if got := h.inv.lastHeader().Get("X-Service-Token"); got == testToken || got == "" {
		t.Fatalf("1.4.3 sent token %q, want a different, non-empty one", got)
	}

	mode = ""
	rel.sync()
	if got := deployVersions(t, &logs); len(got) != 2 || got[1] != "1.4.2" {
		t.Fatalf("deploy lines %v, want [1.4.3 1.4.2]", got)
	}
	h.do("POST", "/orders", orderBody, nil)
	if got := h.inv.lastHeader().Get("X-Service-Token"); got != testToken {
		t.Fatalf("token %q after the rollback", got)
	}
}

// T4.4-3
func TestRejectedToken(t *testing.T) {
	h := newHarness(t, &fakeInventory{status: 200, stock: 10, token: testToken}, time.Second)
	h.client.token = func() string { return tokenFor(nextVersion, testToken) }
	rec := h.do("POST", "/orders", orderBody, nil)
	if rec.Code != 502 {
		t.Fatalf("status %d, want 502", rec.Code)
	}
	l := h.logLine(t)
	if l["level"] != "ERROR" || l["error"] != errUnauthorized.Error() || l["item_id"] != "SKU-1001" {
		t.Fatalf("log line %v", l)
	}
	if err := testutil.GatherAndCompare(h.m.Registry, strings.NewReader(upstreamWant("unauthorized")), "inventory_upstream_errors_total"); err != nil {
		t.Fatal(err)
	}
}
