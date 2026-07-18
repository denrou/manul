package ui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/denrou/manul/internal/config"
	"github.com/denrou/manul/internal/doc"
)

// maxRenderBytes caps how much markdown goes through the interactive
// render pipeline (Truncate runs before Parse so links and render agree).
const maxRenderBytes = 1 << 20 // 1 MiB

// minRenderWidth keeps glamour from degenerate wrapping on tiny terminals.
const minRenderWidth = 20

// ResolveStyle maps the user configuration to a glamour style option.
// It must be called once in main() before the TUI takes over the
// terminal: background detection is unreliable after that.
func ResolveStyle(cfg config.Config) glamour.TermRendererOption {
	if cfg.StylePath != "" {
		return glamour.WithStylesFromJSONFile(cfg.StylePath)
	}
	switch cfg.Theme {
	case "dark", "light", "notty":
		return glamour.WithStandardStyle(cfg.Theme)
	}
	// "auto" and unknown values fall back to background detection.
	if lipgloss.HasDarkBackground() {
		return glamour.WithStandardStyle("dark")
	}
	return glamour.WithStandardStyle("light")
}

// clampWidth is the wrap width for a terminal width: capped by the
// configured max width, floored so rendering stays sane.
func clampWidth(termWidth, maxWidth int) int {
	w := termWidth
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if w < minRenderWidth {
		w = minRenderWidth
	}
	return w
}

const (
	reverseOn  = "\x1b[7m"
	reverseOff = "\x1b[27m"
)

func markerNeedle(n int) string {
	return "[" + strconv.Itoa(n) + "]"
}

// markerSpan locates the rendered form of the **[n]** marker inside
// glamour output. Glamour may split the marker text across several
// styled chunks (for example "[" and "1]" as separate SGR runs), so the
// needle is matched across interleaved ANSI sequences, and — when
// requireBold is set — only where the SGR state in effect at the opening
// bracket has bold enabled, which is the signature of the **...**
// marker. The returned span includes any ANSI sequences between the
// marker's characters.
func markerSpan(rendered string, n int, requireBold bool) (start, end int, ok bool) {
	needle := markerNeedle(n)
	bold := false
	i := 0
	for i < len(rendered) {
		if isCSIStart(rendered, i) {
			length, params, sgr := readCSI(rendered, i)
			if sgr {
				bold = applyBold(bold, params)
			}
			i += length
			continue
		}
		if rendered[i] == needle[0] && (bold || !requireBold) {
			if e, matched := matchAcrossANSI(rendered, i, needle); matched {
				return i, e, true
			}
		}
		i++
	}
	return 0, 0, false
}

// findMarker is markerSpan with the fallback the spec allows: when the
// bold ANSI signature is absent (plain styles such as notty), fall back
// to the first literal occurrence of the marker text.
func findMarker(rendered string, n int) (start, end int, ok bool) {
	if s, e, found := markerSpan(rendered, n, true); found {
		return s, e, true
	}
	return markerSpan(rendered, n, false)
}

// highlightMarker restyles the nth link marker with reverse video on top
// of the cached rendered string — no glamour pass per selection change.
func highlightMarker(rendered string, n int) (string, bool) {
	start, end, ok := findMarker(rendered, n)
	if !ok {
		return rendered, false
	}
	return highlightSpan(rendered, start, end), true
}

// highlightSpan wraps rendered[start:end] in reverse video. Reverse video
// is re-asserted after every ANSI sequence inside the span because
// glamour resets styling between its chunks.
func highlightSpan(rendered string, start, end int) string {
	span := rendered[start:end]
	var b strings.Builder
	b.Grow(len(rendered) + 4*len(reverseOn))
	b.WriteString(rendered[:start])
	b.WriteString(reverseOn)
	i := 0
	for i < len(span) {
		if isCSIStart(span, i) {
			length, _, _ := readCSI(span, i)
			b.WriteString(span[i : i+length])
			b.WriteString(reverseOn)
			i += length
			continue
		}
		b.WriteByte(span[i])
		i++
	}
	b.WriteString(reverseOff)
	b.WriteString(rendered[end:])
	return b.String()
}

// markerLine returns the 0-based rendered line holding link n's marker.
func markerLine(rendered string, n int) (int, bool) {
	start, _, ok := findMarker(rendered, n)
	if !ok {
		return 0, false
	}
	return strings.Count(rendered[:start], "\n"), true
}

// markerLoc is the rendered location of one link marker.
type markerLoc struct {
	line       int // 0-based rendered line
	start, end int // byte span, including interleaved ANSI sequences
}

// markerHit is one "[n]" occurrence found while scanning rendered output.
type markerHit struct {
	index int
	loc   markerLoc
	bold  bool
}

// markerIndex maps each annotatable link index to its rendered marker
// location using a single left-to-right scan of the rendered string —
// O(rendered bytes), independent of the link count, so link-heavy
// documents (llms-full.txt) index in milliseconds.
//
// Annotate emits markers in document order, so occurrences are resolved
// in ascending index order with a moving position: link n's marker must
// come after link n-1's. That also keeps a bold literal like "# Notes
// [2]" from shadowing the real **[2]** marker further down. Bold
// occurrences (the **[n]** signature) win over plain ones; plain
// occurrences are the spec-allowed fallback for styles without bold.
func markerIndex(rendered string, links []doc.Link) map[int]markerLoc {
	wanted := make(map[int]bool, len(links))
	order := make([]int, 0, len(links))
	for _, l := range links {
		if l.Annotatable() {
			wanted[l.Index] = true
			order = append(order, l.Index) // document order: ascending
		}
	}
	if len(order) == 0 {
		return nil
	}
	hits := scanMarkers(rendered, wanted)
	out := make(map[int]markerLoc, len(order))
	pos := 0
	lo := 0
	for _, idx := range order {
		for lo < len(hits) && hits[lo].loc.start < pos {
			lo++
		}
		pick := -1
		for h := lo; h < len(hits); h++ {
			if hits[h].index != idx {
				continue
			}
			if pick < 0 {
				pick = h
			}
			if hits[h].bold {
				pick = h
				break
			}
		}
		if pick < 0 {
			continue // marker lost in rendering; digits still work
		}
		out[idx] = hits[pick].loc
		pos = hits[pick].loc.end
	}
	return out
}

