package doc

import (
	"sort"
	"strconv"
)

// Annotate splices a **[n]** marker in front of every annotatable link,
// working by byte offsets in reverse document order so earlier offsets stay
// valid while splicing. The marker text is identical regardless of selected —
// bold gives a constant rendered cell width, and selection highlighting is
// applied post-render by the UI. The selected parameter is part of the
// signature for future use only.
//
// Reference-style links carry no valid offsets (see Link.Annotatable) and get
// no marker, but they keep their document-order Index so numbering matches
// the digit-buffer follow feature.
func Annotate(markdown string, links []Link, selected int) string {
	_ = selected
	spliceable := make([]Link, 0, len(links))
	for _, l := range links {
		if l.Annotatable() && l.End <= len(markdown) {
			spliceable = append(spliceable, l)
		}
	}
	// Links arrive in document order; sort defensively so the reverse splice
	// stays correct even for a caller-filtered or reordered slice.
	sort.Slice(spliceable, func(i, j int) bool { return spliceable[i].Start > spliceable[j].Start })
	out := markdown
	for _, l := range spliceable {
		out = out[:l.Start] + "**[" + strconv.Itoa(l.Index) + "]**" + out[l.Start:]
	}
	return out
}
