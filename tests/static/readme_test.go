package static

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// readmeSections splits README.md into its ## sections, keyed by heading.
func readmeSections(t *testing.T) map[string]string {
	t.Helper()
	sections := map[string]string{}
	var cur string
	for _, l := range strings.Split(readFile(t, "../../README.md"), "\n") {
		if h, ok := strings.CutPrefix(l, "## "); ok {
			cur = h
			sections[cur] = ""
			continue
		}
		if cur != "" {
			sections[cur] += l + "\n"
		}
	}
	return sections
}

// T5.6-2
func TestReadmeCommands(t *testing.T) {
	readme := readFile(t, "../../README.md")
	for _, f := range regexp.MustCompile("(?s)```sh\n(.*?)```").FindAllStringSubmatch(readme, -1) {
		if strings.Contains(f[1], "./incident.sh") {
			t.Error("README runs ./incident.sh; attendees start modes with ./unlock.sh")
		}
	}
	port := envExample(t)["GRAFANA_PORT"]
	for _, m := range regexp.MustCompile(`localhost:(\d+)`).FindAllStringSubmatch(readme, -1) {
		if m[1] != port && !strings.Contains(readme, "GRAFANA_PORT="+m[1]) {
			t.Errorf("README uses localhost:%s, but .env.example has GRAFANA_PORT=%s", m[1], port)
		}
	}
	for _, m := range regexp.MustCompile(`\]\(#([^)]+)\)`).FindAllStringSubmatch(readme, -1) {
		if !hasAnchor(t, m[1]) {
			t.Errorf("README links to #%s, but has no such section", m[1])
		}
	}
}

// hasAnchor reports whether a README ## section has the given GitHub anchor.
func hasAnchor(t *testing.T, anchor string) bool {
	t.Helper()
	for h := range readmeSections(t) {
		if strings.ReplaceAll(strings.ToLower(h), " ", "-") == anchor {
			return true
		}
	}
	return false
}

// T5.6-3: the same words T4.1-5 keeps out of incident.sh's output.
func TestReadmeRevealsNothing(t *testing.T) {
	readme := strings.ToLower(readFile(t, "../../README.md"))
	for _, w := range []string{"sku", "latency", "slow", "token", "auth", "deploy", "1.4.3"} {
		if strings.Contains(readme, w) {
			t.Errorf("README mentions %q", w)
		}
	}
}
