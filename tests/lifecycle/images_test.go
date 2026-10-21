//go:build lifecycle

package lifecycle

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// thirdPartyImages returns the compose images that are pulled, not built.
func thirdPartyImages(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, img := range strings.Fields(compose(t, "config", "--images")) {
		if !strings.HasPrefix(img, "grafana-lab/") {
			out = append(out, img)
		}
	}
	return out
}

// inspectImage reads img's manifest list from its registry, retrying when
// Docker Hub rate-limits anonymous requests (429).
func inspectImage(t *testing.T, img string) string {
	t.Helper()
	var out string
	var code int
	for wait := 10 * time.Second; wait <= 80*time.Second; wait *= 2 {
		out, code = run(t, nil, "docker", "buildx", "imagetools", "inspect", img)
		if code == 0 || !strings.Contains(out, "429") {
			break
		}
		t.Logf("%s: rate-limited, retrying in %s", img, wait)
		time.Sleep(wait)
	}
	if code != 0 {
		t.Fatalf("docker buildx imagetools inspect %s: exit %d\n%s", img, code, out)
	}
	return out
}

// T5.1-3
func TestMultiArch(t *testing.T) {
	for _, img := range thirdPartyImages(t) {
		out := inspectImage(t, img)
		for _, p := range []string{"linux/amd64", "linux/arm64"} {
			if !regexp.MustCompile(`Platform:\s+` + p + `\b`).MatchString(out) {
				t.Errorf("%s has no %s image", img, p)
			}
		}
	}
	// The service build cross-compiles, so it needs no emulation; the
	// docker-container driver is needed for a multi-platform build.
	if _, code := run(t, nil, "docker", "buildx", "inspect", "lab-multiarch"); code != 0 {
		must(t, nil, "docker", "buildx", "create", "--name", "lab-multiarch", "--driver", "docker-container")
		t.Cleanup(func() { run(t, nil, "docker", "buildx", "rm", "lab-multiarch") })
	}
	for _, target := range []string{"inventory", "orders", "loadgen"} {
		must(t, nil, "docker", "buildx", "build", "--builder", "lab-multiarch",
			"--platform", "linux/amd64,linux/arm64", "--target", target, "services")
	}
}

// T5.1-7
func TestImagesFitOnDisk(t *testing.T) {
	compose(t, "up", "-d", "--wait")
	var total int64
	for _, img := range strings.Fields(compose(t, "config", "--images")) {
		n, err := strconv.ParseInt(strings.TrimSpace(must(t, nil, "docker", "image", "inspect", "--format", "{{.Size}}", img)), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%-40s %4d MB", img, n/1e6)
		total += n
	}
	t.Logf("total %d MB (target < 2000 MB)", total/1e6)
	if total >= 3e9 {
		t.Fatalf("images take %d MB, more than the README's 3 GB free-disk prerequisite", total/1e6)
	}
}
