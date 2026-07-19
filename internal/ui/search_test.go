package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFindSearchMatchesPlainText(t *testing.T) {
	rendered := "alpha llms.txt beta\ngamma llms.txt\nno hit here"
	matches, lines := findSearchMatches(rendered, "llms.txt")
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
	if lines[0] != 0 || lines[1] != 1 {
		t.Errorf("match lines = %v, want [0 1]", lines)
	}
	if got := rendered[matches[0].start:matches[0].end]; got != "llms.txt" {
		t.Errorf("first match span = %q", got)
	}
}

func TestFindSearchMatchesAcrossANSI(t *testing.T) {
	rendered := "x ll\x1b[1mms.t\x1b[0mxt y"
	matches, _ := findSearchMatches(rendered, "llms.txt")
	if len(matches) != 1 {
		t.Fatalf("got %d matches across ANSI chunks, want 1", len(matches))
	}
	span := rendered[matches[0].start:matches[0].end]
	if !strings.HasPrefix(span, "ll") || !strings.HasSuffix(span, "xt") {
		t.Errorf("span = %q, want it to cover the split needle", span)
	}
}

func TestFindSearchMatchesSmartCase(t *testing.T) {
	rendered := "Foo foo FOO"
	if m, _ := findSearchMatches(rendered, "foo"); len(m) != 3 {
		t.Errorf("lowercase query: got %d matches, want 3 (case-insensitive)", len(m))
	}
	if m, _ := findSearchMatches(rendered, "Foo"); len(m) != 1 {
		t.Errorf("mixed-case query: got %d matches, want 1 (exact)", len(m))
	}
}

func TestFindSearchMatchesNeverCrossLines(t *testing.T) {
	rendered := "wrap ends fo\no continues"
	if m, _ := findSearchMatches(rendered, "foo"); len(m) != 0 {
		t.Errorf("match crossed a rendered line break: %v", m)
	}
}

func TestHighlightSpansAppliesAllAndSkipsOverlaps(t *testing.T) {
	rendered := "aaa bbb aaa"
	matches, _ := findSearchMatches(rendered, "aaa")
	out := highlightSpans(rendered, matches)
	if got := strings.Count(out, reverseOn); got < 2 {
		t.Errorf("expected both matches highlighted, reverseOn count = %d", got)
	}
	// Overlapping span must be dropped, not corrupt the output.
	out = highlightSpans(rendered, []span{{0, 5}, {3, 8}})
	if !strings.Contains(stripANSITest(out), "aaa bbb aaa") {
		t.Errorf("overlap handling corrupted text: %q", out)
	}
}

func stripANSITest(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if isCSIStart(s, i) {
			length, _, _ := readCSI(s, i)
			i += length
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func TestSearchFlow(t *testing.T) {
	m := testModel(t)
	m.setPage("https://a.test/x.md", strings.Repeat("filler\n\n", 40)+"needle one\n\nmore filler\n\nneedle two\n")

	m, _ = apply(t, m, runeKey('/'))
	if !m.promptOpen || m.promptKind != promptSearch {
		t.Fatalf("'/' did not open search prompt: open=%v kind=%d", m.promptOpen, m.promptKind)
	}
	for _, r := range "needle" {
		var cmd tea.Cmd
		m, cmd = apply(t, m, runeKey(r))
		if producesQuit(cmd) {
			t.Fatal("typing in the search prompt triggered quit")
		}
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.search.active() || len(m.search.matches) != 2 {
		t.Fatalf("search state: active=%v matches=%d, want 2", m.search.active(), len(m.search.matches))
	}
	if m.viewport.YOffset == 0 {
		t.Error("viewport did not scroll to the first match")
	}
	firstLine := m.search.lines[0]
	if firstLine < m.viewport.YOffset || firstLine > m.viewport.YOffset+m.viewport.Height-1 {
		t.Errorf("first match line %d not visible at offset %d", firstLine, m.viewport.YOffset)
	}

	m, _ = apply(t, m, runeKey('n'))
	if m.search.current != 1 {
		t.Errorf("n: current = %d, want 1", m.search.current)
	}
	m, _ = apply(t, m, runeKey('n')) // wraps
	if m.search.current != 0 {
		t.Errorf("n wrap: current = %d, want 0", m.search.current)
	}
	m, _ = apply(t, m, runeKey('N')) // wraps backward
	if m.search.current != 1 {
		t.Errorf("N wrap: current = %d, want 1", m.search.current)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.active() {
		t.Error("esc did not clear the search")
	}
}

func TestSearchClearedOnNavigation(t *testing.T) {
	m := testModel(t)
	m.setPage("https://a.test/x.md", "needle\n")
	m.startSearch("needle")
	if !m.search.active() {
		t.Fatal("search not active after startSearch")
	}
	m.setPage("https://a.test/y.md", "other\n")
	if m.search.active() {
		t.Error("search survived navigation to a new page")
	}
}

func TestSearchNoMatches(t *testing.T) {
	m := testModel(t)
	m.setPage("https://a.test/x.md", "content\n")
	m.startSearch("absent")
	if len(m.search.matches) != 0 {
		t.Fatal("unexpected matches")
	}
	if !strings.Contains(m.statusLeft(), "no matches") {
		t.Errorf("statusLeft = %q, want a no-matches notice", m.statusLeft())
	}
	m.status = "" // transient cleared by next keypress; indicator remains
	if !strings.Contains(m.statusLeft(), "/absent") {
		t.Errorf("statusLeft = %q, want persistent /absent indicator", m.statusLeft())
	}
}
