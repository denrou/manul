package doc

import (
	"strings"
	"unicode/utf8"
)

// TruncationNotice is the visible line appended to a truncated document.
const TruncationNotice = "*Document truncated — showing the beginning only.*"

// Truncate cuts markdown at the last newline before max bytes (falling back
// to the nearest rune boundary when no newline exists), closes an odd number
// of ``` fences so the cut cannot leak an open code block into the rest of
// the page, and appends TruncationNotice. It reports whether anything was
// cut. Truncate must run before Parse so offsets describe the final text.
func Truncate(markdown string, max int) (out string, truncated bool) {
	if len(markdown) <= max {
		return markdown, false
	}
	if max < 0 {
		max = 0
	}
	cut := markdown[:max]
	if idx := strings.LastIndexByte(cut, '\n'); idx >= 0 {
		cut = cut[:idx]
	} else {
		for len(cut) > 0 && !utf8.RuneStart(markdown[len(cut)]) {
			cut = cut[:len(cut)-1]
		}
	}
	if fenceCount(cut)%2 == 1 {
		cut += "\n```"
	}
	return cut + "\n\n" + TruncationNotice + "\n", true
}

// fenceCount counts lines opening or closing a ``` code fence (up to three
// leading spaces allowed, per CommonMark).
func fenceCount(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && strings.HasPrefix(trimmed, "```") {
			count++
		}
	}
	return count
}
