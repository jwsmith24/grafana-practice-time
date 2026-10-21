package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runLabctl(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestLabctl(t *testing.T) {
	t.Setenv("LAB_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("INCIDENT_DURATION", "")

	if code, out, _ := runLabctl(t, "status"); code != 0 || out != "none\n" {
		t.Fatalf("status: %d %q", code, out)
	}
	outputs := map[string]string{}
	for _, l := range []string{"A", "b", "C"} {
		code, out, _ := runLabctl(t, l)
		if code != 0 {
			t.Fatalf("%s: exit %d", l, code)
		}
		outputs[strings.ToUpper(l)] = out
	}
	// The outputs differ only in the letter.
	norm := func(s, l string) string { return strings.Replace(s, l, "?", 1) }
	if norm(outputs["A"], "A") != norm(outputs["B"], "B") || norm(outputs["A"], "A") != norm(outputs["C"], "C") {
		t.Fatalf("outputs differ beyond the letter: %q", outputs)
	}
	if code, out, _ := runLabctl(t, "status"); code != 0 || !strings.HasPrefix(out, "active: C") {
		t.Fatalf("status after C: %d %q", code, out)
	}
	if code, out, _ := runLabctl(t, "clear"); code != 0 || out != "cleared\n" {
		t.Fatalf("clear: %d %q", code, out)
	}
	if code, _, errOut := runLabctl(t, "X"); code == 0 || !strings.Contains(errOut, "usage") {
		t.Fatalf("X: exit %d, stderr %q", code, errOut)
	}
	if code, _, _ := runLabctl(t); code == 0 {
		t.Fatal("no args: exit 0")
	}
	for l, out := range outputs {
		if strings.Contains(strings.ToLower(out), "incident") {
			t.Fatalf("%s output mentions incident: %q", l, out)
		}
	}
}
