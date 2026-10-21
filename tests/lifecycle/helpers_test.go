//go:build lifecycle

package lifecycle

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
