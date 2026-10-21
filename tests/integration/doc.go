// Package integration runs against the live compose stack:
//
//	docker compose -f docker-compose.yml -f docker-compose.test.yml up -d --wait
//	go test -tags integration ./integration/...
//
// Tests reach Loki and Prometheus through Grafana's datasource proxy.
package integration
