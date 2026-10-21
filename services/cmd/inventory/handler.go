package main

import (
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/catalog"
	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
)

type stockResponse struct {
	ItemID string `json:"item_id"`
	Stock  int    `json:"stock"`
}

// store looks up stock levels. mode reports the lab mode (nil means none),
// and sleep is time.Sleep outside tests.
type store struct {
	mode  func() string
	sleep func(time.Duration)
}

// lookupDelay is how long a stock lookup takes in the given mode.
func lookupDelay(mode string) time.Duration {
	if mode != "B" {
		return 0
	}
	return 2*time.Second + rand.N(2*time.Second)
}

func (s store) stock(id string) int {
	if s.mode != nil {
		if d := lookupDelay(s.mode()); d > 0 {
			s.sleep(d)
		}
	}
	return stockFor(id)
}

func newHandler(logger *slog.Logger, token string, m *metrics.HTTP, st store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", m.Handler())
	mux.HandleFunc("GET /inventory/{item_id}", st.getStock)
	return httpx.Instrument(logger, m, auth.Require(token, mux, "/healthz", "/metrics"), "/healthz", "/metrics")
}

func (s store) getStock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("item_id")
	httpx.Annotate(r.Context(), "item_id", id)
	if !catalog.Known(id) {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "unknown item"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stockResponse{ItemID: id, Stock: s.stock(id)})
}

// stockFor returns a deterministic stock level for id.
func stockFor(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32() % 200)
}
