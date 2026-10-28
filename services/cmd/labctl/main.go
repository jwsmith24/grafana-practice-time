// Command labctl sets, clears and reports the lab mode, and unlocks the
// worksheet's questions.
//
//	labctl A|B|C               activate a mode (restarts its timer)
//	labctl clear               end the active mode
//	labctl status              print the active mode, then the worksheet progress
//	labctl unlock N <answer>   check the answer to scenario N; print the next one
//	                           and start the mode it needs (or clear the lab)
//	labctl hint N [LEVEL]      print hint LEVEL (1 to 3) for an unlocked scenario
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

const usage = "usage: labctl A|B|C|clear|status, labctl unlock N <answer>, or labctl hint N [LEVEL]"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	sub := ""
	if len(args) > 0 && (args[0] == "unlock" || args[0] == "hint") {
		sub = args[0]
	}
	if len(args) != 1 && sub == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	d, err := incident.DurationFromEnv()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	stateFile := httpx.Getenv("LAB_STATE_FILE", "/var/lib/lab/state.json")
	sw := incident.New(stateFile, d)
	prog := progressFor(stateFile)
	switch sub {
	case "unlock":
		return unlock(args[1:], labWorksheet, sw, prog, stdout, stderr)
	case "hint":
		return hint(args[1:], labWorksheet, prog, stdout, stderr)
	}

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
		unlocked, err := prog.Unlocked()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if letter == "" {
			fmt.Fprintln(stdout, "none")
		} else {
			fmt.Fprintf(stdout, "active: %s (%s left)\n", letter, left.Round(time.Second))
		}
		if total := len(labWorksheet.questions); unlocked > total {
			fmt.Fprintf(stdout, "progress: all %d scenarios solved\n", total)
		} else {
			fmt.Fprintf(stdout, "progress: scenario %d of %d unlocked\n", unlocked, total)
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
