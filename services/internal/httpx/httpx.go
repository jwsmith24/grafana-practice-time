// Package httpx holds HTTP helpers shared by the lab services.
package httpx

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

type ctxKey struct{}

// RequestID returns the request ID stored on ctx by Log.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// requestLog collects attributes handlers add to the request's log line.
type requestLog struct{ attrs []any }

type logKey struct{}

// Annotate adds key/value attributes to the request's log line.
func Annotate(ctx context.Context, args ...any) {
	if l, ok := ctx.Value(logKey{}).(*requestLog); ok {
		l.attrs = append(l.attrs, args...)
	}
}

// Log wraps h so each request emits one log line with request_id, endpoint
// (the matched route pattern), status and duration_ms, plus anything
// handlers add with Annotate. Paths in skip are not logged.
func Log(logger *slog.Logger, h http.Handler, skip ...string) http.Handler {
	skipped := make(map[string]bool, len(skip))
	for _, p := range skip {
		skipped[p] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipped[r.URL.Path] {
			h.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		id := requestid.FromRequest(r)
		rl := &requestLog{}
		ctx := context.WithValue(r.Context(), ctxKey{}, id)
		ctx = context.WithValue(ctx, logKey{}, rl)
		r = r.WithContext(ctx)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(rec, r)

		endpoint := r.Pattern
		if endpoint == "" {
			endpoint = r.Method + " unmatched"
		}
		level := slog.LevelInfo
		if rec.status >= 500 {
			level = slog.LevelError
		}
		attrs := append([]any{
			"request_id", id,
			"endpoint", endpoint,
			"status", rec.status,
			"duration_ms", DurationMS(time.Since(start)),
		}, rl.attrs...)
		logger.Log(ctx, level, "request", attrs...)
	})
}

// DurationMS converts d to milliseconds rounded to microsecond precision.
func DurationMS(d time.Duration) float64 {
	return math.Round(float64(d.Microseconds())) / 1000
}

// HealthcheckFlag registers -healthcheck. When it is set, callers run
// RunHealthcheck instead of serving: the images have no shell or curl, so
// the binary checks itself.
func HealthcheckFlag() *bool {
	return flag.Bool("healthcheck", false, "GET /healthz on the local server and exit 0 if it returns 200")
}

// RunHealthcheck exits 0 if GET http://127.0.0.1:<port>/healthz returns 200.
func RunHealthcheck(port string) {
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%s/healthz", port))
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

// Getenv returns the environment variable key, or def if it is unset.
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
