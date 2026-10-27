package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const unlockUsage = "usage: ./unlock.sh N <answer>   (N is the scenario you solved, 1 to %d)\n"

type question struct {
	title string // "N. Title", as in the worksheet's headings
	body  string
	mode  string // the lab mode that runs while this question is open, if any
	hash  string // answerHash of this question's answer
}

// modeSwitch is the part of incident.Switch that unlock uses.
type modeSwitch interface {
	Set(letter string) error
	Clear() error
	Duration() time.Duration
}

type worksheet struct {
	questions []question
	closing   string
}

// normalize makes matching lenient: case, surrounding spaces and quotes or
// backticks, and runs of inner spaces don't matter.
func normalize(answer string) string {
	s := strings.Trim(strings.TrimSpace(answer), "\"'`")
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func answerHash(n int, answer string) string {
	sum := sha256.Sum256([]byte("lab-q" + strconv.Itoa(n) + ":" + normalize(answer)))
	return hex.EncodeToString(sum[:])
}

// unlock checks the answer to question N (args are N and the answer) and
// prints the next question, or the closing message after the last one. It
// starts the mode the next question needs, or clears the lab when it needs
// none. A wrong answer changes nothing, and running a right one again
// restarts the mode.
func unlock(args []string, ws worksheet, sw modeSwitch, stdout, stderr io.Writer) int {
	n := 0
	if len(args) == 2 {
		n, _ = strconv.Atoi(args[0])
	}
	if n < 1 || n > len(ws.questions) || normalize(args[1]) == "" {
		fmt.Fprintf(stderr, unlockUsage, len(ws.questions))
		return 2
	}
	if answerHash(n, args[1]) != ws.questions[n-1].hash {
		fmt.Fprintln(stdout, "Not quite. Check your answer and try again.")
		return 1
	}
	var next question
	if n < len(ws.questions) {
		next = ws.questions[n]
	}
	var err error
	if next.mode != "" {
		err = sw.Set(next.mode)
	} else {
		err = sw.Clear()
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if n == len(ws.questions) {
		fmt.Fprintf(stdout, "Correct.\n\n%s\n", ws.closing)
		return 0
	}
	fmt.Fprintf(stdout, "Correct. Here is scenario %d.\n\n## %s\n\n", n+1, next.title)
	if next.mode != "" {
		fmt.Fprintf(stdout, "Mode %s is now running. It ends by itself after %s; to restart it, run\n./unlock.sh %d <answer> again.\n\n", next.mode, sw.Duration(), n)
	}
	fmt.Fprintf(stdout, "%s\n\nWhen you have the password: ./unlock.sh %d <answer>\n", next.body, n+1)
	return 0
}
