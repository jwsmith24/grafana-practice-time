package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logschema"
	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

type recorder struct {
	mu   sync.Mutex
	reqs []*http.Request
	ids  []string
}

func (rc *recorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc.mu.Lock()
	rc.reqs = append(rc.reqs, r)
	rc.ids = append(rc.ids, r.Header.Get(requestid.Header))
	n := len(rc.reqs)
	rc.mu.Unlock()
	if r.Method == "POST" {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": fmt.Sprintf("ORD-%06d", n)})
		return
	}
	w.WriteHeader(http.StatusOK)
}

// T2.3-1
func TestRate(t *testing.T) {
	rc := &recorder{}
	srv := httptest.NewServer(rc)
	defer srv.Close()

	g := newGenerator(srv.URL, 5, slog.New(slog.DiscardHandler))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	g.run(ctx)

	rc.mu.Lock()
	defer rc.mu.Unlock()
	if n := len(rc.reqs); n < 16 || n > 24 {
		t.Fatalf("%d requests in 4s, want 16-24", n)
	}
	var posts, gets int
	for _, r := range rc.reqs {
		switch {
		case r.Method == "POST" && r.URL.Path == "/orders":
			posts++
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/orders/"):
			gets++
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}
	if posts <= gets {
		t.Errorf("want mostly POSTs, got %d POST / %d GET", posts, gets)
	}
	seen := map[string]bool{}
	for _, id := range rc.ids {
		if seen[id] {
			t.Errorf("duplicate X-Request-ID %s", id)
		}
		seen[id] = true
	}
}

// T2.3-3
func TestItems(t *testing.T) {
	g := newGenerator("http://unused", 5, slog.New(slog.DiscardHandler))
	if len(g.items) != 50 {
		t.Fatalf("%d items, want 50", len(g.items))
	}
	re := regexp.MustCompile(`^SKU-\d{4}$`)
	for _, id := range g.items {
		if !re.MatchString(id) {
			t.Errorf("bad item id %q", id)
		}
	}
	if !slices.Contains(g.items, "SKU-1042") {
		t.Fatal("SKU-1042 missing")
	}
}

func TestSummaryLine(t *testing.T) {
	var buf bytes.Buffer
	srv := httptest.NewServer(&recorder{})
	defer srv.Close()
	g := newGenerator(srv.URL, 20, logging.NewWithWriter(service, &buf))
	g.summary = 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	g.run(ctx)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("no summary lines")
	}
	for _, l := range lines {
		if err := logschema.Validate([]byte(l), logschema.Base); err != nil {
			t.Fatalf("%v: %s", err, l)
		}
	}
}

func TestPickItem(t *testing.T) {
	g := newGenerator("http://unused", 1, nil)
	const n = 50000
	counts := map[string]int{}
	for i := 0; i < n; i++ {
		counts[g.pickItem()]++
	}
	if len(counts) != len(g.items) {
		t.Fatalf("picked %d distinct items, want all %d", len(counts), len(g.items))
	}
	// Each bestseller: 0.2/3 + 0.8/50 ≈ 8.3% of orders; the rest 1.6%.
	for _, id := range bestsellers {
		if share := float64(counts[id]) / n; share < 0.07 || share > 0.10 {
			t.Errorf("%s share %.3f, want about 0.083", id, share)
		}
	}
	if share := float64(counts["SKU-1001"]) / n; share < 0.01 || share > 0.025 {
		t.Errorf("SKU-1001 share %.3f, want about 0.016", share)
	}
}
