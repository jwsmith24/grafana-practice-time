// Package incident holds the lab's active mode letter and its expiry.
//
// The state lives in a small JSON file on a volume shared by the services,
// so one write (by labctl) reaches all of them. A mode expires on its own
// after the configured duration; Status computes that from the stored
// expiry, so no timers are involved.
package incident

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultDuration is how long a mode stays active unless overridden.
const DefaultDuration = 3 * time.Minute

// Letters are the valid mode letters.
var Letters = []string{"A", "B", "C"}

// ErrInvalid is returned for a letter that isn't one of Letters.
var ErrInvalid = errors.New("invalid letter")

type state struct {
	Letter  string    `json:"letter"`
	Expires time.Time `json:"expires"`
}

// Switch reads and writes the shared state file.
type Switch struct {
	path     string
	duration time.Duration
	now      func() time.Time

	mu      sync.Mutex
	cached  state
	checked time.Time
}

// New returns a Switch backed by path. A zero duration means DefaultDuration.
func New(path string, duration time.Duration) *Switch {
	if duration <= 0 {
		duration = DefaultDuration
	}
	return &Switch{path: path, duration: duration, now: time.Now}
}

// DurationFromEnv parses INCIDENT_DURATION, falling back to DefaultDuration.
func DurationFromEnv() (time.Duration, error) {
	v := os.Getenv("INCIDENT_DURATION")
	if v == "" {
		return DefaultDuration, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("INCIDENT_DURATION %q: want a positive duration like 3m", v)
	}
	return d, nil
}

// Duration is how long Set keeps a letter active.
func (s *Switch) Duration() time.Duration { return s.duration }

// Set activates letter for the configured duration, restarting the timer if
// a mode is already active.
func (s *Switch) Set(letter string) error {
	if !valid(letter) {
		return fmt.Errorf("%w %q", ErrInvalid, letter)
	}
	return s.write(state{Letter: letter, Expires: s.now().Add(s.duration)})
}

// Clear ends any active mode.
func (s *Switch) Clear() error { return s.write(state{}) }

// Status returns the active letter and the time left, or "" when none is
// active.
func (s *Switch) Status() (string, time.Duration, error) {
	st, err := s.read()
	if err != nil {
		return "", 0, err
	}
	left := st.Expires.Sub(s.now())
	if st.Letter == "" || left <= 0 {
		return "", 0, nil
	}
	return st.Letter, left, nil
}

// Active returns the active letter, re-reading the file at most once a
// second. Errors read as "no active mode" so a missing or damaged file never
// breaks request handling.
func (s *Switch) Active() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if now.Sub(s.checked) >= time.Second {
		if st, err := s.read(); err == nil {
			s.cached = st
		} else {
			s.cached = state{}
		}
		s.checked = now
	}
	if s.cached.Letter == "" || !now.Before(s.cached.Expires) {
		return ""
	}
	return s.cached.Letter
}

func (s *Switch) read() (state, error) {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return state{}, nil
	}
	if err != nil {
		return state{}, err
	}
	var st state
	if err := json.Unmarshal(b, &st); err != nil {
		return state{}, err
	}
	return st, nil
}

func (s *Switch) write(st state) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func valid(letter string) bool {
	for _, l := range Letters {
		if l == letter {
			return true
		}
	}
	return false
}
