//go:build lifecycle

package lifecycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

const root = "../.."

// run runs a command in the repo root with extra environment variables and
// returns its combined output and exit code.
func run(t *testing.T, env []string, name string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("%s %v: %v", name, args, err)
		}
		code = ee.ExitCode()
	}
	return out.String(), code
}

// must runs a command and fails the test if it exits non-zero.
func must(t *testing.T, env []string, name string, args ...string) string {
	t.Helper()
	out, code := run(t, env, name, args...)
	if code != 0 {
		t.Fatalf("%s %s: exit %d\n%s", name, strings.Join(args, " "), code, out)
	}
	return out
}

func compose(t *testing.T, args ...string) string {
	t.Helper()
	return must(t, nil, "docker", append([]string{"compose"}, args...)...)
}

// freshStack resets the lab the way the README tells attendees to, and
// waits until every container is healthy.
func freshStack(t *testing.T) {
	t.Helper()
	compose(t, "down", "-v")
	compose(t, "up", "-d", "--wait")
}

// checkSh runs ./check.sh and returns its output, exit code and duration.
func checkSh(t *testing.T, env ...string) (string, int, time.Duration) {
	t.Helper()
	start := time.Now()
	out, code := run(t, env, "./check.sh")
	return out, code, time.Since(start)
}

// eventually retries f every interval until it returns true or timeout
// passes.
func eventually(timeout, interval time.Duration, f func() bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if f() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(interval)
	}
}

var client = &http.Client{Timeout: 5 * time.Second}

// grafanaGet fetches a path from Grafana on port and returns status and body.
func grafanaGet(port int, path string) (int, string, error) {
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d%s", port, path))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	return resp.StatusCode, b.String(), nil
}

// proxyGet fetches a datasource proxy path through Grafana on 3300 and
// decodes the JSON response into v.
func proxyGet(t *testing.T, uid, path string, q url.Values, v any) {
	t.Helper()
	u := "/api/datasources/proxy/uid/" + uid + path
	if q != nil {
		u += "?" + q.Encode()
	}
	code, body, err := grafanaGet(3300, u)
	if err != nil || code != 200 {
		t.Fatalf("GET %s: %d %v %s", u, code, err, body)
	}
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
}

// lokiLines returns the raw lines matching a LogQL query in [start, end].
func lokiLines(t *testing.T, query string, start, end time.Time) []string {
	t.Helper()
	var r struct {
		Data struct {
			Result []struct {
				Values [][2]string `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	proxyGet(t, "loki", "/loki/api/v1/query_range", url.Values{
		"query": {query},
		"start": {strconv.FormatInt(start.UnixNano(), 10)},
		"end":   {strconv.FormatInt(end.UnixNano(), 10)},
		"limit": {"5000"},
	}, &r)
	var out []string
	for _, s := range r.Data.Result {
		for _, v := range s.Values {
			out = append(out, v[1])
		}
	}
	return out
}
