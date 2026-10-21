package main

import (
	"hash/fnv"
	"log/slog"
	"net/http"

	"github.com/jwsmith24/grafana-practice-time/services/internal/catalog"
	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
)

type stockResponse struct {
	ItemID string `json:"item_id"`
	Stock  int    `json:"stock"`
}

func newHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /inventory/{item_id}", getStock)
	return httpx.Log(logger, mux, "/healthz")
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
