// Command loadgen sends steady traffic to orders.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/logging"
)

const service = "loadgen"

var version = "1.0.0"

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

	rate, err := strconv.ParseFloat(httpx.Getenv("RATE", "5"), 64)
	if err != nil || rate <= 0 {
		logger.Error("bad RATE", "error", "RATE must be a positive number")
		os.Exit(1)
	}
	g := newGenerator(httpx.Getenv("TARGET_URL", "http://orders:8080"), rate, logger)
	logger.Info("service started", "event", "startup", "version", version, "target", g.target, "rate", rate)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			logger.Error("health server stopped", "error", err.Error())
			os.Exit(1)
		}
	}()
	g.run(context.Background())
}
