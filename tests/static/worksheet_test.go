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
			t.Errorf("worksheet heading %q is not a numbered question", l)
		}
		if len(qs) > 0 {
			qs[len(qs)-1].body += l + "\n"
		}
	}
	return qs
}

var incidentCmd = regexp.MustCompile(`\./incident\.sh ([A-Za-z]+)`)

// T6.2-1, updated for S6.5: the worksheet holds question 1 only, a warm-up.
// Questions 2 to 6 are printed by ./unlock.sh; their order is checked by
// labctl's TestLabWorksheet.
func TestWorksheetStructure(t *testing.T) {
	qs := worksheetQuestions(t)
	if len(qs) != 1 || qs[0].n != 1 || !strings.HasPrefix(qs[0].title, "Warm-up: ") {
		t.Fatalf("worksheet questions %+v, want only 1. Warm-up: ...", qs)
	}
}

// T6.5-3: besides question 1, the worksheet has the intro, the notes
// template, how to unlock, and the escape hatch, and runs no mode itself.
func TestWorksheetGated(t *testing.T) {
	ws := readFile(t, worksheetPath)
	intro, q1, _ := strings.Cut(ws, "\n## 1. ")
	for _, w := range []string{"Work in pairs", "./unlock.sh 1 <answer>", "more than 5 minutes, ask the facilitator", "For questions 3 to 5, write down:", "**Root cause:**", "./incident.sh status"} {
		if !strings.Contains(intro, w) {
			t.Errorf("worksheet intro does not say %q", w)
		}
	}
	if !strings.Contains(q1, "The password is ") {
		t.Error("question 1 does not say what its password is")
	}
	for _, m := range incidentCmd.FindAllStringSubmatch(ws, -1) {
		if m[1] != "status" {
			t.Errorf("worksheet runs %q; the mode questions are printed by ./unlock.sh", m[0])
		}
	}
}
