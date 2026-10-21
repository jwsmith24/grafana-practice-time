//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

// T3.2-1
func TestHomeDashboard(t *testing.T) {
	code, b := do(t, "GET", "/api/dashboards/home", nil, false)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
	var r struct {
		Dashboard struct {
			UID string `json:"uid"`
		} `json:"dashboard"`
	}
	decode(t, b, &r)
	if r.Dashboard.UID != "lab-overview" {
		t.Fatalf("home uid %q, want lab-overview", r.Dashboard.UID)
	}
}

type dashPanel struct {
	Title   string `json:"title"`
	Targets []struct {
		RefID      string            `json:"refId"`
		Datasource map[string]string `json:"datasource"`
		Expr       string            `json:"expr"`
	} `json:"targets"`
}

func dashboardPanels(t *testing.T) map[string]dashPanel {
	t.Helper()
	b, err := os.ReadFile("../../grafana/provisioning/dashboards/lab-overview.json")
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Panels []dashPanel `json:"panels"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	m := map[string]dashPanel{}
	for _, p := range d.Panels {
		m[p.Title] = p
	}
	return m
}

type frameSeries struct {
	Labels map[string]string
	Values []*float64
}

// panelQuery runs a panel's first target through Grafana's /api/ds/query
// over the last window and returns one series per frame.
func panelQuery(t *testing.T, p dashPanel, window time.Duration) []frameSeries {
	t.Helper()
	if len(p.Targets) == 0 {
		t.Fatalf("panel %q has no targets", p.Title)
	}
	tg := p.Targets[0]
	q := map[string]any{
		"from": fmt.Sprintf("now-%dm", int(window.Minutes())),
		"to":   "now",
		"queries": []map[string]any{{
			"refId":      tg.RefID,
			"datasource": tg.Datasource,
			"expr":       tg.Expr,
			"range":      true,
		}},
	}
	code, b := do(t, "POST", "/api/ds/query", q, false)
	if code != 200 {
		t.Fatalf("%s: status %d: %s", p.Title, code, b)
	}
	var r struct {
		Results map[string]struct {
			Error  string `json:"error"`
			Frames []struct {
				Schema struct {
					Fields []struct {
						Labels map[string]string `json:"labels"`
					} `json:"fields"`
				} `json:"schema"`
				Data struct {
					Values [][]*float64 `json:"values"`
				} `json:"data"`
			} `json:"frames"`
		} `json:"results"`
	}
	decode(t, b, &r)
	res := r.Results[tg.RefID]
	if res.Error != "" {
		t.Fatalf("%s: %s", p.Title, res.Error)
	}
	var out []frameSeries
	for _, f := range res.Frames {
		if len(f.Schema.Fields) < 2 || len(f.Data.Values) < 2 {
			continue
		}
		out = append(out, frameSeries{Labels: f.Schema.Fields[1].Labels, Values: f.Data.Values[1]})
	}
	return out
}

func lastFinite(vs []*float64) (float64, bool) {
	for i := len(vs) - 1; i >= 0; i-- {
		if vs[i] != nil && !math.IsNaN(*vs[i]) && !math.IsInf(*vs[i], 0) {
			return *vs[i], true
		}
	}
	return 0, false
}

// T3.2-4
func TestOverviewPanelsHaveData(t *testing.T) {
	waitForUptime(t, "loadgen", 30*time.Second)
	panels := dashboardPanels(t)

	rates := map[string]float64{}
	for _, s := range panelQuery(t, panels["Request rate by service"], 5*time.Minute) {
		if v, ok := lastFinite(s.Values); ok {
			rates[s.Labels["service"]] = v
		}
	}
	for _, svc := range []string{"orders", "inventory"} {
		if rates[svc] <= 0 {
			t.Errorf("request rate for %s = %v, want > 0 (all: %v)", svc, rates[svc], rates)
		}
	}

	panelQuery(t, panels["Error rate (5xx) by endpoint"], 5*time.Minute)

	p95 := panelQuery(t, panels["p95 latency by service"], 5*time.Minute)
	finite := false
	for _, s := range p95 {
		if _, ok := lastFinite(s.Values); ok {
			finite = true
		}
	}
	if !finite {
		t.Errorf("p95 latency has no finite values: %+v", p95)
	}
}
