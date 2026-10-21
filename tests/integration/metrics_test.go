//go:build integration

package integration

import (
	"testing"
	"time"
)

const promAPI = "/api/datasources/proxy/uid/prometheus/api/v1"

// T3.1-4
func TestPrometheusTargetsUp(t *testing.T) {
	var health map[string]string
	ok := eventually(t, 30*time.Second, func() bool {
		code, b := do(t, "GET", promAPI+"/targets", nil, false)
		if code != 200 {
			t.Fatalf("targets: status %d: %s", code, b)
		}
		var r struct {
			Data struct {
				ActiveTargets []struct {
					Labels    map[string]string `json:"labels"`
					Health    string            `json:"health"`
					ScrapeURL string            `json:"scrapeUrl"`
				} `json:"activeTargets"`
			} `json:"data"`
		}
		decode(t, b, &r)
		health = map[string]string{}
		for _, tg := range r.Data.ActiveTargets {
			health[tg.Labels["job"]] = tg.Health
		}
		return health["orders"] == "up" && health["inventory"] == "up"
	})
	if !ok {
		t.Fatalf("target health: %v", health)
	}
}

// T3.1-6
func TestPrometheusDatasource(t *testing.T) {
	code, b := do(t, "GET", "/api/datasources/uid/prometheus", nil, true)
	if code != 200 {
		t.Fatalf("status %d: %s", code, b)
	}
	var ds struct{ Type string }
	decode(t, b, &ds)
	if ds.Type != "prometheus" {
		t.Fatalf("type %q", ds.Type)
	}
}
