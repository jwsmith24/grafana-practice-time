// Package requestid generates and propagates X-Request-ID values.
package requestid

import (
	"crypto/rand"
	"fmt"
	"net/http"
)

// Header is the HTTP header that carries the request ID.
const Header = "X-Request-ID"

// New returns a random UUID v4.
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// FromRequest returns the request's X-Request-ID, or a new one if it has none.
func FromRequest(r *http.Request) string {
	if id := r.Header.Get(Header); id != "" {
		return id
	}
	return New()
}
