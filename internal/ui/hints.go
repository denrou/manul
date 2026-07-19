package ui

import (
	"sort"
	"strings"
)

// hintAlphabet orders letters by finger reachability, vimium-style:
// home row first, then the surrounding rows.
const hintAlphabet = "asdfghjklqwertyuiopzxcvbnm"

// hintLabels returns n distinct labels: single letters while they last,
// fixed-length combinations beyond that (all same length, so no label
// is a prefix of another and matching needs no timeout).
func hintLabels(n int) []string {
	if n <= 0 {
		return nil
	}
	if n <= len(hintAlphabet) {
		labels := make([]string, n)
		for i := range labels {
			labels[i] = string(hintAlphabet[i])
		}
		return labels
	}
	width := 2
	for pow := len(hintAlphabet) * len(hintAlphabet); pow < n; pow *= len(hintAlphabet) {
		width++
	}
	labels := make([]string, n)
	for i := range labels {
		v := i
		b := make([]byte, width)
		for p := width - 1; p >= 0; p-- {
			b[p] = hintAlphabet[v%len(hintAlphabet)]
			v /= len(hintAlphabet)
		}
		labels[i] = string(b)
	}
	return labels
}

// hintTarget is one hinted link: its label and where its marker lives
// in the rendered string.
type hintTarget struct {
	label string
	index int // link index (the number the marker normally shows)
	loc   markerLoc
}

// visibleHints assigns labels to the links whose markers are on screen,
// in top-to-bottom order.
func visibleHints(locs map[int]markerLoc, top, height int) []hintTarget {
	if height <= 0 {
		return nil
	}
	var targets []hintTarget
	for idx, loc := range locs {
		if loc.line >= top && loc.line <= top+height-1 {
			targets = append(targets, hintTarget{index: idx, loc: loc})
		}
	}
	sort.Slice(targets, func(a, b int) bool { return targets[a].loc.start < targets[b].loc.start })
	labels := hintLabels(len(targets))
	for i := range targets {
		targets[i].label = labels[i]
	}
	return targets
}

// renderHints replaces each hinted marker in the rendered string with
// its reverse-video letter label, padded to the marker's visible width
// so on-screen columns stay put whenever the label fits.
func renderHints(rendered string, targets []hintTarget) string {
	if len(targets) == 0 {
		return rendered
	}
	var b strings.Builder
	b.Grow(len(rendered))
	pos := 0
	for _, t := range targets {
		if t.loc.start < pos {
			continue
		}
		b.WriteString(rendered[pos:t.loc.start])
		label := "[" + t.label + "]"
		if w := visibleWidth(rendered[t.loc.start:t.loc.end]); w > len(label) {
			label += strings.Repeat(" ", w-len(label))
		}
		b.WriteString(reverseOn)
		b.WriteString(label)
		b.WriteString(reverseOff)
		pos = t.loc.end
	}
	b.WriteString(rendered[pos:])
	return b.String()
}

// visibleWidth counts the non-ANSI bytes of a rendered span (markers
// are ASCII, so bytes equal terminal cells here).
func visibleWidth(span string) int {
	w := 0
	i := 0
	for i < len(span) {
		if isCSIStart(span, i) {
			length, _, _ := readCSI(span, i)
			i += length
			continue
		}
		w++
		i++
	}
	return w
}

// matchHint resolves a typed buffer against the targets: an exact label
// match returns the link index; otherwise it reports whether the buffer
// is still a viable prefix.
func matchHint(targets []hintTarget, buf string) (index int, exact, viable bool) {
	for _, t := range targets {
		if t.label == buf {
			return t.index, true, true
		}
		if strings.HasPrefix(t.label, buf) {
			viable = true
		}
	}
	return 0, false, viable
}
