package static

import (
	"regexp"
	"strings"
	"testing"
)

const cheatSheetPath = "../../materials/logql-cheat-sheet.md"

type fence struct {
	lang, body string
}

var fenceRe = regexp.MustCompile("(?ms)^```([a-z]*)\n(.*?)^```$")

func fences(t *testing.T, path string) []fence {
	t.Helper()
	var out []fence
	for _, m := range fenceRe.FindAllStringSubmatch(readFile(t, path), -1) {
		out = append(out, fence{m[1], strings.TrimSpace(m[2])})
	}
	return out
}

// T6.1-1
func TestCheatSheetSize(t *testing.T) {
	blocks := fences(t, cheatSheetPath)
	if n := len(blocks); n < 8 || n > 10 {
		t.Errorf("%d query blocks, want 8-10", n)
	}
	for _, b := range blocks {
		if strings.Contains(b.body, "\n") {
			t.Errorf("query spans more than one line: %q", b.body)
		}
	}
	if n := strings.Count(strings.TrimRight(readFile(t, cheatSheetPath), "\n"), "\n") + 1; n > 60 {
		t.Errorf("%d lines, want at most 60 to fit one printed page", n)
	}
}

// T6.1-2
func TestCheatSheetTechniques(t *testing.T) {
	techniques := map[string]string{
		"label selector":                `^\{service="<service>", level="<level>"\}$`,
		"line filters |= and !=":        `\|= "[^"]+" != "[^"]+"`,
		"regex |~":                      `\|~ "[^"]+"`,
		"| json + field filter":         `\| json \| <field>="<value>"`,
		"| json + numeric filter":       `\| json \| [a-z_]+ > <[a-z_]+>`,
		"rate()":                        `^sum by \([a-z_]+\) \(rate\(\{.*\}\[[0-9a-z<>_]+\]\)\)$`,
		"count_over_time() with sum by": `^sum by \(<field>\) \(count_over_time\(\{.*\} \| json \[<range>\]\)\)$`,
		"filter out the red herring":    `^\{level="WARN"\} != "deprecated"$`,
		"follow a request_id":           `^\{env="lab"\} \| json \| request_id="<request_id>"$`,
	}
	blocks := fences(t, cheatSheetPath)
	for name, src := range techniques {
		re := regexp.MustCompile(src)
		found := false
		for _, b := range blocks {
			found = found || re.MatchString(b.body)
		}
		if !found {
			t.Errorf("no query covers %s (%s)", name, src)
		}
	}
}

// placeholderNames are the placeholders the integration tests know how to fill.
var placeholderNames = map[string]bool{
	"service": true, "level": true, "text": true, "other_text": true, "pattern": true,
	"field": true, "value": true, "ms": true, "request_id": true, "range": true,
}

// T6.1-6
func TestCheatSheetIsLogQL(t *testing.T) {
	selector := regexp.MustCompile(`\{[a-z_]+=~?"[^"]*"(, [a-z_]+=~?"[^"]*")*\}`)
	metricName := regexp.MustCompile(`[A-Za-z_:]\{`)
	placeholder := regexp.MustCompile(`<([^<>\s]*)>`)
	for _, b := range fences(t, cheatSheetPath) {
		if b.lang != "logql" {
			t.Errorf("block tagged %q, want logql: %s", b.lang, b.body)
		}
		if !selector.MatchString(b.body) {
			t.Errorf("no LogQL stream selector in %s", b.body)
		}
		if metricName.MatchString(b.body) {
			t.Errorf("metric-name selector (PromQL) in %s", b.body)
		}
		for _, m := range placeholder.FindAllStringSubmatch(b.body, -1) {
			if !placeholderNames[m[1]] {
				t.Errorf("unknown placeholder <%s> in %s", m[1], b.body)
			}
		}
	}
}
