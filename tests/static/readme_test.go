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

// T5.6-1
func TestReadmeSections(t *testing.T) {
	s := readmeSections(t)
	for _, h := range []string{"Prerequisites", "Setup", "Check your lab", "Worksheet", "Reset", "Troubleshooting"} {
		if _, ok := s[h]; !ok {
			t.Errorf("README has no ## %s section", h)
		}
	}
	pre := s["Prerequisites"]
	for _, w := range []string{"Docker Desktop", "2 GB", "3 GB", "Git"} {
		if !strings.Contains(pre, w) {
			t.Errorf("Prerequisites does not mention %q", w)
		}
	}
	setup := s["Setup"]
	for _, w := range []string{"cp .env.example .env", "docker compose up -d", "./check.sh", "localhost:3300", "Lab Overview"} {
		if !strings.Contains(setup, w) {
			t.Errorf("Setup does not mention %q", w)
		}
	}
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

// T5.6-4: every check.sh fix hint is explained in Troubleshooting. A hint's
// first sentence is matched, with its shell variables as placeholders.
func TestReadmeTroubleshooting(t *testing.T) {
	trouble := readmeSections(t)["Troubleshooting"]
	placeholders := strings.NewReplacer("${missing# }", "<services>", "$port", "<port>", "$grafana", "<url>", "$service", "<service>")
	hints := regexp.MustCompile(`fail "[^"]*" "([^"]*)"`).FindAllStringSubmatch(readFile(t, "../../check.sh"), -1)
	if len(hints) < 8 {
		t.Fatalf("found %d fix hints in check.sh, want at least 8", len(hints))
	}
	for _, h := range hints {
		first, _, _ := strings.Cut(h[1], ". ")
		first = placeholders.Replace(strings.TrimSuffix(first, "."))
		if strings.Contains(first, "$") {
			t.Errorf("hint %q has a variable with no placeholder", first)
			continue
		}
		if !strings.Contains(trouble, "**"+first+".**") {
			t.Errorf("Troubleshooting has no **%s.** entry", first)
		}
	}
}

// T5.6-6
func TestCheckFooter(t *testing.T) {
	if !strings.Contains(readFile(t, "../../check.sh"), `echo "If it still fails, message the facilitator and include this output."`) {
		t.Error("check.sh's failure footer does not tell attendees to message the facilitator")
	}
}

// T6.5-6
func TestReadmeWorksheet(t *testing.T) {
	sec := readmeSections(t)["Worksheet"]
	for _, w := range []string{"materials/worksheet.md", "./unlock.sh 1 <answer>"} {
		if !strings.Contains(sec, w) {
			t.Errorf("the Worksheet section does not mention %q", w)
		}
	}
}
