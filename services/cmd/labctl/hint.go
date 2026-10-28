package main

import (
	"fmt"
	"io"
	"strconv"
)

const hintUsage = "usage: ./hint.sh N [LEVEL]   (N is the scenario, 1 to %d; LEVEL is 1 to 3, default 1)\n"

// hint prints one hint for scenario N (args are N and, optionally, the
// level). Scenario 1's hints are always available; any other scenario's only
// once it is unlocked. Hints change neither the mode nor the progress.
func hint(args []string, ws worksheet, prog progress, stdout, stderr io.Writer) int {
	n, level := 0, 1
	if len(args) == 1 || len(args) == 2 {
		n, _ = strconv.Atoi(args[0])
	}
	if len(args) == 2 {
		level, _ = strconv.Atoi(args[1])
	}
	if n < 1 || n > len(ws.questions) || level < 1 || level > len(ws.questions[n-1].hints) {
		fmt.Fprintf(stderr, hintUsage, len(ws.questions))
		return 2
	}
	unlocked, err := prog.Unlocked()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if n > unlocked {
		fmt.Fprintf(stdout, "Scenario %d isn't unlocked yet.\n", n)
		return 1
	}
	hints := ws.questions[n-1].hints
	fmt.Fprintf(stdout, "Hint %d of %d for scenario %d:\n\n%s\n", level, len(hints), n, hints[level-1])
	if level < len(hints) {
		fmt.Fprintf(stdout, "\nStill stuck? Run ./hint.sh %d %d for the next one.\n", n, level+1)
	}
	return 0
}
