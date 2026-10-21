// Package auth carries the shared service token between lab services.
package auth

import (
	"crypto/subtle"
	"net/http"
)

// Header is the HTTP header that carries the service token.
const Header = "X-Service-Token"

// Require wraps h so requests without the expected token get 401.
// Paths in open skip the check.
func Require(token string, h http.Handler, open ...string) http.Handler {
	skip := make(map[string]bool, len(open))
	for _, p := range open {
		skip[p] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !skip[r.URL.Path] && !Valid(token, r.Header.Get(Header)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}` + "\n"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Valid reports whether got matches the non-empty expected token.
func Valid(want, got string) bool {
	return want != "" && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}
