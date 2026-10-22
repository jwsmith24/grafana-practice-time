//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	fenceRe       = regexp.MustCompile("(?ms)^```([a-z]*)\n(.*?)^```$")
	placeholderRe = regexp.MustCompile(`<([a-z_]+)>`)
)

// readMaterial returns the text of a file in materials/.
func readMaterial(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../materials/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// queryBlocks returns the fenced query blocks of a materials file, in order.
func queryBlocks(t *testing.T, name string) []string {
	t.Helper()
	var out []string
	for _, m := range fenceRe.FindAllStringSubmatch(readMaterial(t, name), -1) {
		out = append(out, strings.TrimSpace(m[2]))
	}
	if len(out) == 0 {
		t.Fatalf("%s has no query blocks", name)
	}
	return out
}

// template returns the cheat-sheet query that is marker, or else the one
// query that contains it.
func template(t *testing.T, marker string) string {
	t.Helper()
	var found []string
	for _, q := range queryBlocks(t, "logql-cheat-sheet.md") {
		if q == marker {
			return q
		}
		if strings.Contains(q, marker) {
			found = append(found, q)
		}
	}
	if len(found) != 1 {
		t.Fatalf("cheat-sheet queries containing %q: %q, want exactly one", marker, found)
	}
	return found[0]
}

// fill replaces every <placeholder> in q, which must all be in values.
func fill(t *testing.T, q string, values map[string]string) string {
	t.Helper()
	out := placeholderRe.ReplaceAllStringFunc(q, func(p string) string {
		v, ok := values[p[1:len(p)-1]]
		if !ok {
			t.Fatalf("no value for %s in %s", p, q)
		}
		return v
	})
	return out
}

// lokiRangeStatus runs any LogQL query through query_range and returns the
// status code and body.
func lokiRangeStatus(t *testing.T, query string, window time.Duration) (int, []byte) {
	t.Helper()
	now := time.Now()
	v := url.Values{
		"query": {query},
		"start": {strconv.FormatInt(now.Add(-window).UnixNano(), 10)},
		"end":   {strconv.FormatInt(now.UnixNano(), 10)},
		"limit": {"100"},
	}
	return do(t, "GET", lokiAPI+"/query_range?"+v.Encode(), nil, false)
}

// logLine is one parsed log line with the labels of its stream.
type logLine struct {
	ts     time.Time
	labels map[string]string
	fields map[string]any
}

func (l logLine) str(k string) string {
	s, _ := l.fields[k].(string)
	return s
}

// linesSince runs a log query from start until now and returns its lines,
// oldest first.
func linesSince(t *testing.T, query string, start time.Time) []logLine {
	t.Helper()
	var out []logLine
	for _, s := range lokiQueryRange(t, query, start, time.Now()) {
		for _, v := range s.Values {
			ns, _ := strconv.ParseInt(v[0], 10, 64)
			l := logLine{ts: time.Unix(0, ns), labels: s.Stream}
			if err := json.Unmarshal([]byte(v[1]), &l.fields); err != nil {
				t.Fatalf("line %q: %v", v[1], err)
			}
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts.Before(out[j].ts) })
	return out
}

// eventuallyLines retries linesSince until it returns a line or 20s pass.
func eventuallyLines(t *testing.T, query string, start time.Time) []logLine {
	t.Helper()
	var lines []logLine
	if !eventually(t, 20*time.Second, func() bool {
		lines = linesSince(t, query, start)
		return len(lines) > 0
	}) {
		t.Fatalf("no lines for %s", query)
	}
	t.Logf("%s: %d lines", query, len(lines))
	return lines
}

// rangeSince is a LogQL range covering start until now.
func rangeSince(start time.Time) string {
	return fmt.Sprintf("%ds", int(time.Since(start).Seconds())+5)
}

// recentRequestID returns the request_id of a recent POST /orders line.
func recentRequestID(t *testing.T) string {
	t.Helper()
	lines := eventuallyLines(t, `{service="orders", level="INFO"} | json | endpoint="POST /orders"`, time.Now().Add(-time.Minute))
	id := lines[len(lines)-1].str("request_id")
	if id == "" {
		t.Fatalf("POST /orders line has no request_id: %v", lines[len(lines)-1].fields)
	}
	return id
}

// T6.1-3
func TestCheatSheetQueriesRun(t *testing.T) {
	values := map[string]string{
		"service": "orders", "level": "INFO", "text": "request", "other_text": "deprecated",
		"pattern": "post /orders", "field": "endpoint", "value": "POST /orders", "ms": "1",
		"request_id": recentRequestID(t), "range": "5m",
	}
	for _, tmpl := range queryBlocks(t, "logql-cheat-sheet.md") {
		q := fill(t, tmpl, values)
		code, b := lokiRangeStatus(t, q, 5*time.Minute)
		if code != 200 {
			t.Errorf("%s: status %d: %s", q, code, b)
			continue
		}
		var r struct {
			Status string `json:"status"`
		}
		decode(t, b, &r)
		if r.Status != "success" {
			t.Errorf("%s: %s", q, b)
		}
	}
}

// triggerMode waits for a quiet lab, then starts mode letter and returns
// when it did. These tests clear the mode seconds after starting it, often
// before Prometheus has scraped any 5xx, so the next test's waitQuiet can't
// see the errors still in flight. On cleanup, wait until orders has logged no
// ERROR line for 10s instead.
func triggerMode(t *testing.T, letter string) time.Time {
	t.Helper()
	incidentSh(t, "clear")
	t.Cleanup(func() {
		incidentSh(t, "clear")
		if !eventually(t, 30*time.Second, func() bool {
			return len(lokiVector(t, `count_over_time({service="orders", level="ERROR"}[10s])`)) == 0
		}) {
			t.Errorf("orders still logging errors 30s after clearing mode %s", letter)
		}
	})
	waitQuiet(t)
	start := time.Now()
	incidentSh(t, letter)
	return start
}

// erroringServices runs the cheat sheet's rate() query for ERROR lines until
// some service shows up, and returns those services.
func erroringServices(t *testing.T) []string {
	t.Helper()
	q := fill(t, template(t, "rate("), map[string]string{"level": "ERROR"})
	var services []string
	if !eventually(t, 30*time.Second, func() bool {
		services = nil
		for _, s := range lokiVector(t, q) {
			if s.float(t) > 0 {
				services = append(services, s.Metric["service"])
			}
		}
		return len(services) > 0
	}) {
		t.Fatalf("%s: no service logs errors", q)
	}
	sort.Strings(services)
	t.Logf("%s: %v", q, services)
	return services
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// T6.1-4 (lab AC5): each mode's root cause, found by chaining cheat-sheet
// queries only, each filled in from the results before it. The queries cover
// the time since the switch, so they still work after a short mode clears.
func TestCheatSheetFindsA(t *testing.T) {
	start := triggerMode(t, "A")
	selector := template(t, `{service="<service>", level="<level>"}`)

	if s := erroringServices(t); !contains(s, "orders") {
		t.Fatalf("services logging errors: %v, want orders among them", s)
	}
	errs := eventuallyLines(t, fill(t, selector, map[string]string{"service": "orders", "level": "ERROR"}), start)
	// The lines carry the field to group by.
	const field = "item_id"
	if errs[0].str(field) == "" {
		t.Fatalf("orders ERROR line has no %s: %v", field, errs[0].fields)
	}
	q := fill(t, template(t, "sum by (<field>)"), map[string]string{
		"field": field, "service": "orders", "level": "ERROR", "range": rangeSince(start),
	})
	res := lokiVector(t, q)
	if len(res) != 1 || res[0].Metric[field] != "SKU-1042" {
		t.Fatalf("%s: %v, want one series with item_id SKU-1042", q, res)
	}
}

func TestCheatSheetFindsB(t *testing.T) {
	start := triggerMode(t, "B")
	selector := template(t, `{service="<service>", level="<level>"}`)

	if s := erroringServices(t); !contains(s, "orders") {
		t.Fatalf("services logging errors: %v, want orders among them", s)
	}
	errs := eventuallyLines(t, fill(t, selector, map[string]string{"service": "orders", "level": "ERROR"}), start)
	first := errs[0]
	if !strings.Contains(first.str("error"), "context deadline exceeded") || first.str("request_id") == "" {
		t.Fatalf("first orders ERROR line: %v, want a deadline error with a request_id", first.fields)
	}
	id := first.str("request_id")
	q := fill(t, template(t, "request_id="), map[string]string{"request_id": id})
	var inv []logLine
	if !eventually(t, 20*time.Second, func() bool {
		inv = nil
		for _, l := range linesSince(t, q, start) {
			if l.labels["service"] == "inventory" {
				inv = append(inv, l)
			}
		}
		return len(inv) > 0
	}) {
		t.Fatalf("%s: no inventory line", q)
	}
	if d, _ := inv[0].fields["duration_ms"].(float64); d <= 2000 {
		t.Fatalf("inventory line for %s: %v, want duration_ms > 2000", id, inv[0].fields)
	}
}

func TestCheatSheetFindsC(t *testing.T) {
	start := triggerMode(t, "C")
	selector := template(t, `{service="<service>", level="<level>"}`)

	if s := erroringServices(t); !contains(s, "orders") {
		t.Fatalf("services logging errors: %v, want orders among them", s)
	}
	errs := eventuallyLines(t, fill(t, selector, map[string]string{"service": "orders", "level": "ERROR"}), start)
	// The error names the service that turned the request down.
	if !strings.Contains(errs[0].str("error"), "inventory") {
		t.Fatalf("first orders ERROR line: %v, want an error naming inventory", errs[0].fields)
	}
	warns := eventuallyLines(t, template(t, `!= "deprecated"`), start)
	if warns[0].labels["service"] != "inventory" || warns[0].str("msg") != "auth token rejected" {
		t.Fatalf("first WARN after hiding the red herring: %v %v", warns[0].labels, warns[0].fields)
	}
	// The dashboard's Deploys annotation names the field and value to filter on.
	m := regexp.MustCompile(`\| json \| ([a-z_]+)="([^"]+)"`).FindStringSubmatch(deployAnnotation(t).Targets[0].Expr)
	if m == nil {
		t.Fatalf("Deploys annotation %q has no field filter", deployAnnotation(t).Targets[0].Expr)
	}
	q := fill(t, template(t, `| json | <field>="<value>"`), map[string]string{"service": "orders", "field": m[1], "value": m[2]})
	deploys := eventuallyLines(t, q, start.Add(-time.Second))
	if deploys[0].str("version") != "1.4.3" {
		t.Fatalf("first deploy line since the switch: %v, want version 1.4.3", deploys[0].fields)
	}
	if gap := warns[0].ts.Sub(deploys[0].ts); gap < 0 || gap > 5*time.Second {
		t.Fatalf("auth token rejected starts %s after the 1.4.3 deploy, want 0-5s", gap)
	}
}
