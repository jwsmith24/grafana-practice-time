package static

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const demoPath = "../../materials/demo-script.md"

// T6.4-1
func TestDemoScriptStructure(t *testing.T) {
	text := readFile(t, demoPath)
	steps := regexp.MustCompile(`(?m)^## Step \d+\. `).Split(text, -1)[1:]
	if len(steps) < 5 {
		t.Fatalf("%d steps, want at least 5", len(steps))
	}
	minutes := 0
	for _, m := range regexp.MustCompile(`(?m)^## .*\((\d+) min\)$`).FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1])
		minutes += n
	}
	if minutes == 0 || minutes > 15 {
		t.Errorf("the sections add up to %d min, want 1-15", minutes)
	}
	image := regexp.MustCompile(`!\[[^\]]+\]\(([^)]+\.png)\)`)
	for i, s := range steps {
		if s, _, _ = strings.Cut(s, "\n## "); !fenceRe.MatchString(s) {
			t.Errorf("step %d has no query block", i+1)
		}
		for _, f := range fenceRe.FindAllStringSubmatch(s, -1) {
			if f[1] != "logql" && f[1] != "promql" {
				t.Errorf("step %d has a %q block, want logql or promql", i+1, f[1])
			}
		}
		imgs := image.FindAllStringSubmatch(s, -1)
		if len(imgs) == 0 {
			t.Errorf("step %d has no screenshot", i+1)
		}
		for _, m := range imgs {
			p := filepath.Clean(filepath.Join(filepath.Dir(demoPath), m[1]))
			if !strings.HasPrefix(p, filepath.Clean("../../materials")+string(filepath.Separator)) {
				t.Errorf("step %d screenshot %s is outside materials/", i+1, m[1])
			}
			if _, err := os.Stat(p); err != nil {
				t.Errorf("step %d screenshot: %v", i+1, err)
			}
		}
	}
}

// T6.4-2
func TestDemoScriptUsesOnlyB(t *testing.T) {
	text := readFile(t, demoPath)
	ms := incidentCmd.FindAllStringSubmatch(text, -1)
	if len(ms) == 0 {
		t.Fatal("the demo never runs ./incident.sh B")
	}
	for _, m := range ms {
		if m[1] != "B" && m[1] != "clear" && m[1] != "status" {
			t.Errorf("the demo runs ./incident.sh %s", m[1])
		}
	}
	if m := regexp.MustCompile(`\b(?i:modes?|incidents?) [AC]\b|\b[AC] (?i:mode|incident)\b`).FindString(text); m != "" {
		t.Errorf("the demo mentions %q", m)
	}
}
