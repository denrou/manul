// Package doc parses markdown documents into link-indexed Documents for the
// manul reader. Parsing uses goldmark with extension.GFM so the link dialect
// (including bare-URL linkify) matches glamour's render dialect: bare URLs
// become links in both or neither.
//
// Reference-style links ([text][ref], [ref][], [ref]) are extracted as Links
// so they stay followable, but they carry no valid source offsets
// (Start = End = -1) and are skipped by Annotate in the MVP.
package doc

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Link is one hyperlink of a Document, in document order.
type Link struct {
	Index      int    // 1-based, document order
	Text       string // rendered link text (label for autolinks)
	Dest       string // raw destination as written (or from the reference definition)
	Start, End int    // byte offsets of the full link source span; -1,-1 when unknown
}

// Annotatable reports whether the link carries valid source offsets and can
// receive an Annotate marker. Reference-style links (and any link whose span
// could not be located in the source) report false.
func (l Link) Annotatable() bool {
	return l.Start >= 0 && l.End > l.Start
}

// Document is a parsed markdown page.
type Document struct {
	URL      string // final URL (post-redirect)
	Markdown string // possibly truncated — see Truncate
	Links    []Link
}

// Parse extracts the links of a markdown document. Truncation (see Truncate)
// must happen before Parse so Markdown, Links, and the render all describe
// the same text.
func Parse(rawURL, markdown string) Document {
	source := []byte(markdown)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	root := md.Parser().Parse(text.NewReader(source))
	return Document{URL: rawURL, Markdown: markdown, Links: extractLinks(source, root)}
}

// ResolveLink resolves a link destination against the document URL and
// rejects any resulting non-http(s) URL.
func (d Document) ResolveLink(l Link) (string, error) {
	base, err := url.Parse(d.URL)
	if err != nil {
		return "", fmt.Errorf("invalid document URL %q: %w", d.URL, err)
	}
	ref, err := url.Parse(l.Dest)
	if err != nil {
		return "", fmt.Errorf("invalid link destination %q: %w", l.Dest, err)
	}
	resolved := base.ResolveReference(ref)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q in %q", resolved.Scheme, resolved.String())
	}
	return resolved.String(), nil
}

// extractLinks walks the AST in document order. A cursor tracks how far into
// the source the walk has advanced (via text segments), so autolink labels —
// whose segment goldmark does not export — are located by searching forward
// from the cursor, which keeps duplicate URLs anchored to distinct offsets.
func extractLinks(source []byte, root ast.Node) []Link {
	var links []Link
	cursor := 0
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Text:
			if v.Segment.Stop > cursor {
				cursor = v.Segment.Stop
			}
		case *ast.FencedCodeBlock:
			// Code block content is not exposed as ast.Text nodes, so
			// advance past it explicitly: a URL quoted inside a fence
			// must not anchor a later autolink's span.
			if stop := lastLineStop(v.Lines()); stop > cursor {
				cursor = stop
			}
		case *ast.CodeBlock:
			if stop := lastLineStop(v.Lines()); stop > cursor {
				cursor = stop
			}
		case *ast.HTMLBlock:
			stop := lastLineStop(v.Lines())
			if v.HasClosure() && v.ClosureLine.Stop > stop {
				stop = v.ClosureLine.Stop
			}
			if stop > cursor {
				cursor = stop
			}
		case *ast.RawHTML:
			if v.Segments != nil && v.Segments.Len() > 0 {
				if stop := v.Segments.At(v.Segments.Len() - 1).Stop; stop > cursor {
					cursor = stop
				}
			}
		case *ast.Image:
			// Images are not followable links; their alt text must not
			// advance the cursor past a possible surrounding link span.
			// Still advance past the image's own source span so a URL in
			// its destination cannot anchor a later autolink.
			if _, e, ok := inlineSpan(source, v); ok && e > cursor {
				cursor = e
			}
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			l := Link{Text: string(v.Label(source)), Dest: string(v.URL(source)), Start: -1, End: -1}
			if s, e, ok := autoLinkSpan(source, v, cursor); ok {
				l.Start, l.End = s, e
				if e > cursor {
					cursor = e
				}
			}
			links = append(links, l)
		case *ast.Link:
			l := Link{Text: linkText(source, v), Dest: string(v.Destination), Start: -1, End: -1}
			if s, e, ok := inlineSpan(source, v); ok {
				l.Start, l.End = s, e
			}
			links = append(links, l)
		}
		return ast.WalkContinue, nil
	})
	for i := range links {
		links[i].Index = i + 1
	}
	return links
}

