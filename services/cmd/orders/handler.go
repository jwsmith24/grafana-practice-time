package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
)

type order struct {
	ID       string `json:"id"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

type server struct {
	inv *inventoryClient

	mu     sync.Mutex
	orders map[string]order
	next   int
}

func newHandler(logger *slog.Logger, inv *inventoryClient) http.Handler {
	s := &server{inv: inv, orders: map[string]order{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /orders", s.create)
	mux.HandleFunc("GET /orders/{id}", s.get)
	return httpx.Log(logger, mux, "/healthz")
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "body must be {\"item_id\": string, \"quantity\": int}"})
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	ctx := r.Context()
	httpx.Annotate(ctx, "item_id", req.ItemID)

	stock, err := s.inv.stock(ctx, httpx.RequestID(ctx), req.ItemID)
	if err != nil {
		status := upstreamStatus(err)
		if status >= 500 {
			httpx.Annotate(ctx, "error", err.Error())
		}
		httpx.WriteJSON(w, status, map[string]string{"error": http.StatusText(status)})
		return
	}
	if stock < req.Quantity {
		httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": "insufficient stock"})
		return
	}

	s.mu.Lock()
	s.next++
	o := order{ID: fmt.Sprintf("ORD-%06d", s.next), ItemID: req.ItemID, Quantity: req.Quantity, Status: "created"}
	s.orders[o.ID] = o
	s.mu.Unlock()
	httpx.Annotate(ctx, "order_id", o.ID)
	httpx.WriteJSON(w, http.StatusCreated, o)
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	httpx.Annotate(r.Context(), "order_id", id)
	s.mu.Lock()
	o, ok := s.orders[id]
	s.mu.Unlock()
	if !ok {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

// upstreamStatus maps an inventory client error to the orders response code.
func upstreamStatus(err error) int {
	switch {
	case errors.Is(err, errNotFound):
		return http.StatusNotFound
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	default:
		return http.StatusBadGateway
	}
}
