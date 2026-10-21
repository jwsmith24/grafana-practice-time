package requestid

import (
	"regexp"
	"testing"
)

var v4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// T2.3-4
func TestNewIsUniqueV4(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for range 1000 {
		id := New()
		if !v4.MatchString(id) {
			t.Fatalf("%q is not a UUID v4", id)
		}
		if seen[id] {
			t.Fatalf("duplicate %s", id)
		}
		seen[id] = true
	}
}
