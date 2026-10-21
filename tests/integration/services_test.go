//go:build integration

package integration

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// T2.1-7
func TestInventoryInLoki(t *testing.T) {
	if !eventually(t, 30*time.Second, func() bool {
		return countLines(lokiQuery(t, `{service="inventory"}`, 15*time.Minute)) > 0
	}) {
		t.Fatal(`{service="inventory"} still empty after 30s`)
	}
	if n := countLines(lokiQuery(t, `{service="inventory"} | json | __error__!=""`, 15*time.Minute)); n != 0 {
		t.Fatalf("%d inventory lines failed | json", n)
	}
}

// T2.2-6
func TestRequestIDAcrossServices(t *testing.T) {
	id := newUUID(t)
	req, err := http.NewRequest("POST", ordersURL()+"/orders", strings.NewReader(`{"item_id":"SKU-1001","quantity":1}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", id)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("POST /orders status %d", resp.StatusCode)
	}

	query := `{service=~"orders|inventory"} | json | request_id="` + id + `"`
	var seen map[string]int
	ok := eventually(t, 30*time.Second, func() bool {
		seen = map[string]int{}
		for _, s := range lokiQuery(t, query, 5*time.Minute) {
			seen[s.Stream["service"]] += len(s.Values)
		}
		return seen["orders"] > 0 && seen["inventory"] > 0
	})
	if !ok {
		t.Fatalf("lines per service for request_id %s: %v", id, seen)
	}
}

func newUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// T2.3-5
func TestLoadgenInLoki(t *testing.T) {
	waitForLines(t, "loadgen")
}

// T2.3-2: needs a full minute of traffic, so it waits for loadgen to have
// run for at least 65s before measuring.
func TestLoadgenRate(t *testing.T) {
	waitForLines(t, "loadgen")
	streams := lokiQuery(t, `{service="loadgen"} | json | event="startup"`, 15*time.Minute)
	if countLines(streams) == 0 {
		t.Fatal("no loadgen startup line")
	}
	ns, _ := strconv.ParseInt(streams[0].Values[0][0], 10, 64)
	if wait := time.Until(time.Unix(0, ns).Add(65 * time.Second)); wait > 0 {
		t.Logf("waiting %s for a full minute of traffic", wait.Round(time.Second))
		time.Sleep(wait)
	}
	n := lokiInstant(t, `sum(count_over_time({service="orders"} | json | endpoint=~"POST /orders|GET /orders.*" [1m]))`)
	if n < 240 || n > 360 {
		t.Fatalf("%v orders requests in the last minute, want 240-360", n)
	}
}
