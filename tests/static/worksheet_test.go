package static

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const worksheetPath = "../../materials/worksheet.md"

type question struct {
	n     int
	title string
	body  string
}

func worksheetQuestions(t *testing.T) []question {
	t.Helper()
	heading := regexp.MustCompile(`^## (\d+)\. (.+)$`)
	var qs []question
	for _, l := range strings.Split(readFile(t, worksheetPath), "\n") {
		if m := heading.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[1])
			qs = append(qs, question{n: n, title: m[2]})
			continue
		}
		if strings.HasPrefix(l, "## ") {
			t.Errorf("worksheet heading %q is not a numbered scenario", l)
		}
		if len(qs) > 0 {
			qs[len(qs)-1].body += l + "\n"
		}
	}
	return qs
}

var incidentCmd = regexp.MustCompile(`\./incident\.sh ([A-Za-z]+)`)

// T6.2-1, updated for S6.5: the worksheet holds scenario 1 only. Scenarios 2
// to 6 are printed by ./unlock.sh; their order is checked by labctl's
// TestLabWorksheet.
func TestWorksheetStructure(t *testing.T) {
	qs := worksheetQuestions(t)
	if len(qs) != 1 || qs[0].n != 1 {
		t.Fatalf("worksheet scenarios %+v, want only 1", qs)
	}
	if !strings.Contains(qs[0].body, "Grafana skill: ") {
		t.Error("scenario 1 does not say which Grafana skill it practises")
	}
}

// T6.5-3: besides scenario 1, the worksheet has the intro, the tips, how to
// unlock, and the escape hatch. It never runs incident.sh: unlocking does.
func TestWorksheetGated(t *testing.T) {
	ws := readFile(t, worksheetPath)
	intro, q1, _ := strings.Cut(ws, "\n## 1. ")
	for _, w := range []string{"Work in pairs", "./unlock.sh 1 <answer>", "more than 5 minutes, ask the facilitator", "**Code**", "run the same `./unlock.sh` command again"} {
		if !strings.Contains(intro, w) {
			t.Errorf("worksheet intro does not say %q", w)
		}
	}
	if !strings.Contains(q1, "The password is ") {
		t.Error("scenario 1 does not say what its password is")
	}
	if m := incidentCmd.FindString(ws); m != "" {
		t.Errorf("worksheet runs %q; ./unlock.sh starts and clears the modes", m)
	}
}
