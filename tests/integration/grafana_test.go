//go:build integration

package integration

import (
	"regexp"
	"testing"
)

var orgRoleRe = regexp.MustCompile(`"isSignedIn":false,[^}]*"orgRole":"([A-Za-z]+)"`)

// T1.1-1
func TestGrafanaHealth(t *testing.T) {
	code, b := do(t, "GET", "/api/health", nil, false)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
	var h struct {
		Database string `json:"database"`
	}
	decode(t, b, &h)
	if h.Database != "ok" {
		t.Fatalf("database = %q, want ok", h.Database)
	}
}

// T1.1-2
func TestLokiDatasourceProvisioned(t *testing.T) {
	code, b := do(t, "GET", "/api/datasources/uid/loki", nil, true)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
	var ds struct {
		Type      string `json:"type"`
		IsDefault bool   `json:"isDefault"`
	}
	decode(t, b, &ds)
	if ds.Type != "loki" || !ds.IsDefault {
		t.Fatalf("got type=%q isDefault=%v, want type=loki isDefault=true", ds.Type, ds.IsDefault)
	}
}

// T1.1-3
// Anonymous sessions have no user record, so /api/user is 401 by design. The
// role is read from the boot data Grafana's frontend uses, and Explore access
// from the anonymous session's permissions.
func TestAnonymousEditor(t *testing.T) {
	code, b := do(t, "GET", "/", nil, false)
	if code != 200 {
		t.Fatalf("GET / status %d", code)
	}
	m := orgRoleRe.FindSubmatch(b)
	if m == nil {
		t.Fatal("boot data has no orgRole")
	}
	if string(m[1]) != "Editor" {
		t.Fatalf("anonymous orgRole = %q, want Editor", m[1])
	}

	code, b = do(t, "GET", "/api/access-control/user/permissions", nil, false)
	if code != 200 {
		t.Fatalf("GET permissions status %d: %s", code, b)
	}
	var perms map[string]any
	decode(t, b, &perms)
	if _, ok := perms["datasources:explore"]; !ok {
		t.Fatal("anonymous user lacks datasources:explore")
	}

	q := map[string]any{
		"from": "now-5m",
		"to":   "now",
		"queries": []map[string]any{{
			"refId":      "A",
			"datasource": map[string]string{"type": "loki", "uid": "loki"},
			"expr":       `{env="lab"}`,
			"queryType":  "range",
			"maxLines":   10,
		}},
	}
	code, b = do(t, "POST", "/api/ds/query", q, false)
	if code != 200 {
		t.Fatalf("POST /api/ds/query status %d: %s", code, b)
	}
}

// T1.1-4
func TestAdminLogin(t *testing.T) {
	code, b := do(t, "GET", "/api/admin/settings", nil, true)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
}

// T1.1-5
func TestLokiReady(t *testing.T) {
	code, b := do(t, "GET", "/api/datasources/proxy/uid/loki/ready", nil, false)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
}
