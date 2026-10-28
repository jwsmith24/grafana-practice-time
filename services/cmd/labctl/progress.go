package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// progress records the highest scenario a pair has unlocked, in a file next
// to the lab's mode state, so resetting the lab's volumes resets it too.
// Scenario 1 is always unlocked; one past the last scenario means the pair
// has solved them all.
type progress struct{ path string }

// progressFor returns the progress kept beside the mode state file.
func progressFor(stateFile string) progress {
	return progress{path: filepath.Join(filepath.Dir(stateFile), "progress.json")}
}

type progressState struct {
	Unlocked int `json:"unlocked"`
}

// Unlocked returns the highest scenario unlocked so far, 1 if none was.
func (p progress) Unlocked() (int, error) {
	b, err := os.ReadFile(p.path)
	if errors.Is(err, fs.ErrNotExist) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	var st progressState
	if err := json.Unmarshal(b, &st); err != nil {
		return 0, err
	}
	return max(st.Unlocked, 1), nil
}

// Record notes that scenario n is unlocked. Progress only goes up.
func (p progress) Record(n int) error {
	cur, err := p.Unlocked()
	if err != nil {
		return err
	}
	if n <= cur {
		return nil
	}
	b, err := json.Marshal(progressState{Unlocked: n})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}
