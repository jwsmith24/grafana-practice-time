// Command labctl sets, clears and reports the lab mode.
//
//	labctl A|B|C   activate a mode (restarts its timer)
//	labctl clear   end the active mode
//	labctl status  print the active mode
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/httpx"
	"github.com/jwsmith24/grafana-practice-time/services/internal/incident"
)

const usage = "usage: labctl A|B|C|clear|status"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	d, err := incident.DurationFromEnv()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	sw := incident.New(httpx.Getenv("LAB_STATE_FILE", "/var/lib/lab/state.json"), d)

	switch arg := strings.ToUpper(args[0]); arg {
	case "CLEAR":
		if err := sw.Clear(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "cleared")
	case "STATUS":
		letter, left, err := sw.Status()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if letter == "" {
			fmt.Fprintln(stdout, "none")
		} else {
			fmt.Fprintf(stdout, "active: %s (%s left)\n", letter, left.Round(time.Second))
		}
	default:
		if err := sw.Set(arg); err != nil {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		fmt.Fprintf(stdout, "active: %s (%s left)\n", arg, sw.Duration())
	}
	return 0
}
