package static

import (
	"os"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// T3.1-5
func TestPrometheusConfig(t *testing.T) {
	b, err := os.ReadFile("../../prometheus/prometheus.yml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Global struct {
			ScrapeInterval string `yaml:"scrape_interval"`
		} `yaml:"global"`
		ScrapeConfigs []struct {
			JobName        string `yaml:"job_name"`
			ScrapeInterval string `yaml:"scrape_interval"`
			StaticConfigs  []struct {
				Targets []string `yaml:"targets"`
			} `yaml:"static_configs"`
		} `yaml:"scrape_configs"`
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Global.ScrapeInterval != "5s" {
		t.Fatalf("global scrape_interval %q, want 5s", cfg.Global.ScrapeInterval)
	}
	var targets []string
	for _, sc := range cfg.ScrapeConfigs {
		if sc.ScrapeInterval != "" && sc.ScrapeInterval != "5s" {
			t.Errorf("job %s overrides scrape_interval to %s", sc.JobName, sc.ScrapeInterval)
		}
		for _, st := range sc.StaticConfigs {
			targets = append(targets, st.Targets...)
		}
	}
	for _, want := range []string{"orders:8080", "inventory:8080"} {
		if !slices.Contains(targets, want) {
			t.Errorf("no scrape target %s (have %v)", want, targets)
		}
	}
}
