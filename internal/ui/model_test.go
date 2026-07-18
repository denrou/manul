package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/denrou/manul/internal/bookmarks"
	"github.com/denrou/manul/internal/config"
)

func testModel(t *testing.T) Model {
	t.Helper()
	m := New(Options{Config: config.Defaults(), Style: glamour.WithStandardStyle("dark")})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

func apply(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// producesQuit expands a command (one batch level deep is enough for
// these tests) and reports whether it yields tea.QuitMsg. Only call it
// on commands known not to trigger I/O.
func producesQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if c != nil {
				if _, ok := c().(tea.QuitMsg); ok {
					return true
				}
			}
		}
	}
	return false
}

func TestStaleNavigationDropped(t *testing.T) {
	m := testModel(t)
	m, _ = apply(t, m, navigateToMsg{url: "example.com"})
	if !m.loading || m.navSeq != 1 {
		t.Fatalf("navigate did not start: loading=%v seq=%d", m.loading, m.navSeq)
	}

	m, _ = apply(t, m, docLoadedMsg{seq: 0, url: "https://stale.test/", markdown: "# stale"})
	if m.page.url != startURL {
		t.Errorf("stale docLoadedMsg was applied: page=%s", m.page.url)
	}
	if !m.loading {
		t.Error("stale docLoadedMsg cleared the loading state")
	}

	m, _ = apply(t, m, loadFailedMsg{seq: 0, err: errFake})
	if m.status != "" {
		t.Errorf("stale loadFailedMsg set a status: %q", m.status)
	}

	m, _ = apply(t, m, docLoadedMsg{seq: m.navSeq, url: "https://fresh.test/llms.txt", markdown: "# Fresh"})
	if m.page.url != "https://fresh.test/llms.txt" {
		t.Errorf("current docLoadedMsg not applied: page=%s", m.page.url)
	}
	if m.loading {
		t.Error("loading still set after a successful load")
	}
}

func TestEscCancelMakesInFlightResultStale(t *testing.T) {
	m := testModel(t)
	m, _ = apply(t, m, navigateToMsg{url: "example.com"})
	seq := m.navSeq

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.loading {
		t.Fatal("esc did not cancel the in-flight load")
	}
	if m.status != "load canceled" {
		t.Errorf("status = %q", m.status)
	}

	m, _ = apply(t, m, docLoadedMsg{seq: seq, url: "https://late.test/", markdown: "# late"})
	if m.page.url != startURL {
		t.Error("late result of a canceled fetch was applied")
	}
}

