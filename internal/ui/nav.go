package ui

import (
	"bytes"
	"math"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// navigateToMsg asks the model to start a navigation; the initial CLI
// argument goes through this message so it shares the async load path.
type navigateToMsg struct {
	url string
}

// docLoadedMsg delivers a resolved document. seq is the monotonic
// navigation sequence at launch time; stale results are dropped.
type docLoadedMsg struct {
	seq      int
	url      string
	markdown string
}

// loadFailedMsg delivers a navigation failure, same seq contract.
type loadFailedMsg struct {
	seq int
	err error
}

// resizeTickMsg fires after the resize debounce delay.
type resizeTickMsg struct {
	seq int
}

// statusMsg sets a transient statusbar message.
type statusMsg string

// histEntry is one back/forward stack item.
type histEntry struct {
	url      string
	markdown string
	frac     float64
}

// historyCap bounds each of the back and forward stacks.
const historyCap = 50

// pushCapped appends e, dropping the oldest entries beyond historyCap.
func pushCapped(stack []histEntry, e histEntry) []histEntry {
	stack = append(stack, e)
	if len(stack) > historyCap {
		stack = stack[len(stack)-historyCap:]
	}
	return stack
}

// clampFraction sanitizes a scroll percentage into [0, 1]; content that
// fits the viewport (or a NaN from an empty viewport) is fraction 0.
func clampFraction(p float64, fits bool) float64 {
	if fits || math.IsNaN(p) {
		return 0
	}
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// restoreOffset converts a stored scroll fraction back to a YOffset for
// content of totalLines rendered lines in a viewport of height rows.
func restoreOffset(frac float64, totalLines, height int) int {
	scrollable := totalLines - height
	if scrollable < 0 {
		scrollable = 0
	}
	return int(math.Round(frac * float64(scrollable)))
}

// displayHost extracts a host for the loading statusbar message.
func displayHost(input string) string {
	s := strings.TrimSpace(input)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		return u.Host
	}
	return input
}

// hostOf returns the host of a URL, or "" when it cannot be parsed.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// openerWait bounds how long openExternal waits for the opener's exit
// status: `open`/`xdg-open` normally exit immediately, but some xdg-open
// configurations block until the browser itself exits.
const openerWait = 3 * time.Second

// openExternal opens a URL in the system browser. The opener's exit
// status is observed (with a timeout) before success is reported:
// `xdg-open` starts fine yet exits non-zero when no handler is
// configured, and a premature "opened" message would strand the user.
func openExternal(target string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", target)
		default:
			cmd = exec.Command("xdg-open", target)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			return statusMsg("open failed: " + err.Error())
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				msg := "open failed: " + err.Error()
				if detail := strings.TrimSpace(stderr.String()); detail != "" {
					msg += " — " + detail
				}
				return statusMsg(msg)
			}
			return statusMsg("opened in browser: " + target)
		case <-time.After(openerWait):
			// Still running after the grace period: assume a blocking
			// opener that launched the browser successfully.
			return statusMsg("opened in browser: " + target)
		}
	}
}
