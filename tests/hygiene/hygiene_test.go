// Package hygiene keeps the public repo free of internal details and of
// anything that looks like the answer key.
package hygiene

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const root = "../.."

func trackedFiles(t *testing.T) []string {
	t.Helper()
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}
	return strings.Fields(string(out))
}

func denylist(t *testing.T) []*regexp.Regexp {
	t.Helper()
	f, err := os.Open("denylist.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []*regexp.Regexp
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, regexp.MustCompile(l))
	}
	if len(out) == 0 {
		t.Fatal("denylist.txt has no patterns")
	}
	return out
}

// T5.6-5 (the denylist half; gitleaks runs in CI)
func TestNoInternalDetails(t *testing.T) {
	deny := denylist(t)
	answerName := regexp.MustCompile(`(?i)(answer|solution)`)
	for _, f := range trackedFiles(t) {
		if answerName.MatchString(filepath.Base(f)) {
			t.Errorf("%s: file name looks like an answer key", f)
		}
		if strings.HasPrefix(f, "tests/hygiene/") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range deny {
			if loc := re.FindIndex(b); loc != nil {
				line := 1 + strings.Count(string(b[:loc[0]]), "\n")
				t.Errorf("%s:%d matches %s", f, line, re)
			}
		}
	}
}

// causeWords are the words T4.1-5 keeps out of incident.sh's output, plus
// the evidence each mode leaves in the logs.
var causeWords = []string{"sku", "1042", "latency", "slow", "token", "auth", "credential", "rejected", "deploy", "1.4.3", "deadline", "timeout"}

// T6.2-2, and the same check for the cheat sheet. The demo script walks
// through mode B in the open, so it is only kept clear of the other modes'
// evidence.
func TestMaterialsRevealNothing(t *testing.T) {
	files := map[string][]string{
		"materials/logql-cheat-sheet.md": causeWords,
		"materials/worksheet.md":         causeWords,
		"materials/demo-script.md":       {"sku", "1042", "token", "auth", "credential", "rejected", "deploy", "1.4.3"},
	}
	for f, words := range files {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(b))
		for _, w := range words {
			if i := strings.Index(text, w); i >= 0 {
				t.Errorf("%s:%d mentions %q", f, 1+strings.Count(text[:i], "\n"), w)
			}
		}
	}
}
