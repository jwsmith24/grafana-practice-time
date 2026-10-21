//go:build lifecycle

package lifecycle

import (
	"os"
	"strings"
	"testing"
)

var privateNets = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

// egressRules drop new outbound connections from containers (DOCKER-USER)
// and from the host (OUTPUT), except to loopback and private networks.
// Each rule is inserted at position 1, so the last one listed is checked
// first.
func egressRules() [][]string {
	var rules [][]string
	for _, chain := range []string{"DOCKER-USER", "OUTPUT"} {
		rules = append(rules, []string{chain, "-j", "DROP"})
		rules = append(rules, []string{chain, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"})
		for _, n := range privateNets {
			rules = append(rules, []string{chain, "-d", n, "-j", "ACCEPT"})
		}
	}
	return append(rules, []string{"OUTPUT", "-o", "lo", "-j", "ACCEPT"})
}

func iptables(t *testing.T, args ...string) {
	t.Helper()
	if os.Geteuid() == 0 {
		must(t, nil, "iptables", args...)
	} else {
		must(t, nil, "sudo", append([]string{"-n", "iptables"}, args...)...)
	}
}

// blockEgress cuts the lab off from the internet until the test ends.
func blockEgress(t *testing.T) {
	t.Helper()
	rules := egressRules()
	t.Cleanup(func() {
		for _, r := range rules {
			iptables(t, append([]string{"-D"}, r...)...)
		}
	})
	for _, r := range rules {
		iptables(t, append([]string{"-I", r[0], "1"}, r[1:]...)...)
	}
}

// T5.4-1, T5.4-2
func TestOffline(t *testing.T) {
	if os.Getenv("LIFECYCLE_OFFLINE") != "1" {
		t.Skip("set LIFECYCLE_OFFLINE=1 to block egress with iptables")
	}
	freshStack(t) // the first up: pulls and builds, while online
	blockEgress(t)

	// The block works: neither a container nor the host can reach out.
	if out, code := run(t, nil, "docker", "compose", "exec", "-T", "alloy",
		"timeout", "5", "bash", "-c", "exec 3<>/dev/tcp/1.1.1.1/443"); code == 0 {
		t.Fatalf("a container still reaches 1.1.1.1 with egress blocked: %s", out)
	}
	if out, code := run(t, nil, "curl", "--noproxy", "*", "-sS", "-m", "5", "https://1.1.1.1"); code == 0 {
		t.Fatalf("the host still reaches 1.1.1.1 with egress blocked: %s", out)
	}

	for _, down := range [][]string{{"down"}, {"down", "-v"}} {
		t.Run(strings.Join(down, " "), func(t *testing.T) {
			compose(t, down...)
			compose(t, "up", "-d", "--pull", "never", "--wait")
			if out, code, _ := checkSh(t); code != 0 {
				t.Fatalf("check.sh offline: exit %d\n%s", code, out)
			}
		})
	}
}