// scanMarkers finds every "[n]" occurrence whose n is in wanted, in one
// ANSI-state-tracking pass, recording line, span, and the bold state in
// effect at the opening bracket.
func scanMarkers(rendered string, wanted map[int]bool) []markerHit {
	var hits []markerHit
	bold := false
	line := 0
	i := 0
	for i < len(rendered) {
		if isCSIStart(rendered, i) {
			length, params, sgr := readCSI(rendered, i)
			if sgr {
				bold = applyBold(bold, params)
			}
			i += length
			continue
		}
		switch rendered[i] {
		case '\n':
			line++
		case '[':
			if n, end, ok := matchIndexMarker(rendered, i); ok && wanted[n] {
				hits = append(hits, markerHit{
					index: n,
					loc:   markerLoc{line: line, start: i, end: end},
					bold:  bold,
				})
			}
		}
		i++
	}
	return hits
}

// matchIndexMarker matches "[<digits>]" at the '[' at i, skipping ANSI
// CSI sequences between characters, returning the number and the offset
// just past ']'.
func matchIndexMarker(s string, i int) (n, end int, ok bool) {
	digits := 0
	j := i + 1
	for j < len(s) {
		if isCSIStart(s, j) {
			length, _, _ := readCSI(s, j)
			j += length
			continue
		}
		c := s[j]
		switch {
		case c >= '0' && c <= '9':
			if digits >= 7 {
				return 0, 0, false
			}
			n = n*10 + int(c-'0')
			digits++
			j++
		case c == ']' && digits > 0:
			return n, j + 1, true
		default:
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// markerLineIndex maps each annotatable link index to its rendered line.
func markerLineIndex(rendered string, links []doc.Link) map[int]int {
	locs := markerIndex(rendered, links)
	out := make(map[int]int, len(locs))
	for idx, loc := range locs {
		out[idx] = loc.line
	}
	return out
}

func isCSIStart(s string, i int) bool {
	return s[i] == 0x1b && i+1 < len(s) && s[i+1] == '['
}

// readCSI returns the byte length of the CSI sequence starting at i, its
// parameter body, and whether it is an SGR sequence (final byte 'm').
func readCSI(s string, i int) (length int, params string, sgr bool) {
	j := i + 2
	for j < len(s) {
		c := s[j]
		if c >= 0x40 && c <= 0x7e { // final byte
			return j - i + 1, s[i+2 : j], c == 'm'
		}
		j++
	}
	return len(s) - i, "", false
}

// matchAcrossANSI matches needle at i, skipping ANSI CSI sequences
// between characters, and returns the index just past the last byte.
func matchAcrossANSI(s string, i int, needle string) (end int, ok bool) {
	k := 0
	for i < len(s) && k < len(needle) {
		if isCSIStart(s, i) {
			length, _, _ := readCSI(s, i)
			i += length
			continue
		}
		if s[i] != needle[k] {
			return 0, false
		}
		i++
		k++
	}
	if k == len(needle) {
		return i, true
	}
	return 0, false
}

// applyBold folds one SGR parameter list into the bold state. Extended
// color introducers (38/48/58) consume their arguments so a palette
// index of 1 is not mistaken for the bold attribute.
func applyBold(bold bool, params string) bool {
	if params == "" {
		return false // \x1b[m is a reset
	}
	tokens := strings.Split(params, ";")
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "", "0":
			bold = false
		case "1":
			bold = true
		case "21", "22":
			bold = false
		case "38", "48", "58":
			if i+1 < len(tokens) {
				switch tokens[i+1] {
				case "5":
					i += 2
				case "2":
					i += 4
				}
			}
		}
	}
	return bold
}

// sortedIndices returns the link indices of a marker-line map in
// document order.
func sortedIndices(lines map[int]int) []int {
	ids := make([]int, 0, len(lines))
	for id := range lines {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// pickFirstAtOrBelow returns the first link whose marker line is at or
// below top, wrapping to the first link when none is.
func pickFirstAtOrBelow(ids []int, lines map[int]int, top int) int {
	for _, id := range ids {
		if lines[id] >= top {
			return id
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// pickLastAtOrAbove returns the last link whose marker line is at or
// above bottom, wrapping to the last link when none is.
func pickLastAtOrAbove(ids []int, lines map[int]int, bottom int) int {
	for i := len(ids) - 1; i >= 0; i-- {
		if lines[ids[i]] <= bottom {
			return ids[i]
		}
	}
	if len(ids) > 0 {
		return ids[len(ids)-1]
	}
	return 0
}

func cycleNext(ids []int, current int) int {
	for i, id := range ids {
		if id == current {
			return ids[(i+1)%len(ids)]
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func cyclePrev(ids []int, current int) int {
	for i, id := range ids {
		if id == current {
			return ids[(i-1+len(ids))%len(ids)]
		}
	}
	if len(ids) > 0 {
		return ids[len(ids)-1]
	}
	return 0
}

// minimalScroll returns the smallest viewport offset change that makes
// line visible in a window of the given height.
func minimalScroll(offset, height, line int) int {
	if height <= 0 {
		return offset
	}
	if line < offset {
		return line
	}
	if line > offset+height-1 {
		return line - height + 1
	}
	return offset
}
