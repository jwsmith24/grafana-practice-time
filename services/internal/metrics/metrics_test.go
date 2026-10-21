package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestInit(t *testing.T) {
	m := NewHTTP("svc")
	m.Init("GET /x", 500)
	m.Observe("GET /x", 500, time.Millisecond)
	m.Init("GET /x", 500) // must not reset the count
	want := `
# HELP http_requests_total HTTP requests served, by route template and status code.
# TYPE http_requests_total counter
http_requests_total{endpoint="GET /x",service="svc",status="500"} 1
`
	if err := testutil.GatherAndCompare(m.Registry, strings.NewReader(want), "http_requests_total"); err != nil {
		t.Fatal(err)
	}
}
