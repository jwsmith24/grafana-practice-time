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

// unlockNext runs ./unlock.sh n answer and checks that it printed scenario
// n+1, or the closing message after scenario 6.
func unlockNext(t *testing.T, n int, answer string) string {
	t.Helper()
	out, code := unlockSh(t, n, answer)
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
	return out
}

// wantStatus checks what ./incident.sh status prints.
func wantStatus(t *testing.T, prefix string) {
	t.Helper()
	if s := incidentSh(t, "status"); !strings.HasPrefix(s, prefix) {
		t.Fatalf("incident.sh status: %q, want %q", s, prefix)
	}
}

// T6.5-4: the gated worksheet end to end. Each answer comes from the lab,
// found the way the pair finds it, and unlocks the next scenario, which
// starts the mode it needs; the last answers put the lab back to normal and
// print the closing message. Only scenario 1's field and scenario 3's
// service are written down here.
func TestGatedWorksheet(t *testing.T) {
	if out, code := unlockSh(t, 1, "not it"); code != 1 || !strings.Contains(out, "Not quite") {
		t.Fatalf("wrong answer: exit %d, %q", code, out)
	}

	// 1, then 2: the service the followed request reached second, after the
	// one that logged the POST /orders request line.
	unlockNext(t, 1, "request_id")
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

	// Answering 2 starts mode B. Its answer, the service, is written down below.
	t.Run("B", func(t *testing.T) {
		triggerWith(t, "B", func() { unlockNext(t, 2, others[0]) })
		wantStatus(t, "active: B")
	})

	// Answering 3 starts mode C (the wait for a quiet lab clears B first, so
	// C's chain sees only its own errors).
	var version string
	t.Run("C", func(t *testing.T) {
		start := triggerWith(t, "C", func() { unlockNext(t, 3, "inventory") })
		wantStatus(t, "active: C")
		version = chainC(t, start)
	})
	if t.Failed() {
		t.FailNow()
	}

	// Answering 4 starts mode A, after the same wait; answering 5 clears it.
	t.Run("A", func(t *testing.T) {
		start := triggerWith(t, "A", func() { unlockNext(t, 4, version) })
		wantStatus(t, "active: A")
		unlockNext(t, 5, chainA(t, start))
		wantStatus(t, "none")
	})
	if t.Failed() {
		t.FailNow()
	}

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
	unlockNext(t, 6, m[1])
	wantStatus(t, "none")
}
