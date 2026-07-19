package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHintLabels(t *testing.T) {
	if got := hintLabels(3); got[0] != "a" || got[1] != "s" || got[2] != "d" {
		t.Errorf("hintLabels(3) = %v, want home-row order a s d", got)
	}
	if got := hintLabels(26); len(got) != 26 || got[25] != "m" {
		t.Errorf("hintLabels(26) tail = %v", got[len(got)-1:])
	}
	got := hintLabels(27)
	if len(got) != 27 {
		t.Fatalf("hintLabels(27) returned %d labels", len(got))
	}
	seen := map[string]bool{}
	for _, l := range got {
		if len(l) != 2 {
			t.Fatalf("label %q not fixed-width 2", l)
		}
		if seen[l] {
			t.Fatalf("duplicate label %q", l)
		}
		seen[l] = true
	}
	if got := hintLabels(700); len(got) != 700 || len(got[0]) != 3 {
		t.Errorf("hintLabels(700): len=%d width=%d, want 700 labels of width 3", len(got), len(got[0]))
	}
}

func TestVisibleHintsFiltersAndOrders(t *testing.T) {
	locs := map[int]markerLoc{
		1: {line: 2, start: 20, end: 25},
		2: {line: 50, start: 500, end: 505}, // below the fold
		3: {line: 0, start: 3, end: 8},
	}
	targets := visibleHints(locs, 0, 24)
	if len(targets) != 2 {
		t.Fatalf("got %d targets, want 2 visible", len(targets))
	}
	if targets[0].index != 3 || targets[1].index != 1 {
		t.Errorf("order = [%d %d], want top-to-bottom [3 1]", targets[0].index, targets[1].index)
	}
	if targets[0].label != "a" || targets[1].label != "s" {
		t.Errorf("labels = [%s %s], want [a s]", targets[0].label, targets[1].label)
	}
}

func TestRenderHintsPadsToMarkerWidth(t *testing.T) {
	rendered := "see [12] here"
	targets := []hintTarget{{label: "a", index: 12, loc: markerLoc{line: 0, start: 4, end: 8}}}
	out := renderHints(rendered, targets)
	plain := stripANSITest(out)
	if !strings.Contains(plain, "see [a]  here") {
		t.Errorf("hint not padded to marker width: %q", plain)
	}
	if !strings.Contains(out, reverseOn+"[a] "+reverseOff) {
		t.Errorf("hint not reverse-video styled: %q", out)
	}
}

func TestMatchHint(t *testing.T) {
	targets := []hintTarget{{label: "aa", index: 1}, {label: "as", index: 2}}
	if _, exact, viable := matchHint(targets, "a"); exact || !viable {
		t.Error("prefix 'a' should be viable, not exact")
	}
	if idx, exact, _ := matchHint(targets, "as"); !exact || idx != 2 {
		t.Errorf("'as' → exact=%v idx=%d, want exact idx 2", exact, idx)
	}
	if _, exact, viable := matchHint(targets, "z"); exact || viable {
		t.Error("'z' should match nothing")
	}
}

func TestHintModeFlow(t *testing.T) {
	m := testModel(t)
	m.setPage("https://a.test/x.md",
		"[one](https://a.test/1.md) and [two](https://a.test/2.md)\n")

	m, _ = apply(t, m, runeKey('f'))
	if !m.hintMode || len(m.hintTargets) != 2 {
		t.Fatalf("hint mode: on=%v targets=%d, want 2", m.hintMode, len(m.hintTargets))
	}
	if view := stripANSITest(m.viewport.View()); !strings.Contains(view, "[a]") || !strings.Contains(view, "[s]") {
		t.Errorf("letter labels not rendered: %q", view)
	}

	// Second label follows link 2 and leaves hint mode.
	m, cmd := apply(t, m, runeKey('s'))
	if m.hintMode {
		t.Error("hint mode still on after following")
	}
	if cmd == nil {
		t.Fatal("following a hint produced no navigation command")
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc}) // cancel the in-flight load

	// Esc restores numbers without following.
	m, _ = apply(t, m, runeKey('f'))
	m, cmd = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.hintMode || cmd != nil {
		t.Errorf("esc: hintMode=%v cmd=%v, want clean exit", m.hintMode, cmd)
	}
	if view := stripANSITest(m.viewport.View()); !strings.Contains(view, "[1]") {
		t.Errorf("numeric markers not restored: %q", view)
	}

	// Unknown letter stays in hint mode with a notice.
	m, _ = apply(t, m, runeKey('f'))
	m, _ = apply(t, m, runeKey('z'))
	if !m.hintMode {
		t.Error("unknown hint letter should not exit hint mode")
	}
	if !strings.Contains(m.statusLeft(), "no hint") {
		t.Errorf("statusLeft = %q, want a no-hint notice", m.statusLeft())
	}

	// Any non-letter key exits hint mode and is handled normally.
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.hintMode {
		t.Error("tab should exit hint mode")
	}
	if m.selected == 0 {
		t.Error("tab after exiting hints should select a link as usual")
	}
}

func TestHintModeNoLinks(t *testing.T) {
	m := testModel(t)
	m.setPage("https://a.test/x.md", "plain text, no links\n")
	m, _ = apply(t, m, runeKey('f'))
	if m.hintMode {
		t.Error("hint mode entered with no links on screen")
	}
	if !strings.Contains(m.statusLeft(), "no links") {
		t.Errorf("statusLeft = %q, want no-links notice", m.statusLeft())
	}
}
