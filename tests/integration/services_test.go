//go:build integration

package integration

import (
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
