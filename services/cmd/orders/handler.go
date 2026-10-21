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
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type order struct {
	ID       string `json:"id"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

type server struct {
	inv            *inventoryClient
	upstreamErrors *prometheus.CounterVec

	mu     sync.Mutex
	orders map[string]order
	next   int
}

// upstreamReasons are the inventory_upstream_errors_total reason values.
var upstreamReasons = []string{"timeout", "server_error", "unauthorized"}

func newHandler(logger *slog.Logger, inv *inventoryClient, m *metrics.HTTP) http.Handler {
	s := &server{
		inv:    inv,
		orders: map[string]order{},
		upstreamErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "inventory_upstream_errors_total",
			Help: "Failed calls from orders to inventory, by reason.",
		}, []string{"reason"}),
	}
	m.Registry.MustRegister(s.upstreamErrors)
	for _, r := range upstreamReasons {
		s.upstreamErrors.WithLabelValues(r)
	}
	m.Init("POST /orders", http.StatusBadGateway, http.StatusGatewayTimeout)
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", m.Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /orders", s.create)
	mux.HandleFunc("GET /orders/{id}", s.get)
	return httpx.Instrument(logger, m, mux, "/healthz", "/metrics")
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
		if reason := upstreamReason(err); reason != "" {
			s.upstreamErrors.WithLabelValues(reason).Inc()
		}
		if status >= 500 || errors.Is(err, errNotFound) {
			httpx.Annotate(ctx, "error", err.Error())
			httpx.SetLevel(ctx, slog.LevelError)
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

// upstreamReason classifies a failed inventory call for metrics. It returns
// "" for errors that aren't upstream failures, like an unknown item.
func upstreamReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, errUnauthorized):
		return "unauthorized"
	case errors.Is(err, errUpstream5xx):
		return "server_error"
	case errors.Is(err, errNotFound):
		return ""
	default:
		return "server_error"
	}
}
