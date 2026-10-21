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
func TestLinesInLoki(t *testing.T) {
	waitForLines(t, "orders")
}

// T1.2-2
func TestStreamLabels(t *testing.T) {
	waitForLines(t, "orders")
	if got, want := lokiGet(t, "/labels"), []string{"env", "level", "service"}; !slices.Equal(got, want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	if got, want := lokiGet(t, "/label/env/values"), []string{"lab"}; !slices.Equal(got, want) {
		t.Fatalf("env values = %v, want %v", got, want)
	}
}

// T1.2-3: every entry's timestamp is exactly its line's millisecond-precision
// time field. Ingest timestamps would not land on whole milliseconds.
func TestTimestampFromLine(t *testing.T) {
	waitForLines(t, "orders")
	streams := lokiQuery(t, `{env="lab"}`, 5*time.Minute)
	checked := 0
	for _, s := range streams {
		for _, v := range s.Values {
			var line struct {
				Time string `json:"time"`
			}
			if err := json.Unmarshal([]byte(v[1]), &line); err != nil {
				t.Fatalf("line %q: %v", v[1], err)
			}
			want, err := time.Parse(time.RFC3339Nano, line.Time)
			if err != nil {
				t.Fatal(err)
			}
			ns, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			if got := time.Unix(0, ns); !got.Equal(want) {
				t.Fatalf("entry timestamp %s != line time %s\nline: %s", got.UTC().Format(time.RFC3339Nano), line.Time, v[1])
			}
			checked++
		}
	}
	if checked < 10 {
		t.Fatalf("only %d entries checked", checked)
	}
}

// T1.2-4
func TestLinesParseAsJSON(t *testing.T) {
	waitForLines(t, "orders")
	if n := countLines(lokiQuery(t, `{env="lab"} | json | __error__!=""`, 15*time.Minute)); n != 0 {
		t.Fatalf("%d lines failed | json", n)
	}
	if n := countLines(lokiQuery(t, `{service="orders"} | json | request_id!=""`, 15*time.Minute)); n == 0 {
		t.Fatal("no lines with a parsed request_id")
	}
	if labels := lokiGet(t, "/labels"); slices.Contains(labels, "request_id") {
		t.Fatalf("request_id is a stream label: %v", labels)
	}
}

func waitForLines(t *testing.T, service string) {
	t.Helper()
	if !eventually(t, 30*time.Second, func() bool {
		return countLines(lokiQuery(t, `{service="`+service+`"}`, 5*time.Minute)) > 0
	}) {
		t.Fatalf(`{service=%q} still empty after 30s`, service)
	}
}
