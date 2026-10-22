package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func runUnlock(ws worksheet, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := unlock(args, ws, &out, &errb)
	return code, out.String(), errb.String()
}

// T6.5-1, with made-up answers.
func TestUnlock(t *testing.T) {
	ws := worksheet{
		questions: []question{
			{title: "1. One", hash: answerHash(1, "alpha")},
			{title: "2. Two", body: "Second body.", hash: answerHash(2, "Beta Gamma")},
			{title: "3. Three", body: "Third body.", hash: answerHash(3, "v-9")},
		},
		closing: "The end.",
	}

	code, out, _ := runUnlock(ws, "1", "alpha")
	if code != 0 || !strings.Contains(out, "## 2. Two\n\nSecond body.") || !strings.Contains(out, "./unlock.sh 2 <answer>") {
		t.Fatalf("right answer: %d %q", code, out)
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
		code, out, _ := runUnlock(ws, args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1", args, code)
		}
		wrong = append(wrong, out)
	}
	for _, out := range wrong {
		if out != wrong[0] || !strings.Contains(out, "Not quite") {
			t.Errorf("wrong-answer output %q, want the same neutral line every time", out)
		}
	}

	for _, args := range [][]string{{}, {"1"}, {"0", "alpha"}, {"4", "alpha"}, {"x", "alpha"}, {"1", "  "}, {"1", "alpha", "extra"}} {
		if code, out, errOut := runUnlock(ws, args...); code != 2 || !strings.Contains(errOut, "usage") || out != "" {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want usage and 2", args, code, out, errOut)
		}
	}
}

func TestUnlockThroughRun(t *testing.T) {
	if code, _, errOut := runLabctl(t, "unlock"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Fatalf("labctl unlock: %d %q", code, errOut)
	}
	if code, out, _ := runLabctl(t, "unlock", "1", "not the answer"); code != 1 || !strings.Contains(out, "Not quite") {
		t.Fatalf("labctl unlock 1 <wrong>: %d %q", code, out)
	}
}

// T6.5-1 (stored answers are hashes only) and the labctl half of T6.2-1: the
// questions after the worksheet's first come in the order warm-up, mode A, B,
// C, stretch, and each mode question runs only its own letter.
func TestLabWorksheet(t *testing.T) {
	hexHash := regexp.MustCompile(`^[0-9a-f]{64}$`)
	kinds := []string{"Warm-up: ", "Warm-up: ", "Mode A", "Mode B", "Mode C", "Stretch: "}
	qs := labWorksheet.questions
	if len(qs) != len(kinds) {
		t.Fatalf("%d questions, want %d", len(qs), len(kinds))
	}
	incidentCmd := regexp.MustCompile(`\./incident\.sh ([A-Za-z]+)`)
	for i, q := range qs {
		if !hexHash.MatchString(q.hash) {
			t.Errorf("question %d: stored answer %q is not a SHA-256 hex digest", i+1, q.hash)
		}
		prefix := string(rune('1'+i)) + ". " + kinds[i]
		if !strings.HasPrefix(q.title, prefix) {
			t.Errorf("question %d is %q, want it to start with %q", i+1, q.title, prefix)
		}
		if (i == 0) != (q.body == "") {
			t.Errorf("question %d: body %q; only question 1 (in the worksheet) has none", i+1, q.body)
		}
		if i > 0 && !strings.Contains(q.body, "The password is ") {
			t.Errorf("question %d does not say what its password is", i+1)
		}
		var letters []string
		for _, m := range incidentCmd.FindAllStringSubmatch(q.body, -1) {
			letters = append(letters, m[1])
		}
		want := ""
		if letter, ok := strings.CutPrefix(kinds[i], "Mode "); ok {
			want = letter
		}
		if strings.Join(letters, ",") != want {
			t.Errorf("question %d runs ./incident.sh %v, want %q", i+1, letters, want)
		}
	}
	if labWorksheet.closing == "" {
		t.Error("no closing message")
	}
}
