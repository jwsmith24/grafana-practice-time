package static

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const dashboardPath = "../../grafana/provisioning/dashboards/lab-overview.json"

type panel struct {
	ID         int            `json:"id"`
	Title      string         `json:"title"`
	Type       string         `json:"type"`
	Datasource *datasourceRef `json:"datasource"`
	Targets    []target       `json:"targets"`
}

type datasourceRef struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

type target struct {
	RefID      string         `json:"refId"`
	Datasource *datasourceRef `json:"datasource"`
	Expr       string         `json:"expr"`
}

type dashboard struct {
	UID    string  `json:"uid"`
	Title  string  `json:"title"`
	Panels []panel `json:"panels"`
}

func loadDashboard(t *testing.T) (dashboard, any) {
	t.Helper()
	b, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatal(err)
	}
	var d dashboard
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	var raw any
	_ = json.Unmarshal(b, &raw)
	return d, raw
}

func panelByTitle(t *testing.T, d dashboard, title string) panel {
	t.Helper()
	for _, p := range d.Panels {
		if p.Title == title {
			return p
		}
	}
	t.Fatalf("no panel titled %q", title)
	return panel{}
}

// T3.2-2
func TestPrometheusPanels(t *testing.T) {
	d, _ := loadDashboard(t)
	if d.UID != "lab-overview" {
		t.Fatalf("uid %q, want lab-overview", d.UID)
	}
	want := map[string]string{
		"Request rate by service":      `sum by (service) (rate(http_requests_total[1m]))`,
		"Error rate (5xx) by endpoint": `sum by (service, endpoint) (rate(http_requests_total{status=~"5.."}[1m]))`,
		"p95 latency by service":       `histogram_quantile(0.95, sum by (service, le) (rate(http_request_duration_seconds_bucket[1m])))`,
	}
	for title, expr := range want {
		p := panelByTitle(t, d, title)
		if p.Datasource == nil || p.Datasource.UID != "prometheus" {
			t.Errorf("%s: datasource %+v, want uid prometheus", title, p.Datasource)
		}
		if len(p.Targets) != 1 || p.Targets[0].Expr != expr {
			t.Errorf("%s: targets %+v, want expr %s", title, p.Targets, expr)
		}
	}
}

var knownDatasources = map[string]string{
	"loki":          "loki",
	"prometheus":    "prometheus",
	"-- Grafana --": "grafana",
}

// T3.2-3
func TestDatasourcesByUID(t *testing.T) {
	_, raw := loadDashboard(t)
	n := 0
	walk(raw, "$", func(path string, v any) {
		n++
		m, ok := v.(map[string]any)
		if !ok {
			t.Errorf("%s: datasource is %T (%v), want {type, uid}", path, v, v)
			return
		}
		uid, _ := m["uid"].(string)
		typ, _ := m["type"].(string)
		if wantType, ok := knownDatasources[uid]; !ok || typ != wantType {
			t.Errorf("%s: datasource %v is not a known {type, uid}", path, m)
		}
	})
	if n == 0 {
		t.Fatal("no datasource refs found")
	}
}

// walk calls f for every value stored under a "datasource" key.
func walk(v any, path string, f func(string, any)) {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			p := path + "." + k
			if k == "datasource" {
				f(p, c)
			}
			walk(c, p, f)
		}
	case []any:
		for i, c := range x {
			walk(c, fmt.Sprintf("%s[%d]", path, i), f)
		}
	}
}

// T3.3-1, T3.3-2
func TestLokiPanels(t *testing.T) {
	d, _ := loadDashboard(t)
	volume := panelByTitle(t, d, "Error log volume by service")
	recent := panelByTitle(t, d, "Recent errors")
	for _, p := range []panel{volume, recent} {
		if p.Datasource == nil || p.Datasource.UID != "loki" {
			t.Errorf("%s: datasource %+v, want uid loki", p.Title, p.Datasource)
		}
		if len(p.Targets) != 1 || !strings.Contains(p.Targets[0].Expr, `level="ERROR"`) {
			t.Errorf("%s: targets %+v, want a level=\"ERROR\" query", p.Title, p.Targets)
		}
	}
	if recent.Type != "logs" {
		t.Errorf("Recent errors is a %q panel, want logs", recent.Type)
	}
	if want := `sum by (service) (count_over_time({env="lab", level="ERROR"}[1m]))`; len(volume.Targets) == 1 && volume.Targets[0].Expr != want {
		t.Errorf("volume expr %q, want %q", volume.Targets[0].Expr, want)
	}
}
