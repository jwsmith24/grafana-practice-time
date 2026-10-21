// Package logschema validates lab log lines against the spec's field schema.
// Tests in each service use it so every service logs the same shape.
package logschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
)

// Kind selects which fields a line must carry beyond the base set.
type Kind int

const (
	// Base lines carry time, level, service and msg.
	Base Kind = iota
	// Request lines also carry request_id, endpoint, status and duration_ms.
	Request
	// Startup lines also carry event="startup" and version.
	Startup
)

var (
	timeRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)
	uuidRe   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	levels   = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	required = map[Kind][]string{
		Base:    {"time", "level", "service", "msg"},
		Request: {"time", "level", "service", "msg", "request_id", "endpoint", "status", "duration_ms"},
		Startup: {"time", "level", "service", "msg", "event", "version"},
	}
)

// Validate checks that line is a single JSON object of the given kind with
// strictly typed fields. Extra fields are allowed but must still use the
// known types when they share a name (item_id, error).
func Validate(line []byte, kind Kind) error {
	line = bytes.TrimSuffix(line, []byte("\n"))
	if bytes.Contains(line, []byte("\n")) {
		return fmt.Errorf("line contains a raw newline")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(line, &m); err != nil {
		return fmt.Errorf("not a JSON object: %w", err)
	}
	for _, k := range required[kind] {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("missing %q", k)
		}
	}
	for k, raw := range m {
		if err := checkField(k, raw); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
	}
	if kind == Startup {
		var ev string
		_ = json.Unmarshal(m["event"], &ev)
		if ev != "startup" {
			return fmt.Errorf(`event = %q, want "startup"`, ev)
		}
	}
	return nil
}

func checkField(k string, raw json.RawMessage) error {
	switch k {
	case "time":
		s, err := str(raw)
		if err == nil && !timeRe.MatchString(s) {
			err = fmt.Errorf("%q is not RFC 3339 UTC with milliseconds", s)
		}
		return err
	case "level":
		s, err := str(raw)
		if err == nil && !levels[s] {
			err = fmt.Errorf("unknown level %q", s)
		}
		return err
	case "request_id":
		s, err := str(raw)
		if err == nil && !uuidRe.MatchString(s) {
			err = fmt.Errorf("%q is not a UUID v4", s)
		}
		return err
	case "service", "msg", "endpoint", "item_id", "event", "version", "error":
		_, err := str(raw)
		return err
	case "status":
		var n int
		if err := json.Unmarshal(raw, &n); err != nil {
			return fmt.Errorf("want integer, got %s", raw)
		}
		return nil
	case "duration_ms":
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			return fmt.Errorf("want number, got %s", raw)
		}
		if f < 0 {
			return fmt.Errorf("negative duration %v", f)
		}
		return nil
	}
	return nil
}

func str(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("want string, got %s", raw)
	}
	return s, nil
}
