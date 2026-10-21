package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/catalog"
	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

// generator sends a steady stream of orders traffic to target.
type generator struct {
	target  string
	rate    float64 // requests per second
	items   []string
	client  *http.Client
	logger  *slog.Logger
	summary time.Duration

	mu     sync.Mutex
	recent []string // recently created order IDs, for GETs

	sent, failed atomic.Int64
}

func newGenerator(target string, rate float64, logger *slog.Logger) *generator {
	return &generator{
		target:  target,
		rate:    rate,
		items:   catalog.Items(),
		client:  &http.Client{Timeout: 5 * time.Second},
		logger:  logger,
		summary: 10 * time.Second,
	}
}

// run sends requests at g.rate until ctx is done. Each request runs in its
// own goroutine so slow responses don't lower the rate.
func (g *generator) run(ctx context.Context) {
	tick := time.NewTicker(time.Duration(float64(time.Second) / g.rate))
	defer tick.Stop()
	report := time.NewTicker(g.summary)
	defer report.Stop()
	var wg sync.WaitGroup
	defer wg.Wait()
	var lastSent, lastFailed int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				g.once(ctx)
			}()
		case <-report.C:
			sent, failed := g.sent.Load(), g.failed.Load()
			g.logger.Info("traffic summary", "sent", sent-lastSent, "failed", failed-lastFailed, "interval_s", g.summary.Seconds())
			lastSent, lastFailed = sent, failed
		}
	}
}

// once sends one request: mostly new orders, sometimes a lookup of a
// recent one.
func (g *generator) once(ctx context.Context) {
	var req *http.Request
	var err error
	if id, ok := g.recentOrder(); ok && rand.IntN(5) == 0 {
		req, err = http.NewRequestWithContext(ctx, "GET", g.target+"/orders/"+id, nil)
	} else {
		body, _ := json.Marshal(map[string]any{
			"item_id":  g.pickItem(),
			"quantity": 1 + rand.IntN(3),
		})
		req, err = http.NewRequestWithContext(ctx, "POST", g.target+"/orders", bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return
	}
	req.Header.Set(requestid.Header, requestid.New())
	g.sent.Add(1)
	resp, err := g.client.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			g.failed.Add(1)
		}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		g.failed.Add(1)
	}
	if req.Method == "POST" && resp.StatusCode == http.StatusCreated {
		var o struct {
			ID string `json:"id"`
		}
		if json.NewDecoder(resp.Body).Decode(&o) == nil && o.ID != "" {
			g.remember(o.ID)
		}
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}

// Like a real shop, a few items sell more than the rest: bestsellerShare of
// orders go to one of bestsellers, and the rest are spread over all items.
var bestsellers = []string{"SKU-1007", "SKU-1023", "SKU-1042"}

const bestsellerShare = 0.2

func (g *generator) pickItem() string {
	if rand.Float64() < bestsellerShare {
		return bestsellers[rand.IntN(len(bestsellers))]
	}
	return g.items[rand.IntN(len(g.items))]
}

const recentMax = 100

func (g *generator) remember(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recent = append(g.recent, id)
	if len(g.recent) > recentMax {
		g.recent = g.recent[len(g.recent)-recentMax:]
	}
}

func (g *generator) recentOrder() (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.recent) == 0 {
		return "", false
	}
	return g.recent[rand.IntN(len(g.recent))], true
}
