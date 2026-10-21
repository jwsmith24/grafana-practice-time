package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
)

// T2.1-3
func TestNewTruncatesAndWritesBoth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svc.log")
	if err := os.WriteFile(path, []byte("old content from last run\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	logger, closer, err := New("svc", dir, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello")
	closer.Close()

	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(file), "old content") {
		t.Fatalf("log file was not truncated: %q", file)
	}
	if !bytes.Equal(file, stdout.Bytes()) {
		t.Fatalf("file and stdout differ:\nfile:   %q\nstdout: %q", file, stdout.Bytes())
	}
	if err := logschema.Validate(file, logschema.Base); err != nil {
		t.Fatal(err)
	}
}

// T2.1-4
func TestMultilineErrorStaysOneLine(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("svc", &buf)
	trace := "boom\ngoroutine 1 [running]:\nmain.main()\n\t/app/main.go:12 +0x1d"
	logger.Error("failed", "error", errors.New(trace).Error())

	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.HasSuffix(out, "\n") {
		t.Fatalf("want exactly one trailing newline, got %q", out)
	}
	var line struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line.Error != trace {
		t.Fatalf("error did not round-trip:\ngot  %q\nwant %q", line.Error, trace)
	}
}

func TestLevelsAndTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("svc", &buf)
	logger.Debug("d")
	logger.Info("i")
	logger.Warn("w")
	logger.Error("e")
	want := []string{"debug", "info", "warn", "error"}
	for i, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Errorf("line %d: %v", i, err)
		}
		var v struct{ Level, Service string }
		_ = json.Unmarshal([]byte(l), &v)
		if v.Level != want[i] || v.Service != "svc" {
			t.Errorf("line %d: level=%q service=%q", i, v.Level, v.Service)
		}
	}
}
