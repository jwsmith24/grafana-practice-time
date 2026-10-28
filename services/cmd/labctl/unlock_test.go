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

// tempProgress returns progress kept in a fresh state directory.
func tempProgress(t *testing.T) progress {
	t.Helper()
	return progressFor(filepath.Join(t.TempDir(), "state.json"))
}

func runUnlock(t *testing.T, ws worksheet, args ...string) (int, string, string) {
	t.Helper()
	code, out, errOut, _ := runUnlockSw(t, ws, args...)
	return code, out, errOut
}

func runUnlockSw(t *testing.T, ws worksheet, args ...string) (int, string, string, []string) {
	t.Helper()
	return runUnlockProg(ws, tempProgress(t), args...)
}

func runUnlockProg(ws worksheet, prog progress, args ...string) (int, string, string, []string) {
	var out, errb bytes.Buffer
	sw := &fakeSwitch{}
	code := unlock(args, ws, sw, prog, &out, &errb)
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
	code, out, _, calls := runUnlockSw(t, ws, "1", "alpha")
	if code != 0 || !strings.Contains(out, "Scenario 2 of 3\n\nSecond body.") || !strings.Contains(out, "./hint.sh 2") || !strings.Contains(out, "./unlock.sh 2 <answer>") || strings.Contains(out, "Two") {
		t.Fatalf("right answer: %d %q", code, out)
	}
	if strings.Join(calls, ",") != "B" {
		t.Fatalf("right answer: mode calls %v; want B started", calls)
	}
	// The next question needs no mode: the lab is cleared.
	if _, _, _, calls := runUnlockSw(t, ws, "2", "beta gamma"); strings.Join(calls, ",") != "clear" {
		t.Errorf("answer 2: mode calls %v; want a clear", calls)
	}
	if _, _, _, calls := runUnlockSw(t, ws, "3", "v-9"); strings.Join(calls, ",") != "clear" {
		t.Errorf("last answer: mode calls %v, want a clear", calls)
	}
	for _, a := range []string{"ALPHA", "  alpha ", `"alpha"`, "`alpha`", "'Alpha'"} {
		if code, _, _ := runUnlock(t, ws, "1", a); code != 0 {
			t.Errorf("answer %q: exit %d, want 0", a, code)
		}
	}
	if code, out, _ := runUnlock(t, ws, "2", "  beta   GAMMA "); code != 0 || !strings.Contains(out, "Scenario 3 of 3") {
		t.Errorf("multi-word answer: %d %q", code, out)
	}
	if code, out, _ := runUnlock(t, ws, "3", "V-9"); code != 0 || !strings.Contains(out, "The end.") || strings.Contains(out, "Scenario") {
		t.Errorf("last answer: %d %q", code, out)
	}

	// A wrong answer gets the same neutral line whatever was tried.
	var wrong []string
	for _, args := range [][]string{{"1", "beta"}, {"1", "alph"}, {"2", "alpha"}, {"3", "v-8"}} {
		code, out, _, calls := runUnlockSw(t, ws, args...)
		if code != 1 || len(calls) != 0 {
			t.Errorf("%v: exit %d, mode calls %v; want 1 and none", args, code, calls)
		}
		wrong = append(wrong, out)
	}
	for _, out := range wrong {
		if out != wrong[0] || out == "" {
			t.Errorf("wrong-answer output %q, want the same neutral line every time", out)
		}
	}

	for _, args := range [][]string{{}, {"1"}, {"0", "alpha"}, {"4", "alpha"}, {"x", "alpha"}, {"1", "  "}, {"1", "alpha", "extra"}} {
		if code, out, errOut, calls := runUnlockSw(t, ws, args...); code != 2 || !strings.Contains(errOut, "usage") || out != "" || len(calls) != 0 {
			t.Errorf("%q: exit %d, stdout %q, stderr %q, mode calls %v; want usage, 2 and no mode change", args, code, out, errOut, calls)
		}
	}
}

// A right answer records the next scenario as unlocked; a wrong one, bad
// input or an earlier answer leaves the progress where it was.
func TestUnlockRecordsProgress(t *testing.T) {
	ws := worksheet{
		questions: []question{
			{title: "1. One", hash: answerHash(1, "alpha")},
			{title: "2. Two", body: "Second body.", hash: answerHash(2, "beta")},
			{title: "3. Three", body: "Third body.", hash: answerHash(3, "gamma")},
		},
		closing: "The end.",
	}
	prog := tempProgress(t)
	step := func(want int, args ...string) {
		t.Helper()
		runUnlockProg(ws, prog, args...)
		if got, err := prog.Unlocked(); err != nil || got != want {
			t.Fatalf("after unlock %q: progress %d, %v; want %d", args, got, err, want)
		}
	}
	step(1, "1", "wrong")
	step(1, "1")
	step(2, "1", "alpha")
	step(3, "2", "beta")
	step(3, "1", "alpha")
	step(4, "3", "gamma")
}

func TestUnlockThroughRun(t *testing.T) {
	t.Setenv("LAB_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("INCIDENT_DURATION", "")
	if code, _, errOut := runLabctl(t, "unlock"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Fatalf("labctl unlock: %d %q", code, errOut)
	}
	if code, out, _ := runLabctl(t, "unlock", "1", "not the answer"); code != 1 {
		t.Fatalf("labctl unlock 1 <wrong>: %d %q", code, out)
	}
}

// T6.5-1 (stored answers are hashes only) and the labctl half of T6.2-1: six
// numbered scenarios; scenarios 3 to 5 start modes B, C and A, in that
// order, and none asks the pair to switch modes themselves.
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
		if q.mode != modes[i] {
			t.Errorf("scenario %d starts mode %q, want %q", i+1, q.mode, modes[i])
		}
		if strings.Contains(q.body, "incident.sh") {
			t.Errorf("scenario %d tells the pair to run incident.sh", i+1)
		}
		for j, h := range q.hints {
			if strings.TrimSpace(h) == "" {
				t.Errorf("scenario %d has no hint %d", i+1, j+1)
			}
		}
		for j := range i {
			if q.hash == qs[j].hash {
				t.Errorf("scenarios %d and %d store the same hash", j+1, i+1)
			}
		}
	}
	if labWorksheet.closing == "" {
		t.Error("no closing message")
	}
}

// The text is printed from Go raw strings, which can't hold a backtick.
func TestNoBackticks(t *testing.T) {
	texts := []string{labWorksheet.closing}
	for _, q := range labWorksheet.questions {
		texts = append(texts, q.title, q.body)
		texts = append(texts, q.hints[:]...)
	}
	for _, s := range texts {
		if strings.Contains(s, "`") {
			t.Errorf("text contains a backtick: %q", s)
		}
	}
}
