package main

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
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
	if mode != "A" {
		return 0
	}
	return 2*time.Second + rand.N(2*time.Second)
}

func (s store) stock(id string) (int, error) {
	mode := ""
	if s.mode != nil {
		mode = s.mode()
	}
	if d := lookupDelay(mode); d > 0 {
		s.sleep(d)
	}
	rec := record(id, mode)
	n, err := strconv.Atoi(rec)
	if err != nil {
		return 0, fmt.Errorf("parse stock record for %s: %w", id, err)
	}
	return n, nil
}

// record returns the stored stock record for id.
func record(id, mode string) string {
	n := strconv.Itoa(stockFor(id))
	if mode == "C" && id == "SKU-1042" {
		return n + ".0"
	}
	return n
}

func newHandler(logger *slog.Logger, token string, m *metrics.HTTP, st store) http.Handler {
	m.Init("GET /inventory/{item_id}", http.StatusInternalServerError)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", m.Handler())
	mux.HandleFunc("GET /inventory/{item_id}", st.getStock)
	rl := newRejectLog(logger)
	return httpx.Instrument(logger, m, auth.RequireNotify(token, mux, rl.note, "/healthz", "/metrics"), "/healthz", "/metrics")
}

func (s store) getStock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("item_id")
	httpx.Annotate(r.Context(), "item_id", id)
	if !catalog.Known(id) {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "unknown item"})
		return
	}
	n, err := s.stock(id)
	if err != nil {
		httpx.Annotate(r.Context(), "error", err.Error())
		httpx.SetLevel(r.Context(), slog.LevelError)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stockResponse{ItemID: id, Stock: n})
}

// stockFor returns a deterministic stock level for id.
func stockFor(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32() % 200)
}
