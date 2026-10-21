// Command orders accepts orders and checks stock with inventory.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/incident"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
	"github.com/jwsmith24/grafana-practice-time/services/internal/metrics"
)

const service = "orders"

var version = "1.4.2"

type config struct {
	port             string
	inventoryURL     string
	inventoryTimeout time.Duration
	token            string
}

func loadConfig() (config, error) {
	c := config{
		port:             httpx.Getenv("PORT", "8080"),
		inventoryURL:     httpx.Getenv("INVENTORY_URL", "http://inventory:8080"),
		inventoryTimeout: DefaultInventoryTimeout,
		token:            os.Getenv("SERVICE_TOKEN"),
	}
	if v := os.Getenv("INVENTORY_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return c, err
		}
		c.inventoryTimeout = d
	}
	return c, nil
}

func main() {
	healthcheck := httpx.HealthcheckFlag()
	flag.Parse()
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("bad config", "error", err.Error())
		os.Exit(1)
	}
	if *healthcheck {
		httpx.RunHealthcheck(cfg.port)
	}

	logger, closer, err := logging.New(service, httpx.Getenv("LOG_DIR", "/var/log/lab"), os.Stdout)
	if err != nil {
		slog.Error("logger setup failed", "error", err.Error())
		os.Exit(1)
	}
	defer closer.Close()

	logStartup(logger, cfg.port)
	lab := incident.New(httpx.Getenv("LAB_STATE_FILE", "/var/lib/lab/state.json"), 0)
	rel := newRelease(logger, lab.Active)
	tick := time.NewTicker(releaseCheckInterval)
	defer tick.Stop()
	go rel.run(context.Background(), tick.C)
	inv := newInventoryClient(cfg.inventoryURL, cfg.token, cfg.inventoryTimeout)
	inv.token = func() string { return tokenFor(rel.Version(), cfg.token) }
	srv := &http.Server{Addr: ":" + cfg.port, Handler: newHandler(logger, inv, metrics.NewHTTP(service))}
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}

func logStartup(logger *slog.Logger, port string) {
	logger.Info("service started", "event", "startup", "version", version, "port", port)
	logger.Info("deploy completed", "event", "deploy", "version", version)
}
