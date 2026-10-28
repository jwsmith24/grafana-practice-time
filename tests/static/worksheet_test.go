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
}

// T6.5-3: the worksheet shows how to unlock the next scenario and how to get
// a hint, and never runs incident.sh: unlocking does.
func TestWorksheetGated(t *testing.T) {
	ws := readFile(t, worksheetPath)
	if !strings.Contains(ws, "./unlock.sh 1 <answer>") {
		t.Error("worksheet does not show how to unlock scenario 2")
	}
	if !strings.Contains(ws, "./hint.sh N") {
		t.Error("worksheet does not show how to get a hint")
	}
	if m := incidentCmd.FindString(ws); m != "" {
		t.Errorf("worksheet runs %q; ./unlock.sh starts and clears the modes", m)
	}
}
