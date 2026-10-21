// Package lifecycle tests the lab as an attendee runs it: each test starts,
// stops, resets or breaks the stack itself, using docker-compose.yml alone.
//
// The tests have the build tag lifecycle and run one at a time:
//
//	go test -tags lifecycle -count=1 -timeout 60m -v ./lifecycle/...
//
// LIFECYCLE_OFFLINE=1 enables the tests that block outbound traffic with
// iptables (root or passwordless sudo). LAB_SOAK=1 enables the long memory
// soak, which the nightly CI run sets.
package lifecycle
