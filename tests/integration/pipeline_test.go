//go:build integration

package integration

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"
)

// T1.2-1
func TestDummyLinesInLoki(t *testing.T) {
	ok := eventually(t, 30*time.Second, func() bool {
		return countLines(lokiQuery(t, `{service="dummy"}`, 5*time.Minute)) > 0
	})
	if !ok {
		t.Fatal(`{service="dummy"} still empty after 30s`)
	}
}

// T1.2-2
func TestStreamLabels(t *testing.T) {
	waitForDummy(t)
	if got, want := lokiGet(t, "/labels"), []string{"env", "level", "service"}; !slices.Equal(got, want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	if got, want := lokiGet(t, "/label/env/values"), []string{"lab"}; !slices.Equal(got, want) {
		t.Fatalf("env values = %v, want %v", got, want)
	}
}

// T1.2-3
func TestTimestampFromLine(t *testing.T) {
	waitForDummy(t)
	var streams []lokiStream
	ok := eventually(t, 30*time.Second, func() bool {
		streams = lokiQuery(t, `{service="dummy"} |= "\"msg\":\"marker\""`, 15*time.Minute)
		return countLines(streams) > 0
	})
	if !ok {
		t.Fatal("marker line not found")
	}
	entry := streams[0].Values[0]
	var line struct {
		Time string `json:"time"`
	}
	if err := json.Unmarshal([]byte(entry[1]), &line); err != nil {
		t.Fatalf("marker line %q: %v", entry[1], err)
	}
	want, err := time.Parse(time.RFC3339Nano, line.Time)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := strconv.ParseInt(entry[0], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	got := time.Unix(0, ns)
	if d := got.Sub(want).Abs(); d > time.Millisecond {
		t.Fatalf("entry timestamp %s, line time %s (off by %s)", got, want, d)
	}
	if age := time.Since(got); age < 60*time.Second {
		t.Fatalf("marker entry is only %s old; want the backdated line time, not ingest time", age)
	}
}

// T1.2-4
func TestLinesParseAsJSON(t *testing.T) {
	waitForDummy(t)
	if n := countLines(lokiQuery(t, `{service="dummy"} | json | __error__!=""`, 15*time.Minute)); n != 0 {
		t.Fatalf("%d lines failed | json", n)
	}
	if n := countLines(lokiQuery(t, `{service="dummy"} | json | request_id!=""`, 15*time.Minute)); n == 0 {
		t.Fatal("no lines with a parsed request_id")
	}
	if labels := lokiGet(t, "/labels"); slices.Contains(labels, "request_id") {
		t.Fatalf("request_id is a stream label: %v", labels)
	}
}

func waitForDummy(t *testing.T) {
	t.Helper()
	if !eventually(t, 30*time.Second, func() bool {
		return countLines(lokiQuery(t, `{service="dummy"}`, 5*time.Minute)) > 0
	}) {
		t.Fatal("no dummy lines in Loki")
	}
}
