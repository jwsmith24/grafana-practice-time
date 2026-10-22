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

// T6.2-1
func TestWorksheetStructure(t *testing.T) {
	qs := worksheetQuestions(t)
	kinds := []string{"Warm-up: ", "Warm-up: ", "Mode A", "Mode B", "Mode C", "Stretch: "}
	if len(qs) != len(kinds) {
		t.Fatalf("%d questions, want %d", len(qs), len(kinds))
	}
	for i, q := range qs {
		if q.n != i+1 {
			t.Errorf("question %d is numbered %d", i+1, q.n)
		}
		if !strings.HasPrefix(q.title, kinds[i]) {
			t.Errorf("question %d is %q, want it to start with %q", i+1, q.title, kinds[i])
		}
		var letters []string
		for _, m := range incidentCmd.FindAllStringSubmatch(q.body, -1) {
			letters = append(letters, m[1])
		}
		want := []string{}
		if letter, ok := strings.CutPrefix(kinds[i], "Mode "); ok {
			want = []string{letter}
		}
		if strings.Join(letters, ",") != strings.Join(want, ",") {
			t.Errorf("question %d runs ./incident.sh %v, want %v", i+1, letters, want)
		}
	}
}
