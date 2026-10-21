//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
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

type dashTarget struct {
	RefID      string            `json:"refId"`
	Datasource map[string]string `json:"datasource"`
	Expr       string            `json:"expr"`
}

type dashPanel struct {
	Title   string       `json:"title"`
	Targets []dashTarget `json:"targets"`
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

// panelRaw runs a panel's first target through Grafana's /api/ds/query
// over the last window and returns the raw response.
func panelRaw(t *testing.T, p dashPanel, window time.Duration) []byte {
	t.Helper()
	if len(p.Targets) == 0 {
		t.Fatalf("panel %q has no targets", p.Title)
	}
	tg := p.Targets[0]
	query := map[string]any{
		"refId":      tg.RefID,
		"datasource": tg.Datasource,
		"expr":       tg.Expr,
		"range":      true,
	}
	if tg.Datasource["type"] == "loki" {
		query["queryType"] = "range"
		query["maxLines"] = 1000
	}
	q := map[string]any{
		"from":    fmt.Sprintf("now-%dm", int(window.Minutes())),
		"to":      "now",
		"queries": []map[string]any{query},
	}
	code, b := do(t, "POST", "/api/ds/query", q, false)
	if code != 200 {
		t.Fatalf("%s: status %d: %s", p.Title, code, b)
	}
	return b
}

// panelQuery runs a panel's first target and returns one numeric series per
// frame.
func panelQuery(t *testing.T, p dashPanel, window time.Duration) []frameSeries {
	t.Helper()
	tg := p.Targets
	b := panelRaw(t, p, window)
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
	res := r.Results[tg[0].RefID]
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

// T3.3-3
func TestLokiPanelsRun(t *testing.T) {
	panels := dashboardPanels(t)
	for _, title := range []string{"Error log volume by service", "Recent errors"} {
		b := panelRaw(t, panels[title], 5*time.Minute)
		var r struct {
			Results map[string]struct {
				Error string `json:"error"`
			} `json:"results"`
		}
		decode(t, b, &r)
		for ref, res := range r.Results {
			if res.Error != "" {
				t.Errorf("%s %s: %s", title, ref, res.Error)
			}
		}
	}
}

// T3.3-4: an order for an unknown item makes orders log an ERROR, which
// both panels must show.
func TestLokiPanelsShowErrors(t *testing.T) {
	id := newUUID(t)
	req, err := http.NewRequest("POST", ordersURL()+"/orders", strings.NewReader(`{"item_id":"SKU-0000","quantity":1}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", id)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("POST /orders unknown item: status %d, want 404", resp.StatusCode)
	}

	panels := dashboardPanels(t)
	var volume float64
	ok := eventually(t, 30*time.Second, func() bool {
		volume = 0
		for _, s := range panelQuery(t, panels["Error log volume by service"], 5*time.Minute) {
			if s.Labels["service"] != "orders" {
				continue
			}
			for _, v := range s.Values {
				if v != nil {
					volume += *v
				}
			}
		}
		return volume > 0 && bytes.Contains(panelRaw(t, panels["Recent errors"], 5*time.Minute), []byte(id))
	})
	if !ok {
		t.Fatalf("orders error volume %v; recent errors panel has request_id %s: %v", volume, id,
			bytes.Contains(panelRaw(t, panels["Recent errors"], 5*time.Minute), []byte(id)))
	}
}

// deployAnnotation returns the dashboard's Deploys annotation query as a
// panel, so panelRaw can run it.
func deployAnnotation(t *testing.T) dashPanel {
	t.Helper()
	b, err := os.ReadFile("../../grafana/provisioning/dashboards/lab-overview.json")
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Annotations struct {
			List []struct {
				Name       string            `json:"name"`
				Datasource map[string]string `json:"datasource"`
				Expr       string            `json:"expr"`
			} `json:"list"`
		} `json:"annotations"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	var expr string
	for _, a := range d.Annotations.List {
		if a.Name == "Deploys" {
			expr = a.Expr
		}
	}
	if expr == "" {
		t.Fatal("no Deploys annotation")
	}
	return dashPanel{Title: "Deploys", Targets: []dashTarget{
		{RefID: "Anno", Datasource: map[string]string{"type": "loki", "uid": "loki"}, Expr: expr},
	}}
}

// T3.5-3
func TestDeployAnnotationQuery(t *testing.T) {
	p := deployAnnotation(t)
	expr := p.Targets[0].Expr
	if !eventually(t, 30*time.Second, func() bool {
		return bytes.Contains(panelRaw(t, p, 15*time.Minute), []byte(`\"event\":\"deploy\"`))
	}) {
		t.Fatalf("annotation query %s returned no deploy events", expr)
	}
}
