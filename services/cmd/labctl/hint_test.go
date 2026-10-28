package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func runHint(ws worksheet, prog progress, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := hint(args, ws, prog, &out, &errb)
	return code, out.String(), errb.String()
}

func TestHint(t *testing.T) {
	ws := worksheet{
		questions: []question{
			{title: "1. One", hints: [3]string{"one-a", "one-b", "one-c"}},
			{title: "2. Two", hints: [3]string{"two-a", "two-b", "two-c"}},
			{title: "3. Three", hints: [3]string{"three-a", "three-b", "three-c"}},
		},
	}
	prog := tempProgress(t)

	// Scenario 1's hints are there from the start; the level defaults to 1.
	if code, out, _ := runHint(ws, prog, "1"); code != 0 || !strings.HasPrefix(out, "Hint 1 of 3 for scenario 1:") || !strings.Contains(out, "one-a") {
		t.Fatalf("hint 1: %d %q", code, out)
	}
	if code, out, _ := runHint(ws, prog, "1", "3"); code != 0 || !strings.HasPrefix(out, "Hint 3 of 3 for scenario 1:") || !strings.Contains(out, "one-c") {
		t.Fatalf("hint 1 3: %d %q", code, out)
	}

	// Later scenarios only once they're unlocked.
	for _, n := range []string{"2", "3"} {
		if code, out, _ := runHint(ws, prog, n); code == 0 || out != "Scenario "+n+" isn't unlocked yet.\n" {
			t.Errorf("hint %s while locked: %d %q", n, code, out)
		}
	}
	if err := prog.Record(2); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runHint(ws, prog, "2", "2"); code != 0 || !strings.HasPrefix(out, "Hint 2 of 3 for scenario 2:") || !strings.Contains(out, "two-b") {
		t.Errorf("hint 2 2 once unlocked: %d %q", code, out)
	}
	if code, _, _ := runHint(ws, prog, "3"); code == 0 {
		t.Error("hint 3 before scenario 3 is unlocked: exit 0")
	}

	// Bad arguments: usage and 2, and the progress never moves.
	for _, args := range [][]string{{}, {"0"}, {"4"}, {"x"}, {"1", "0"}, {"1", "4"}, {"1", "x"}, {"1", "1", "1"}} {
		if code, out, errOut := runHint(ws, prog, args...); code != 2 || out != "" || !strings.Contains(errOut, "usage") {
			t.Errorf("hint %q: exit %d, stdout %q, stderr %q; want usage and 2", args, code, out, errOut)
		}
	}
	if got, _ := prog.Unlocked(); got != 2 {
		t.Errorf("progress %d after hints, want 2", got)
	}
}

// Every hint of the real worksheet, once every scenario is unlocked.
func TestHintLabWorksheet(t *testing.T) {
	prog := tempProgress(t)
	if err := prog.Record(len(labWorksheet.questions) + 1); err != nil {
		t.Fatal(err)
	}
	for i, q := range labWorksheet.questions {
		for j, h := range q.hints {
			n, level := fmt.Sprint(i+1), fmt.Sprint(j+1)
			code, out, _ := runHint(labWorksheet, prog, n, level)
			want := fmt.Sprintf("Hint %d of 3 for scenario %d:\n\n%s\n", j+1, i+1, h)
			if code != 0 || !strings.HasPrefix(out, want) {
				t.Errorf("hint %s %s: %d %q, want it to start with %q", n, level, code, out, want)
			}
		}
	}
}

func TestProgress(t *testing.T) {
	state := filepath.Join(t.TempDir(), "lab", "state.json")
	unlocked := func(p progress) int {
		t.Helper()
		n, err := p.Unlocked()
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	p := progressFor(state)
	if n := unlocked(p); n != 1 {
		t.Fatalf("empty state directory: progress %d, want 1", n)
	}
	for _, step := range []struct{ record, want int }{{3, 3}, {2, 3}, {1, 3}, {5, 5}} {
		if err := p.Record(step.record); err != nil {
			t.Fatal(err)
		}
		if n := unlocked(p); n != step.want {
			t.Errorf("after Record(%d): %d, want %d", step.record, n, step.want)
		}
	}
	// A new labctl process reads the same file.
	if n := unlocked(progressFor(state)); n != 5 {
		t.Errorf("after a restart: %d, want 5", n)
	}
	// Another, empty state directory starts over.
	if n := unlocked(tempProgress(t)); n != 1 {
		t.Errorf("fresh state directory: %d, want 1", n)
	}
}

func TestHintThroughRun(t *testing.T) {
	t.Setenv("LAB_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("INCIDENT_DURATION", "")
	if code, out, _ := runLabctl(t, "hint", "1", "2"); code != 0 || !strings.HasPrefix(out, "Hint 2 of 3 for scenario 1:") {
		t.Fatalf("labctl hint 1 2: %d %q", code, out)
	}
	if code, out, _ := runLabctl(t, "hint", "2"); code != 1 || out != "Scenario 2 isn't unlocked yet.\n" {
		t.Fatalf("labctl hint 2: %d %q", code, out)
	}
	if code, _, errOut := runLabctl(t, "hint"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Fatalf("labctl hint: %d %q", code, errOut)
	}
	if code, out, _ := runLabctl(t, "status"); code != 0 || out != "none\nprogress: scenario 1 of 6 unlocked\n" {
		t.Fatalf("status after hints: %d %q", code, out)
	}
}
