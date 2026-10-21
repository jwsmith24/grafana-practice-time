//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func ordersURL() string { return envOr("ORDERS_URL", "http://localhost:8080") }

func grafanaURL() string {
	return envOr("GRAFANA_URL", "http://localhost:"+envOr("GRAFANA_PORT", "3300"))
}

// do issues a request to Grafana and returns the status code and body.
// With admin set, it uses the admin basic-auth credentials; otherwise it is anonymous.
func do(t *testing.T, method, path string, body any, admin bool) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, grafanaURL()+path, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if admin {
		req.SetBasicAuth(envOr("GRAFANA_ADMIN_USER", "admin"), envOr("GRAFANA_ADMIN_PASSWORD", "admin"))
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}

func decode(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode %q: %v", b, err)
	}
}

const lokiAPI = "/api/datasources/proxy/uid/loki/loki/api/v1"

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

// lokiQuery runs a LogQL range query over the last window through Grafana's
// datasource proxy and returns the streams.
func lokiQuery(t *testing.T, query string, window time.Duration) []lokiStream {
	t.Helper()
	now := time.Now()
	v := url.Values{
		"query":     {query},
		"start":     {strconv.FormatInt(now.Add(-window).UnixNano(), 10)},
		"end":       {strconv.FormatInt(now.UnixNano(), 10)},
		"limit":     {"5000"},
		"direction": {"backward"},
	}
	code, b := do(t, "GET", lokiAPI+"/query_range?"+v.Encode(), nil, false)
	if code != 200 {
		t.Fatalf("query %q: status %d: %s", query, code, b)
	}
	var r struct {
		Data struct {
			Result []lokiStream `json:"result"`
		} `json:"data"`
	}
	decode(t, b, &r)
	return r.Data.Result
}

func countLines(streams []lokiStream) int {
	n := 0
	for _, s := range streams {
		n += len(s.Values)
	}
	return n
}

// lokiGet fetches a Loki API path (e.g. "/labels") and returns its data array.
func lokiGet(t *testing.T, path string) []string {
	t.Helper()
	code, b := do(t, "GET", lokiAPI+path, nil, false)
	if code != 200 {
		t.Fatalf("GET %s: status %d: %s", path, code, b)
	}
	var r struct {
		Data []string `json:"data"`
	}
	decode(t, b, &r)
	sort.Strings(r.Data)
	return r.Data
}

// eventually retries f every second until it returns true or the timeout passes.
func eventually(t *testing.T, timeout time.Duration, f func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if f() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Second)
	}
}

// lokiInstant runs an instant LogQL metric query that returns one series and
// returns its value.
func lokiInstant(t *testing.T, query string) float64 {
	t.Helper()
	v := url.Values{"query": {query}, "time": {strconv.FormatInt(time.Now().UnixNano(), 10)}}
	code, b := do(t, "GET", lokiAPI+"/query?"+v.Encode(), nil, false)
	if code != 200 {
		t.Fatalf("query %q: status %d: %s", query, code, b)
	}
	var r struct {
		Data struct {
			Result []struct {
				Value [2]any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	decode(t, b, &r)
	if len(r.Data.Result) != 1 {
		t.Fatalf("query %q: want 1 series, got %d: %s", query, len(r.Data.Result), b)
	}
	s, _ := r.Data.Result[0].Value[1].(string)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("query %q: value %v: %v", query, r.Data.Result[0].Value, err)
	}
	return f
}
