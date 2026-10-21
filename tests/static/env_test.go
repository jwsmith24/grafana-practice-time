package static

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// envExample parses ../../.env.example into key/value pairs.
func envExample(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	env := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf(".env.example: bad line %q", line)
		}
		env[k] = v
	}
	return env
}

type composeService struct {
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment"`
	Build       *struct {
		Args map[string]string `json:"args"`
	} `json:"build"`
	MemLimit any `json:"mem_limit"`
	Volumes  []struct {
		Type     string `json:"type"`
		Source   string `json:"source"`
		Target   string `json:"target"`
		ReadOnly bool   `json:"read_only"`
	} `json:"volumes"`
}

// composeServices renders the compose file with extra docker compose
// arguments (such as --env-file) and returns its services.
func composeServices(t *testing.T, args ...string) map[string]composeService {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	cmd := append([]string{"compose", "-f", "../../docker-compose.yml"}, args...)
	out, err := exec.Command("docker", append(cmd, "config", "--format", "json")...).Output()
	if err != nil {
		t.Fatalf("docker compose config: %v", err)
	}
	var c struct {
		Services map[string]composeService `json:"services"`
	}
	if err := json.Unmarshal(out, &c); err != nil {
		t.Fatal(err)
	}
	return c.Services
}

var thirdParty = map[string]string{
	"grafana":    "GRAFANA_TAG",
	"loki":       "LOKI_TAG",
	"alloy":      "ALLOY_TAG",
	"prometheus": "PROMETHEUS_TAG",
}

// T5.1-1
func TestEnvExample(t *testing.T) {
	env := envExample(t)
	if env["GRAFANA_PORT"] != "3300" {
		t.Errorf("GRAFANA_PORT = %q, want 3300", env["GRAFANA_PORT"])
	}
	if env["IMAGE_REGISTRY"] == "" {
		t.Error("IMAGE_REGISTRY is not set")
	}
	keys := []string{"GO_TAG"}
	for _, k := range thirdParty {
		keys = append(keys, k)
	}
	for _, k := range keys {
		if env[k] == "" {
			t.Errorf("%s is not set", k)
		}
	}
}

// T5.1-2: with and without .env.example the images are the same, pinned,
// and prefixed by IMAGE_REGISTRY (Alloy and Prometheus tags start with v).
func TestImagesPinned(t *testing.T) {
	env := envExample(t)
	pinned := regexp.MustCompile(`^` + regexp.QuoteMeta(env["IMAGE_REGISTRY"]) + `/.+:v?\d+\.\d+\.\d+$`)
	defaults := composeServices(t)
	withEnv := composeServices(t, "--env-file", "../../.env.example")
	for svc, tagKey := range thirdParty {
		img := withEnv[svc].Image
		if !pinned.MatchString(img) {
			t.Errorf("%s image %q is not %s/<name>:<exact version>", svc, img, env["IMAGE_REGISTRY"])
		}
		if !strings.HasSuffix(img, ":"+env[tagKey]) {
			t.Errorf("%s image %q does not use %s=%s", svc, img, tagKey, env[tagKey])
		}
		if defaults[svc].Image != img {
			t.Errorf("%s: compose default %q differs from .env.example %q", svc, defaults[svc].Image, img)
		}
	}
	for _, svc := range []string{"orders", "inventory", "loadgen"} {
		b := withEnv[svc].Build
		if b == nil || b.Args["IMAGE_REGISTRY"] != env["IMAGE_REGISTRY"] || b.Args["GO_TAG"] != env["GO_TAG"] {
			t.Errorf("%s build args %+v, want IMAGE_REGISTRY and GO_TAG from .env.example", svc, b)
		}
	}
	moved := composeServices(t, "--env-file", writeEnv(t, "IMAGE_REGISTRY=ghcr.io/example"))
	for svc := range thirdParty {
		if !strings.HasPrefix(moved[svc].Image, "ghcr.io/example/") {
			t.Errorf("IMAGE_REGISTRY ignored by %s: %q", svc, moved[svc].Image)
		}
	}
}

func writeEnv(t *testing.T, lines ...string) string {
	t.Helper()
	p := t.TempDir() + "/env"
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// T5.1-5
func TestGrafanaPhoneHomeOff(t *testing.T) {
	env := composeServices(t)["grafana"].Environment
	for _, k := range []string{
		"GF_ANALYTICS_REPORTING_ENABLED",
		"GF_ANALYTICS_CHECK_FOR_UPDATES",
		"GF_ANALYTICS_CHECK_FOR_PLUGIN_UPDATES",
		"GF_NEWS_NEWS_FEED_ENABLED",
	} {
		if env[k] != "false" {
			t.Errorf("%s = %q, want false", k, env[k])
		}
	}
}

// T5.1-6
func TestSevenServices(t *testing.T) {
	want := []string{"alloy", "grafana", "inventory", "loadgen", "loki", "orders", "prometheus"}
	var got []string
	for svc := range composeServices(t) {
		got = append(got, svc)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("services %v, want %v", got, want)
	}
}

// T5.4-3
func TestNoRuntimePlugins(t *testing.T) {
	for k := range composeServices(t)["grafana"].Environment {
		if k == "GF_INSTALL_PLUGINS" || k == "GF_PLUGINS_PREINSTALL" || strings.HasPrefix(k, "GF_PLUGINS_PREINSTALL") {
			t.Errorf("grafana sets %s", k)
		}
	}
}
