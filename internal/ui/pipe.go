package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pipeURL is the pseudo-URL of a pipe-result page; like other internal
// pages it enters history, so back returns to the piped document.
const pipeURL = "manul:pipe"

// pipeTimeout bounds how long a piped command may run: the page source
// arrives on stdin and there is no terminal, so anything longer than
// this is a command waiting for input that will never come.
const pipeTimeout = 30 * time.Second

// pipeOutputCap truncates command output to keep the result page
// renderable; matches the interactive render cap.
const pipeOutputCap = 1 << 20 // 1 MiB

// pipeDoneMsg delivers a finished pipe command as a ready-to-show page.
type pipeDoneMsg struct {
	page string
}

// runPipe executes command via `sh -c` with the page's raw markdown on
// stdin and returns the result as an internal page.
func runPipe(command, input, sourceURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), pipeTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Stdin = strings.NewReader(input)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		if ctx.Err() == context.DeadlineExceeded {
			err = errors.New("timed out after 30s (command must read stdin and exit)")
		}
		return pipeDoneMsg{page: pipePage(command, sourceURL, out.Bytes(), err)}
	}
}

// pipePage renders a pipe result as markdown. The output lives in a
// four-backtick fence so triple-backtick content cannot break out.
func pipePage(command, sourceURL string, output []byte, err error) string {
	var b strings.Builder
	b.WriteString("# Pipe result\n\n")
	fmt.Fprintf(&b, "`%s` ← %s\n\n", command, sourceURL)
	truncated := false
	if len(output) > pipeOutputCap {
		output = output[:pipeOutputCap]
		truncated = true
	}
	text := strings.TrimRight(string(output), "\n")
	if text == "" {
		text = "(no output)"
	}
	b.WriteString("````text\n")
	b.WriteString(text)
	b.WriteString("\n````\n")
	if truncated {
		b.WriteString("\n*(output truncated at 1 MiB)*\n")
	}
	if err != nil {
		fmt.Fprintf(&b, "\n**command failed:** %s\n", err)
	}
	return b.String()
}