func TestPromptSwallowsQ(t *testing.T) {
	m := testModel(t)
	m, _ = apply(t, m, runeKey(':'))
	if !m.promptOpen {
		t.Fatal("':' did not open the prompt")
	}

	m, cmd := apply(t, m, runeKey('q'))
	if producesQuit(cmd) {
		t.Fatal("'q' typed in the prompt quit the program")
	}
	if m.prompt.Value() != "q" {
		t.Errorf("prompt value = %q, want %q", m.prompt.Value(), "q")
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.promptOpen {
		t.Fatal("esc did not close the prompt")
	}

	_, cmd = apply(t, m, runeKey('q'))
	if !producesQuit(cmd) {
		t.Error("'q' outside the prompt did not quit")
	}
}

func TestPromptEnterNavigates(t *testing.T) {
	m := testModel(t)
	m, _ = apply(t, m, runeKey(':'))
	for _, r := range "example.com" {
		m, _ = apply(t, m, runeKey(r))
	}
	m, cmd := apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.promptOpen {
		t.Error("enter did not close the prompt")
	}
	if !m.loading || m.loadingHost != "example.com" {
		t.Errorf("navigation not started: loading=%v host=%q", m.loading, m.loadingHost)
	}
	if cmd == nil {
		t.Error("no command returned for the fetch")
	}
}

func TestPromptRowChangesChrome(t *testing.T) {
	m := testModel(t)
	base := m.viewport.Height
	m, _ = apply(t, m, runeKey(':'))
	if m.viewport.Height != base-1 {
		t.Errorf("viewport height with prompt = %d, want %d", m.viewport.Height, base-1)
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.viewport.Height != base {
		t.Errorf("viewport height after close = %d, want %d", m.viewport.Height, base)
	}
}

func TestEscPrecedenceAndNeverQuits(t *testing.T) {
	m := testModel(t)
	(&m).setPage("https://a.test/", "[one](https://example.com/one.md)")
	m.loading = true
	m.digits = "3"
	m.selected = 1
	m, _ = apply(t, m, runeKey(':'))

	esc := tea.KeyMsg{Type: tea.KeyEsc}

	m, _ = apply(t, m, esc)
	if m.promptOpen {
		t.Fatal("layer 1: prompt still open")
	}
	if !m.loading || m.digits == "" || m.selected == 0 {
		t.Fatal("layer 1: esc peeled more than the prompt")
	}

	m, _ = apply(t, m, esc)
	if m.loading {
		t.Fatal("layer 2: load not canceled")
	}
	if m.digits == "" || m.selected == 0 {
		t.Fatal("layer 2: esc peeled more than the fetch")
	}

	m, _ = apply(t, m, esc)
	if m.digits != "" {
		t.Fatal("layer 3: digit buffer not cleared")
	}
	if m.selected == 0 {
		t.Fatal("layer 3: esc also cleared the selection")
	}

	m, _ = apply(t, m, esc)
	if m.selected != 0 {
		t.Fatal("layer 4: selection not cleared")
	}

	_, cmd := apply(t, m, esc)
	if producesQuit(cmd) {
		t.Fatal("esc with nothing left to clear quit the program")
	}
}

func TestDigitBufferFollow(t *testing.T) {
	m := testModel(t)
	(&m).setPage("https://a.test/", "[one](https://example.com/one.md)")

	m, _ = apply(t, m, runeKey('4'))
	m, _ = apply(t, m, runeKey('2'))
	if m.digits != "42" {
		t.Fatalf("digits = %q, want 42", m.digits)
	}
	if got := m.statusLeft(); !strings.Contains(got, "follow: 42_") {
		t.Errorf("statusbar = %q, want follow: 42_", got)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.digits != "" {
		t.Error("enter did not consume the digit buffer")
	}
	if !strings.Contains(m.status, "no link 42") {
		t.Errorf("status = %q, want a no-link error", m.status)
	}

	m, _ = apply(t, m, runeKey('1'))
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.loading || m.loadingHost != "example.com" {
		t.Errorf("follow by number did not navigate: loading=%v host=%q", m.loading, m.loadingHost)
	}
}

func TestTabSelectionCycles(t *testing.T) {
	m := testModel(t)
	(&m).setPage("https://a.test/", "first [one](https://example.com/one.md) then [two](https://example.com/two.md)")
	if len(m.markerLines) != 2 {
		t.Fatalf("markerLines = %v, want 2 markers", m.markerLines)
	}

	tab := tea.KeyMsg{Type: tea.KeyTab}
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}

	m, _ = apply(t, m, tab)
	if m.selected != 1 {
		t.Fatalf("first tab selected %d, want 1", m.selected)
	}
	m, _ = apply(t, m, tab)
	if m.selected != 2 {
		t.Fatalf("second tab selected %d, want 2", m.selected)
	}
	m, _ = apply(t, m, tab)
	if m.selected != 1 {
		t.Fatalf("third tab selected %d, want 1 (cycle)", m.selected)
	}
	m, _ = apply(t, m, shiftTab)
	if m.selected != 2 {
		t.Fatalf("shift+tab selected %d, want 2 (cycle back)", m.selected)
	}

	// Enter follows the selection.
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.loading || m.loadingHost != "example.com" {
		t.Errorf("enter on selection did not navigate: loading=%v host=%q", m.loading, m.loadingHost)
	}
}

func TestBackForwardRestoresScroll(t *testing.T) {
	m := testModel(t)
	long := "# A\n\n" + strings.Repeat("line of text\n\n", 120)
	(&m).showDocument("https://a.test/a.md", long)
	m.viewport.SetYOffset(30)
	fracA := m.currentFraction()
	if fracA <= 0 {
		t.Fatal("test setup: page A should be scrolled")
	}

	(&m).showDocument("https://b.test/b.md", "# B")
	if m.viewport.YOffset != 0 {
		t.Fatal("new document did not start at the top")
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.page.url != "https://a.test/a.md" {
		t.Fatalf("back landed on %s", m.page.url)
	}
	want := restoreOffset(fracA, m.viewport.TotalLineCount(), m.viewport.Height)
	if m.viewport.YOffset != want {
		t.Errorf("restored offset = %d, want %d", m.viewport.YOffset, want)
	}

	m, _ = apply(t, m, runeKey(']'))
	if m.page.url != "https://b.test/b.md" {
		t.Errorf("forward landed on %s", m.page.url)
	}

	// B is a back alias (w3m habit).
	m, _ = apply(t, m, runeKey('B'))
	if m.page.url != "https://a.test/a.md" {
		t.Errorf("'B' back alias landed on %s", m.page.url)
	}
}

func TestStartPageIncludesBookmarks(t *testing.T) {
	store := bookmarks.New(t.TempDir())
	if err := store.Add("Example", "https://example.com/llms.txt"); err != nil {
		t.Fatal(err)
	}
	m := New(Options{Config: config.Defaults(), Style: glamour.WithStandardStyle("dark"), Bookmarks: store})
	md := (&m).startMarkdown()
	if !strings.Contains(md, "## Directory") {
		t.Error("start page lost its Directory section")
	}
	if !strings.Contains(md, "## Bookmarks") {
		t.Error("start page missing the Bookmarks section")
	}
	if !strings.Contains(md, "- [Example](https://example.com/llms.txt)") {
		t.Error("start page missing the bookmark entry")
	}

	empty := New(Options{Config: config.Defaults(), Style: glamour.WithStandardStyle("dark"), Bookmarks: bookmarks.New(t.TempDir())})
	if strings.Contains((&empty).startMarkdown(), "## Bookmarks") {
		t.Error("empty store must not add a Bookmarks section")
	}
}

func TestChromeHeight(t *testing.T) {
	if chromeHeight(false) != 1 || chromeHeight(true) != 2 {
		t.Errorf("chromeHeight = %d/%d, want 1/2", chromeHeight(false), chromeHeight(true))
	}
}

func TestFirstH1(t *testing.T) {
	md := "```\n# not a heading\n```\nintro\n# Real Title\n# Second"
	if got := firstH1(md); got != "Real Title" {
		t.Errorf("firstH1 = %q, want Real Title", got)
	}
	if got := firstH1("no headings"); got != "" {
		t.Errorf("firstH1 = %q, want empty", got)
	}
}

var errFake = &fakeError{}

type fakeError struct{}

func (*fakeError) Error() string { return "fake failure" }
