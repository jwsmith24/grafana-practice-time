// Command inventory serves stock levels for catalog items.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
)

const service = "inventory"

var version = "2.3.0"

func main() {
	healthcheck := httpx.HealthcheckFlag()
	flag.Parse()
	port := httpx.Getenv("PORT", "8080")
	if *healthcheck {
		httpx.RunHealthcheck(port)
	}

	logger, closer, err := logging.New(service, httpx.Getenv("LOG_DIR", "/var/log/lab"), os.Stdout)
	if err != nil {
		slog.Error("logger setup failed", "error", err.Error())
		os.Exit(1)
	}
	defer closer.Close()

	logStartup(logger, port)
	reload := time.NewTicker(configReloadInterval)
	defer reload.Stop()
	go watchConfig(context.Background(), logger, settings, reload.C)
	srv := &http.Server{Addr: ":" + port, Handler: newHandler(logger, os.Getenv("SERVICE_TOKEN"))}
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}

func logStartup(logger *slog.Logger, port string) {
	logger.Info("service started", "event", "startup", "version", version, "port", port)
}
