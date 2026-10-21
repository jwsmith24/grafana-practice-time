package incident

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestSwitch(t *testing.T, d time.Duration) (*Switch, *fakeClock) {
	t.Helper()
	clock := &fakeClock{t: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)}
	s := New(filepath.Join(t.TempDir(), "state.json"), d)
	s.now = clock.now
	return s, clock
}

func status(t *testing.T, s *Switch) string {
	t.Helper()
	l, _, err := s.Status()
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// T4.1-1
func TestSetAndStatus(t *testing.T) {
	s, _ := newTestSwitch(t, 0)
	if got := status(t, s); got != "" {
		t.Fatalf("fresh status %q, want none", got)
	}
	for _, l := range Letters {
		if err := s.Set(l); err != nil {
			t.Fatal(err)
		}
		if got := status(t, s); got != l {
			t.Fatalf("status %q after Set(%s)", got, l)
		}
		if got := s.Active(); got != l {
			t.Fatalf("Active %q after Set(%s)", got, l)
		}
		s.checked = time.Time{} // let Active re-read on the next call
	}
	if err := s.Set("X"); err == nil {
		t.Fatal("Set(X) succeeded")
	}
}

// T4.1-2
func TestAutoClear(t *testing.T) {
	if DefaultDuration != 3*time.Minute {
		t.Fatalf("default duration %s, want 3m", DefaultDuration)
	}
	t.Setenv("INCIDENT_DURATION", "")
	if d, err := DurationFromEnv(); err != nil || d != 3*time.Minute {
		t.Fatalf("DurationFromEnv() = %s, %v; want 3m", d, err)
	}
	if New("x", 0).Duration() != 3*time.Minute {
		t.Fatal("New with zero duration does not default to 3m")
	}

	s, clock := newTestSwitch(t, 0)
	if err := s.Set("B"); err != nil {
		t.Fatal(err)
	}
	clock.advance(2*time.Minute + 59*time.Second)
	if got := status(t, s); got != "B" {
		t.Fatalf("at 2:59 status %q, want B", got)
	}
	s.checked = time.Time{}
	if got := s.Active(); got != "B" {
		t.Fatalf("at 2:59 Active %q, want B", got)
	}
	clock.advance(time.Second)
	if got := status(t, s); got != "" {
		t.Fatalf("at 3:00 status %q, want cleared", got)
	}
	if got := s.Active(); got != "" {
		t.Fatalf("at 3:00 Active %q, want cleared", got)
	}
}

func TestDurationOverride(t *testing.T) {
	t.Setenv("INCIDENT_DURATION", "15s")
	if d, err := DurationFromEnv(); err != nil || d != 15*time.Second {
		t.Fatalf("DurationFromEnv() = %s, %v; want 15s", d, err)
	}
	t.Setenv("INCIDENT_DURATION", "soon")
	if _, err := DurationFromEnv(); err == nil {
		t.Fatal("accepted a bad duration")
	}
}

// T4.1-3
func TestClearAndRetrigger(t *testing.T) {
	s, clock := newTestSwitch(t, 0)
	_ = s.Set("A")
	clock.advance(time.Minute)
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if got := status(t, s); got != "" {
		t.Fatalf("status %q after Clear", got)
	}

	_ = s.Set("A")
	clock.advance(2 * time.Minute)
	_ = s.Set("A") // restarts the 3m timer
	clock.advance(2 * time.Minute)
	if got := status(t, s); got != "A" {
		t.Fatalf("4m after first Set, 2m after re-trigger: status %q, want A", got)
	}
	clock.advance(time.Minute)
	if got := status(t, s); got != "" {
		t.Fatalf("3m after re-trigger: status %q, want cleared", got)
	}
}

// T4.1-7
func TestSwitchingIsSilent(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(prev)

	out := captureStdio(t, func() {
		s, clock := newTestSwitch(t, 0)
		for _, l := range Letters {
			_ = s.Set(l)
			s.Active()
			_, _, _ = s.Status()
		}
		_ = s.Clear()
		clock.advance(time.Hour)
		s.Active()
	})
	if all := logs.String() + out; strings.Contains(strings.ToLower(all), "incident") || all != "" {
		t.Fatalf("switching produced output: %q", all)
	}
}

// captureStdio returns everything written to os.Stdout and os.Stderr by f.
func captureStdio(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	f()
	os.Stdout, os.Stderr = stdout, stderr
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}
