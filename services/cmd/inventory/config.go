package main

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"
)

// configReloadInterval is how often inventory re-reads its settings.
const configReloadInterval = 2 * time.Second

// renamedKeys maps deprecated config keys to their replacements.
var renamedKeys = map[string]string{
	"ttl_seconds": "cache_ttl",
}

// settings is inventory's runtime configuration.
var settings = map[string]string{
	"ttl_seconds": "300",
	"max_items":   "500",
}

// watchConfig re-validates settings on every tick until ctx is done,
// warning about deprecated keys each time.
func watchConfig(ctx context.Context, logger *slog.Logger, cfg map[string]string, tick <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			for _, k := range slices.Sorted(maps.Keys(cfg)) {
				if repl, ok := renamedKeys[k]; ok {
					logger.Warn("config key " + k + " is deprecated, use " + repl)
				}
			}
		}
	}
}
