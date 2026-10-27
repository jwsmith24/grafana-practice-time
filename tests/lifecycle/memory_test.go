//go:build lifecycle

package lifecycle

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const memBudget = 2 << 30 // bytes

var memUnits = map[string]float64{"B": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30}

// parseMem parses the used half of docker stats' MemUsage, e.g. "33.5MiB".
func parseMem(t *testing.T, s string) float64 {
	t.Helper()
	used, _, _ := strings.Cut(s, " / ")
	for unit, mult := range memUnits {
		if num, ok := strings.CutSuffix(used, unit); ok {
			if f, err := strconv.ParseFloat(num, 64); err == nil {
				return f * mult
			}
		}
	}
	t.Fatalf("can't parse memory %q", s)
	return 0
}

// labMemory returns the memory the lab's containers use right now.
func labMemory(t *testing.T) float64 {
	t.Helper()
	ids := strings.Fields(compose(t, "ps", "-q"))
	if len(ids) != 7 {
		t.Fatalf("%d lab containers running, want 7", len(ids))
	}
	out := must(t, nil, "docker", append([]string{"stats", "--no-stream", "--format", "{{.MemUsage}}"}, ids...)...)
	var total float64
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		total += parseMem(t, l)
	}
	return total
}

// sampleMemory samples the lab's memory every interval until done returns
// true, and returns the highest total seen.
func sampleMemory(t *testing.T, interval time.Duration, done func() bool) float64 {
	t.Helper()
	var peak float64
	for {
		if m := labMemory(t); m > peak {
			peak = m
		}
		if done() {
			return peak
		}
		time.Sleep(interval)
	}
}

func TestMemory(t *testing.T) {
	if os.Getenv("LAB_SOAK") != "1" {
		t.Skip("set LAB_SOAK=1 for the 10-minute memory soak (nightly)")
	}
	freshStack(t)

	// T5.5-1
	t.Run("10 minutes with loadgen", func(t *testing.T) {
		end := time.Now().Add(10 * time.Minute)
		peak := sampleMemory(t, 30*time.Second, func() bool { return time.Now().After(end) })
		t.Logf("peak %.0f MiB", peak/(1<<20))
		if peak >= memBudget {
			t.Fatalf("peak %.0f MiB, want < 2048 MiB", peak/(1<<20))
		}
	})

	// T5.5-2: A makes requests pile up in inventory and orders.
	t.Run("during A", func(t *testing.T) {
		must(t, nil, "./incident.sh", "A")
		t.Cleanup(func() { run(t, nil, "./incident.sh", "clear") })
		peak := sampleMemory(t, 10*time.Second, func() bool {
			return strings.TrimSpace(must(t, nil, "./incident.sh", "status")) == "none"
		})
		t.Logf("peak %.0f MiB", peak/(1<<20))
		if peak >= memBudget {
			t.Fatalf("peak %.0f MiB during A, want < 2048 MiB", peak/(1<<20))
		}
	})
}
