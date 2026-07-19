package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPipePage(t *testing.T) {
	page := pipePage("wc -l", "https://a.test/llms.txt", []byte("42\n"), nil)
	for _, want := range []string{"# Pipe result", "`wc -l`", "https://a.test/llms.txt", "42"} {
		if !strings.Contains(page, want) {
			t.Errorf("pipe page missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "command failed") {
		t.Error("successful pipe reported as failed")
	}
}

func TestPipePageFailureAndEmptyOutput(t *testing.T) {
	page := pipePage("false", "manul:start", nil, errors.New("exit status 1"))
	if !strings.Contains(page, "(no output)") {
		t.Error("empty output not marked")
	}
	if !strings.Contains(page, "command failed") || !strings.Contains(page, "exit status 1") {
		t.Errorf("failure not surfaced:\n%s", page)
	}
}

func TestPipePageFenceSafety(t *testing.T) {
	page := pipePage("cat", "manul:start", []byte("```go\ncode\n```"), nil)
	if !strings.Contains(page, "````text\n```go\ncode\n```\n````") {
		t.Errorf("triple-backtick output not contained by four-backtick fence:\n%s", page)
	}
}

func TestRunPipeExecutesCommand(t *testing.T) {
	msg := runPipe("grep -c llms.txt", "- a llms.txt\n- b llms.txt\n- c other\n", "https://a.test")()
	done, ok := msg.(pipeDoneMsg)
	if !ok {
		t.Fatalf("runPipe returned %T, want pipeDoneMsg", msg)
	}
	if !strings.Contains(done.page, "\n2\n") {
		t.Errorf("grep -c result missing, page:\n%s", done.page)
	}
}

func TestPipePromptFlow(t *testing.T) {
	m := testModel(t)
	m.page = page{url: "https://a.test/llms.txt", markdown: "one llms.txt\n"}

	m, _ = apply(t, m, runeKey('|'))
	if !m.promptOpen || m.promptKind != promptPipe {
		t.Fatalf("'|' did not open the pipe prompt: open=%v kind=%d", m.promptOpen, m.promptKind)
	}

	// Keys route to the input, not to global bindings ('q' must not quit).
	var cmd tea.Cmd
	for _, r := range "wc -q" {
		m, cmd = apply(t, m, runeKey(r))
		if producesQuit(cmd) {
			t.Fatal("typing in the pipe prompt triggered quit")
		}
	}
	if m.prompt.Value() != "wc -q" {
		t.Fatalf("prompt value = %q", m.prompt.Value())
	}

	m, cmd = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.promptOpen {
		t.Error("prompt still open after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter on pipe prompt produced no command")
	}

	// Result page enters history; back returns to the piped document.
	before := m.page.url
	m, _ = apply(t, m, pipeDoneMsg{page: "# Pipe result\n\nok\n"})
	if m.page.url != pipeURL {
		t.Fatalf("pipe result not shown: on %s", m.page.url)
	}
	m, _ = apply(t, m, runeKey('H'))
	if m.page.url != before {
		t.Errorf("back from pipe result landed on %s, want %s", m.page.url, before)
	}
}
