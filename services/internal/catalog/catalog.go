// Package catalog lists the item IDs the lab services know about.
package catalog

import "fmt"

const (
	first = 1001
	count = 50
)

// Items returns every known item ID, SKU-1001 through SKU-1050.
func Items() []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = fmt.Sprintf("SKU-%04d", first+i)
	}
	return ids
}

var known = func() map[string]bool {
	m := make(map[string]bool, count)
	for _, id := range Items() {
		m[id] = true
	}
	return m
}()

// Known reports whether id is in the catalog.
func Known(id string) bool { return known[id] }
