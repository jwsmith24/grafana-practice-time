package main

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fakeSwitch records the mode changes unlock makes.
type fakeSwitch struct{ calls []string }

func (f *fakeSwitch) Set(l string) error      { f.calls = append(f.calls, l); return nil }
func (f *fakeSwitch) Clear() error            { f.calls = append(f.calls, "clear"); return nil }
func (f *fakeSwitch) Duration() time.Duration { return 3 * time.Minute }

func runUnlock(ws worksheet, args ...string) (int, string, string) {
	code, out, errOut, _ := runUnlockSw(ws, args...)
	return code, out, errOut
}

func runUnlockSw(ws worksheet, args ...string) (int, string, string, []string) {
	var out, errb bytes.Buffer
	sw := &fakeSwitch{}
	code := unlock(args, ws, sw, &out, &errb)
	return code, out.String(), errb.String(), sw.calls
}

// T6.5-1, with made-up answers.
func TestUnlock(t *testing.T) {
	ws := worksheet{
		questions: []question{
			{title: "1. One", hash: answerHash(1, "alpha")},
			{title: "2. Two", body: "Second body.", mode: "B", hash: answerHash(2, "Beta Gamma")},
			{title: "3. Three", body: "Third body.", hash: answerHash(3, "v-9")},
		},
		closing: "The end.",
	}

	// A right answer prints the next question and starts its mode.
	code, out, _, calls := runUnlockSw(ws, "1", "alpha")
	if code != 0 || !strings.Contains(out, "## 2. Two\n\n") || !strings.Contains(out, "Second body.") || !strings.Contains(out, "./unlock.sh 2 <answer>") {
		t.Fatalf("right answer: %d %q", code, out)
	}
	if strings.Join(calls, ",") != "B" || !strings.Contains(out, "Mode B is now running") || !strings.Contains(out, "./unlock.sh 1 <answer> again") {
		t.Fatalf("right answer: mode calls %v, output %q; want B started and how to restart it", calls, out)
	}
	// The next question needs no mode: the lab is cleared.
	if _, out, _, calls := runUnlockSw(ws, "2", "beta gamma"); strings.Join(calls, ",") != "clear" || strings.Contains(out, "Mode") {
		t.Errorf("answer 2: mode calls %v, output %q; want a clear and no mode line", calls, out)
	}
	if _, _, _, calls := runUnlockSw(ws, "3", "v-9"); strings.Join(calls, ",") != "clear" {
		t.Errorf("last answer: mode calls %v, want a clear", calls)
	}
	for _, a := range []string{"ALPHA", "  alpha ", `"alpha"`, "`alpha`", "'Alpha'"} {
		if code, _, _ := runUnlock(ws, "1", a); code != 0 {
			t.Errorf("answer %q: exit %d, want 0", a, code)
		}
	}
	if code, out, _ := runUnlock(ws, "2", "  beta   GAMMA "); code != 0 || !strings.Contains(out, "## 3. Three") {
		t.Errorf("multi-word answer: %d %q", code, out)
	}
	if code, out, _ := runUnlock(ws, "3", "V-9"); code != 0 || !strings.Contains(out, "The end.") || strings.Contains(out, "##") {
		t.Errorf("last answer: %d %q", code, out)
	}

	// A wrong answer gets the same neutral line whatever was tried.
	var wrong []string
	for _, args := range [][]string{{"1", "beta"}, {"1", "alph"}, {"2", "alpha"}, {"3", "v-8"}} {
		code, out, _, calls := runUnlockSw(ws, args...)
		if code != 1 || len(calls) != 0 {
			t.Errorf("%v: exit %d, mode calls %v; want 1 and none", args, code, calls)
		}
		wrong = append(wrong, out)
	}
	for _, out := range wrong {
		if out != wrong[0] || !strings.Contains(out, "Not quite") {
			t.Errorf("wrong-answer output %q, want the same neutral line every time", out)
		}
	}

	for _, args := range [][]string{{}, {"1"}, {"0", "alpha"}, {"4", "alpha"}, {"x", "alpha"}, {"1", "  "}, {"1", "alpha", "extra"}} {
		if code, out, errOut, calls := runUnlockSw(ws, args...); code != 2 || !strings.Contains(errOut, "usage") || out != "" || len(calls) != 0 {
			t.Errorf("%q: exit %d, stdout %q, stderr %q, mode calls %v; want usage, 2 and no mode change", args, code, out, errOut, calls)
		}
	}
}

func TestUnlockThroughRun(t *testing.T) {
	t.Setenv("LAB_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("INCIDENT_DURATION", "")
	if code, _, errOut := runLabctl(t, "unlock"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Fatalf("labctl unlock: %d %q", code, errOut)
	}
	if code, out, _ := runLabctl(t, "unlock", "1", "not the answer"); code != 1 || !strings.Contains(out, "Not quite") {
		t.Fatalf("labctl unlock 1 <wrong>: %d %q", code, out)
	}
}

// T6.5-1 (stored answers are hashes only) and the labctl half of T6.2-1: six
// numbered scenarios, the last a stretch; scenarios 3 to 5 start modes B, C
// and A, in that order, so the easiest one to read comes first; each names
// the Grafana skill it practises, and none asks the pair to switch modes
// themselves.
func TestLabWorksheet(t *testing.T) {
	hexHash := regexp.MustCompile(`^[0-9a-f]{64}$`)
	modes := []string{"", "", "B", "C", "A", ""}
	qs := labWorksheet.questions
	if len(qs) != len(modes) {
		t.Fatalf("%d scenarios, want %d", len(qs), len(modes))
	}
	for i, q := range qs {
		if !hexHash.MatchString(q.hash) {
			t.Errorf("scenario %d: stored answer %q is not a SHA-256 hex digest", i+1, q.hash)
		}
		if prefix := string(rune('1'+i)) + ". "; !strings.HasPrefix(q.title, prefix) {
			t.Errorf("scenario %d is %q, want it to start with %q", i+1, q.title, prefix)
		}
		if (i == 0) != (q.body == "") {
			t.Errorf("scenario %d: body %q; only scenario 1 (in the worksheet) has none", i+1, q.body)
		}
		if i > 0 && (!strings.Contains(q.body, "The password is ") || !strings.Contains(q.body, "Grafana skill: ")) {
			t.Errorf("scenario %d does not say what its password is and which Grafana skill it practises", i+1)
		}
		if q.mode != modes[i] {
			t.Errorf("scenario %d starts mode %q, want %q", i+1, q.mode, modes[i])
		}
		if strings.Contains(q.body, "incident.sh") {
			t.Errorf("scenario %d tells the pair to run incident.sh", i+1)
		}
	}
	if !strings.HasPrefix(qs[len(qs)-1].title, "6. Stretch: ") {
		t.Errorf("last scenario is %q, want a stretch", qs[len(qs)-1].title)
	}
	if labWorksheet.closing == "" {
		t.Error("no closing message")
	}
}