// linkText concatenates the leaf text of a link's subtree.
func linkText(source []byte, n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if t, ok := c.(*ast.Text); ok {
				b.Write(t.Segment.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// autoLinkSpan locates an autolink (<url> form or GFM-linkified bare URL) by
// searching for its label from the walk cursor. The <...> form extends the
// span to include the angle brackets.
func autoLinkSpan(source []byte, n *ast.AutoLink, cursor int) (start, end int, ok bool) {
	label := n.Label(source)
	if len(label) == 0 || cursor < 0 || cursor > len(source) {
		return 0, 0, false
	}
	off := bytes.Index(source[cursor:], label)
	if off < 0 {
		return 0, 0, false
	}
	start = cursor + off
	end = start + len(label)
	if start > 0 && source[start-1] == '<' && end < len(source) && source[end] == '>' {
		start--
		end++
	}
	return start, end, true
}

// lastLineStop returns the source offset just past the last line segment,
// or -1 when the node holds no lines.
func lastLineStop(lines *text.Segments) int {
	if lines == nil || lines.Len() == 0 {
		return -1
	}
	return lines.At(lines.Len() - 1).Stop
}

// inlineSpan computes the full source span of an inline link ([text](dest
// ...)) or image. goldmark's ast.Link has no source position, so the span
// is reconstructed from the text-leaf segments: walk outward over emphasis
// and code-span delimiters to the enclosing brackets, then scan forward past
// the destination (nested parens, optional title). Any mismatch — including
// reference-style links, where ']' is not followed by '(' — reports false.
func inlineSpan(source []byte, n ast.Node) (start, end int, ok bool) {
	minStart, maxStop := textExtent(n)
	if minStart < 0 {
		return 0, 0, false
	}
	i := minStart - 1
	for i >= 0 && isInlineDelim(source[i]) {
		i--
	}
	if i < 0 || source[i] != '[' {
		return 0, 0, false
	}
	start = i
	j := maxStop
	for j < len(source) && isInlineDelim(source[j]) {
		j++
	}
	if j >= len(source) || source[j] != ']' {
		return 0, 0, false
	}
	if j+1 >= len(source) || source[j+1] != '(' {
		return 0, 0, false // reference-style: no offsets in the MVP
	}
	end, ok = scanInlineTail(source, j+2)
	if !ok {
		return 0, 0, false
	}
	return start, end, true
}

// textExtent returns the min start and max stop over the text leaves of a
// subtree, or (-1, -1) when the subtree holds no positioned text.
func textExtent(n ast.Node) (minStart, maxStop int) {
	minStart, maxStop = -1, -1
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if t, ok := c.(*ast.Text); ok {
				if minStart < 0 || t.Segment.Start < minStart {
					minStart = t.Segment.Start
				}
				if t.Segment.Stop > maxStop {
					maxStop = t.Segment.Stop
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return minStart, maxStop
}

func isInlineDelim(c byte) bool {
	return c == '*' || c == '_' || c == '~' || c == '`'
}

func isLinkSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

// scanInlineTail scans an inline link tail starting just after "](": the
// destination (either <...> or a run with balanced parens), an optional
// quoted title, and the closing ')'. Returns the offset just past ')'.
func scanInlineTail(source []byte, i int) (end int, ok bool) {
	for i < len(source) && isLinkSpace(source[i]) {
		i++
	}
	if i >= len(source) {
		return 0, false
	}
	if source[i] == '<' {
		i++
		for i < len(source) && source[i] != '>' && source[i] != '\n' {
			if source[i] == '\\' && i+1 < len(source) {
				i++
			}
			i++
		}
		if i >= len(source) || source[i] != '>' {
			return 0, false
		}
		i++
	} else {
		depth := 0
		for i < len(source) {
			c := source[i]
			if c == '\\' && i+1 < len(source) {
				i += 2
				continue
			}
			if c == '(' {
				depth++
			} else if c == ')' {
				if depth == 0 {
					break
				}
				depth--
			} else if isLinkSpace(c) {
				break
			}
			i++
		}
	}
	for i < len(source) && isLinkSpace(source[i]) {
		i++
	}
	if i >= len(source) {
		return 0, false
	}
	if q := source[i]; q == '"' || q == '\'' || q == '(' {
		closer := q
		if q == '(' {
			closer = ')'
		}
		i++
		for i < len(source) && source[i] != closer {
			if source[i] == '\\' && i+1 < len(source) {
				i++
			}
			i++
		}
		if i >= len(source) {
			return 0, false
		}
		i++
		for i < len(source) && isLinkSpace(source[i]) {
			i++
		}
	}
	if i >= len(source) || source[i] != ')' {
		return 0, false
	}
	return i + 1, true
}
