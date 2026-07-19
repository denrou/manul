package ui

import (
	"sort"
	"strings"
	"unicode"
)

// span is a byte range in the rendered string, including any ANSI
// sequences interleaved with the matched text.
type span struct {
	start, end int
}

// searchState is the in-page search: the query, every match in the
// current render, and the match the user is on.
type searchState struct {
	query   string
	matches []span
	lines   []int // 0-based rendered line of each match
	current int
}

func (s searchState) active() bool { return s.query != "" }

// searchFold reports whether query matching should be case-insensitive
// (smart case: any uppercase letter makes the search exact).
func searchFold(query string) bool {
	return strings.IndexFunc(query, unicode.IsUpper) < 0
}

// foldByte lowercases ASCII letters; multi-byte runes participate in
// matching byte-for-byte (ASCII smart case, documented in help).
func foldByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 'a' - 'A'
	}
	return b
}

// matchAcrossANSIFold is matchAcrossANSI with optional ASCII case
// folding. Matches never cross rendered line breaks: wrapped text is
// searched the way it is displayed, like less.
func matchAcrossANSIFold(s string, i int, needle string, fold bool) (end int, ok bool) {
	k := 0
	for i < len(s) && k < len(needle) {
		if isCSIStart(s, i) {
			length, _, _ := readCSI(s, i)
			i += length
			continue
		}
		if s[i] == '\n' {
			return 0, false
		}
		c := s[i]
		if fold {
			c = foldByte(c)
		}
		if c != needle[k] {
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

// findSearchMatches scans the rendered string once and returns every
// non-overlapping occurrence of query, skipping ANSI sequences.
func findSearchMatches(rendered, query string) ([]span, []int) {
	if query == "" {
		return nil, nil
	}
	fold := searchFold(query)
	needle := query
	if fold {
		needle = strings.Map(func(r rune) rune {
			if r >= 'A' && r <= 'Z' {
				return r + 'a' - 'A'
			}
			return r
		}, query)
	}
	var matches []span
	var lines []int
	line := 0
	i := 0
	for i < len(rendered) {
		if isCSIStart(rendered, i) {
			length, _, _ := readCSI(rendered, i)
			i += length
			continue
		}
		if rendered[i] == '\n' {
			line++
			i++
			continue
		}
		first := rendered[i]
		if fold {
			first = foldByte(first)
		}
		if first == needle[0] {
			if end, ok := matchAcrossANSIFold(rendered, i, needle, fold); ok {
				matches = append(matches, span{start: i, end: end})
				lines = append(lines, line)
				i = end
				continue
			}
		}
		i++
	}
	return matches, lines
}

// highlightSpans wraps every span in reverse video in one left-to-right
// pass. Spans must not overlap; they are applied in start order.
func highlightSpans(rendered string, spans []span) string {
	if len(spans) == 0 {
		return rendered
	}
	sorted := make([]span, len(spans))
	copy(sorted, spans)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].start < sorted[b].start })
	var b strings.Builder
	b.Grow(len(rendered) + len(sorted)*2*len(reverseOn))
	pos := 0
	for _, sp := range sorted {
		if sp.start < pos {
			continue // overlapping span (e.g. search hit inside the selected marker)
		}
		b.WriteString(rendered[pos:sp.start])
		seg := rendered[sp.start:sp.end]
		b.WriteString(reverseOn)
		i := 0
		for i < len(seg) {
			if isCSIStart(seg, i) {
				length, _, _ := readCSI(seg, i)
				b.WriteString(seg[i : i+length])
				b.WriteString(reverseOn)
				i += length
				continue
			}
			b.WriteByte(seg[i])
			i++
		}
		b.WriteString(reverseOff)
		pos = sp.end
	}
	b.WriteString(rendered[pos:])
	return b.String()
}

// firstMatchAtOrBelow picks the first match on or after the given
// rendered line, falling back to the first match overall.
func firstMatchAtOrBelow(lines []int, top int) int {
	for i, l := range lines {
		if l >= top {
			return i
		}
	}
	return 0
}
