//go:build integration

package integration

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This file sorts after materials_test.go, whose mode chains it reuses, and
// before session_test.go.

// unlockSh runs ./unlock.sh n answer and returns its output and exit code.
func unlockSh(t *testing.T, n int, answer string) (string, int) {
	t.Helper()
	cmd := exec.Command("../../unlock.sh", strconv.Itoa(n), answer)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(out), exit.ExitCode()
	}
	if err != nil {
		t.Fatalf("unlock.sh %d: %v", n, err)
	}
	return string(out), 0
}

// T6.5-4: the gated worksheet end to end. Each answer comes from the lab,
// found the way the pair finds it, and unlocks the next question; the last
// one prints the closing message. Only question 1's field and question 4's
// service are written down here.
func TestGatedWorksheet(t *testing.T) {
	if out, code := unlockSh(t, 1, "not it"); code != 1 || !strings.Contains(out, "Not quite") {
		t.Fatalf("wrong answer: exit %d, %q", code, out)
	}

	answers := map[int]string{1: "request_id", 4: "inventory"}

	// 2: the service the followed request reached second, after the one that
	// logged the POST /orders request line.
	id := recentRequestID(t)
	lines := eventuallyLines(t, fill(t, template(t, "request_id="), map[string]string{"request_id": id}), time.Now().Add(-2*time.Minute))
	var others []string
	for _, l := range lines {
		if l.str("endpoint") != "POST /orders" {
			others = append(others, l.labels["service"])
		}
	}
	if len(others) != 1 {
		t.Fatalf("request %s: other services %v, want one", id, others)
	}
	answers[2] = others[0]

	t.Run("A", func(t *testing.T) { answers[3] = chainA(t) })
	t.Run("C", func(t *testing.T) { answers[5] = chainC(t) })

	// 6: the key the most frequent WARN message names.
	top := lokiVector(t, fill(t, template(t, "topk("), map[string]string{"level": "WARN", "range": "1m"}))
	sort.Slice(top, func(i, j int) bool { return top[i].float(t) > top[j].float(t) })
	if len(top) == 0 {
		t.Fatal("no WARN messages")
	}
	m := regexp.MustCompile(`key (\S+)`).FindStringSubmatch(top[0].Metric["msg"])
	if m == nil {
		t.Fatalf("most frequent WARN %q names no key", top[0].Metric["msg"])
	}
	answers[6] = m[1]

	for n := 1; n <= 6; n++ {
		if answers[n] == "" {
			t.Fatalf("no answer found for question %d", n)
		}
		out, code := unlockSh(t, n, answers[n])
		if code != 0 {
			t.Fatalf("unlock.sh %d: exit %d, %q", n, code, out)
		}
		want := "\n## " + strconv.Itoa(n+1) + ". "
		if n == 6 {
			want = "That's all six."
		}
		if !strings.Contains(out, want) {
			t.Fatalf("unlock.sh %d printed %q, want it to contain %q", n, out, want)
		}
		if n == 4 && !strings.Contains(out, "./incident.sh C") {
			t.Fatalf("unlock.sh 4 printed %q, want question 5 to run mode C", out)
		}
	}
}
