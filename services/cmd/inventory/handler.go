package main

import (
	"hash/fnv"
	"log/slog"
	"net/http"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/catalog"
	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
)

type stockResponse struct {
	ItemID string `json:"item_id"`
	Stock  int    `json:"stock"`
}

func newHandler(logger *slog.Logger, token string, m *metrics.HTTP) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", m.Handler())
	mux.HandleFunc("GET /inventory/{item_id}", getStock)
	return httpx.Instrument(logger, m, auth.Require(token, mux, "/healthz", "/metrics"), "/healthz", "/metrics")
}

func getStock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("item_id")
	httpx.Annotate(r.Context(), "item_id", id)
	if !catalog.Known(id) {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "unknown item"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stockResponse{ItemID: id, Stock: stockFor(id)})
}

// stockFor returns a deterministic stock level for id.
func stockFor(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32() % 200)
}
