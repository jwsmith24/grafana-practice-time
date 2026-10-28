package static

import (
	"os"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// T5.6-6: check.sh tells attendees who to ask.
func TestCheckFooter(t *testing.T) {
	if !strings.Contains(readFile(t, "../../check.sh"), `echo "If it still fails, message the facilitator and include this output."`) {
		t.Error("check.sh's failure footer does not tell attendees to message the facilitator")
	}
}
