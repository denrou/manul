package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"

	"github.com/denrou/manul/internal/doc"
)

func TestClampWidth(t *testing.T) {
	cases := []struct {
		term, max, want int
	}{
		{120, 100, 100},
		{80, 100, 80},
		{80, 0, 80},
		{5, 100, minRenderWidth},
	}
	for _, c := range cases {
		if got := clampWidth(c.term, c.max); got != c.want {
			t.Errorf("clampWidth(%d, %d) = %d, want %d", c.term, c.max, got, c.want)
		}
	}
}

func TestApplyBold(t *testing.T) {
	cases := []struct {
		params string
		start  bool
		want   bool
	}{
		{"1", false, true},
		{"38;5;252;1", false, true},
		{"38;5;1", false, false}, // palette index 1, not the bold attribute
		{"38;2;1;1;1", false, false},
		{"0", true, false},
		{"", true, false},
		{"22", true, false},
		{"4", false, false},
	}
	for _, c := range cases {
		if got := applyBold(c.start, c.params); got != c.want {
			t.Errorf("applyBold(%v, %q) = %v, want %v", c.start, c.params, got, c.want)
		}
	}
}

func TestMarkerSpanSplitChunks(t *testing.T) {
	// Real-world glamour output splits the marker across two bold runs:
	// "[" and "1]" each carry their own SGR + reset.
	rendered := "  \x1b[38;5;252mintro \x1b[0m\x1b[38;5;252;1m[\x1b[0m\x1b[38;5;252;1m1]\x1b[0m\x1b[38;5;35;1mdocs\x1b[0m"
	start, end, ok := markerSpan(rendered, 1, true)
	if !ok {
		t.Fatal("markerSpan did not find the split marker")
	}
	span := rendered[start:end]
	if !strings.HasPrefix(span, "[") || !strings.HasSuffix(span, "1]") {
		t.Errorf("span = %q, want it to run from '[' through '1]'", span)
	}
}

func TestMarkerSpanRequiresBold(t *testing.T) {
	if _, _, ok := markerSpan("plain [1] text", 1, true); ok {
		t.Error("found a marker in unstyled text despite requireBold")
	}
	if _, _, ok := markerSpan("\x1b[38;5;1m[1]\x1b[0m", 1, true); ok {
		t.Error("mistook palette color 1 for the bold attribute")
	}
	if _, _, ok := markerSpan("see \x1b[1m[2]\x1b[0m ok", 2, true); !ok {
		t.Error("did not find a plainly bold marker")
	}
}

func TestFindMarkerFallsBackToLiteral(t *testing.T) {
	start, _, ok := findMarker("plain [7] text", 7)
	if !ok || start != 6 {
		t.Errorf("fallback = (%d, %v), want (6, true)", start, ok)
	}
}

func TestHighlightMarkerPreservesText(t *testing.T) {
	rendered := "x \x1b[38;5;252;1m[\x1b[0m\x1b[38;5;252;1m3]\x1b[0m link"
	out, ok := highlightMarker(rendered, 3)
	if !ok {
		t.Fatal("highlightMarker found nothing")
	}
	if !strings.Contains(out, reverseOn) || !strings.Contains(out, reverseOff) {
		t.Error("highlight did not insert reverse-video sequences")
	}
	if stripANSI(out) != stripANSI(rendered) {
		t.Errorf("highlight changed visible text: %q -> %q", stripANSI(rendered), stripANSI(out))
	}
}

func TestHighlightMarkerMissing(t *testing.T) {
	if _, ok := highlightMarker("no markers here", 4); ok {
		t.Error("highlighted a marker that does not exist")
	}
}

func TestMarkerLine(t *testing.T) {
	rendered := "line0\nx \x1b[1m[3]\x1b[0m y\nline2"
	line, ok := markerLine(rendered, 3)
	if !ok || line != 1 {
		t.Errorf("markerLine = (%d, %v), want (1, true)", line, ok)
	}
	if _, ok := markerLine(rendered, 9); ok {
		t.Error("found a line for an absent marker")
	}
}

// TestMarkerSignatureAgainstGlamour guards the post-render highlight
// against changes in glamour's actual ANSI output.
func TestMarkerSignatureAgainstGlamour(t *testing.T) {
	src := "intro [docs](https://example.com/docs.md) and <https://example.com/raw.md> end"
	d := doc.Parse("https://example.com/", src)
	if len(d.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(d.Links))
	}
	annotated := doc.Annotate(src, d.Links, 0)
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(70))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := r.Render(annotated)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 2; n++ {
		if _, _, ok := markerSpan(rendered, n, true); !ok {
			t.Errorf("bold marker [%d] not found in real glamour output %q", n, rendered)
		}
	}
	lines := markerLineIndex(rendered, d.Links)
	if len(lines) != 2 {
		t.Errorf("markerLineIndex found %d markers, want 2", len(lines))
	}
}

func TestPickersAndCycle(t *testing.T) {
	lines := map[int]int{1: 2, 2: 10, 3: 25}
	ids := sortedIndices(lines)

	if got := pickFirstAtOrBelow(ids, lines, 5); got != 2 {
		t.Errorf("pickFirstAtOrBelow(5) = %d, want 2", got)
	}
	if got := pickFirstAtOrBelow(ids, lines, 30); got != 1 {
		t.Errorf("pickFirstAtOrBelow(30) should wrap to 1, got %d", got)
	}
	if got := pickLastAtOrAbove(ids, lines, 12); got != 2 {
		t.Errorf("pickLastAtOrAbove(12) = %d, want 2", got)
	}
	if got := pickLastAtOrAbove(ids, lines, 1); got != 3 {
		t.Errorf("pickLastAtOrAbove(1) should wrap to 3, got %d", got)
	}
	if got := cycleNext(ids, 3); got != 1 {
		t.Errorf("cycleNext(3) = %d, want 1", got)
	}
	if got := cyclePrev(ids, 1); got != 3 {
		t.Errorf("cyclePrev(1) = %d, want 3", got)
	}
	if got := cycleNext(ids, 42); got != 1 {
		t.Errorf("cycleNext(unknown) = %d, want 1", got)
	}
}

func TestMinimalScroll(t *testing.T) {
	cases := []struct {
		offset, height, line, want int
	}{
		{10, 5, 3, 3},   // above: jump to the line
		{10, 5, 20, 16}, // below: line becomes the last visible row
		{10, 5, 12, 10}, // visible: no movement
		{10, 0, 12, 10}, // degenerate height
		{0, 24, 40, 17}, // below on first screen
	}
	for _, c := range cases {
		if got := minimalScroll(c.offset, c.height, c.line); got != c.want {
			t.Errorf("minimalScroll(%d, %d, %d) = %d, want %d", c.offset, c.height, c.line, got, c.want)
		}
	}
}

func stripANSI(s string) string {
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
