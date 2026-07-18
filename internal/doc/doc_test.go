package doc

import (
	"strings"
	"testing"
)

// span returns the source text covered by a link's offsets.
func span(t *testing.T, md string, l Link) string {
	t.Helper()
	if !l.Annotatable() {
		t.Fatalf("link %d (%q -> %q) has no valid offsets: [%d,%d)", l.Index, l.Text, l.Dest, l.Start, l.End)
	}
	if l.Start < 0 || l.End > len(md) || l.Start >= l.End {
		t.Fatalf("link %d offsets out of range: [%d,%d) len=%d", l.Index, l.Start, l.End, len(md))
	}
	return md[l.Start:l.End]
}

func TestParseInlineLinkOffsets(t *testing.T) {
	md := "See [docs](https://example.com/docs) for details."
	d := Parse("https://example.com/", md)

	if len(d.Links) != 1 {
		t.Fatalf("got %d links, want 1", len(d.Links))
	}
	l := d.Links[0]
	if l.Index != 1 {
		t.Errorf("Index = %d, want 1", l.Index)
	}
	if l.Text != "docs" {
		t.Errorf("Text = %q, want %q", l.Text, "docs")
	}
	if l.Dest != "https://example.com/docs" {
		t.Errorf("Dest = %q", l.Dest)
	}
	if got := span(t, md, l); got != "[docs](https://example.com/docs)" {
		t.Errorf("span = %q", got)
	}
}

func TestParseDuplicateLinks(t *testing.T) {
	md := "See [docs](https://example.com/d) and again [docs](https://example.com/d)."
	d := Parse("https://example.com/", md)

	if len(d.Links) != 2 {
		t.Fatalf("got %d links, want 2", len(d.Links))
	}
	for i, l := range d.Links {
		if l.Index != i+1 {
			t.Errorf("link %d: Index = %d, want %d", i, l.Index, i+1)
		}
		if got := span(t, md, l); got != "[docs](https://example.com/d)" {
			t.Errorf("link %d: span = %q", i, got)
		}
	}
	if d.Links[0].Start >= d.Links[1].Start {
		t.Errorf("duplicate links must have distinct ascending offsets: %d vs %d",
			d.Links[0].Start, d.Links[1].Start)
	}
}

func TestParseReferenceLinks(t *testing.T) {
	md := "Full [guide][g], collapsed [g][], shortcut [g].\n\n[g]: https://example.com/g\n"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 3 {
		t.Fatalf("got %d links, want 3", len(d.Links))
	}
	for i, l := range d.Links {
		if l.Dest != "https://example.com/g" {
			t.Errorf("link %d: Dest = %q", i, l.Dest)
		}
		if l.Annotatable() {
			t.Errorf("link %d: reference-style link must not be annotatable, got [%d,%d)",
				i, l.Start, l.End)
		}
	}
	// Reference links are skipped by Annotate: the source stays unchanged.
	if got := Annotate(md, d.Links, 0); got != md {
		t.Errorf("Annotate changed a reference-only document:\n%q", got)
	}
}

func TestParseAutolinksAndBareURLs(t *testing.T) {
	md := "Angle <https://example.com/a> bare https://example.com/b www www.example.com end"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 3 {
		t.Fatalf("got %d links, want 3: %+v", len(d.Links), d.Links)
	}
	if got := span(t, md, d.Links[0]); got != "<https://example.com/a>" {
		t.Errorf("angle autolink span = %q", got)
	}
	if d.Links[0].Dest != "https://example.com/a" {
		t.Errorf("angle autolink Dest = %q", d.Links[0].Dest)
	}
	if got := span(t, md, d.Links[1]); got != "https://example.com/b" {
		t.Errorf("bare URL span = %q", got)
	}
	if d.Links[1].Dest != "https://example.com/b" {
		t.Errorf("bare URL Dest = %q", d.Links[1].Dest)
	}
	if got := span(t, md, d.Links[2]); got != "www.example.com" {
		t.Errorf("www link span = %q", got)
	}
	if d.Links[2].Dest != "http://www.example.com" {
		t.Errorf("www link Dest = %q", d.Links[2].Dest)
	}
}

func TestParseDuplicateBareURLs(t *testing.T) {
	md := "first https://example.com/x then https://example.com/x again"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 2 {
		t.Fatalf("got %d links, want 2", len(d.Links))
	}
	if d.Links[0].Start >= d.Links[1].Start {
		t.Errorf("duplicate bare URLs must resolve to distinct offsets: %d vs %d",
			d.Links[0].Start, d.Links[1].Start)
	}
	for i, l := range d.Links {
		if got := span(t, md, l); got != "https://example.com/x" {
			t.Errorf("link %d: span = %q", i, got)
		}
	}
}

func TestParseLinksInStructures(t *testing.T) {
	md := "# Head [h](https://example.com/h)\n" +
		"\n" +
		"- item [i](https://example.com/i)\n" +
		"\n" +
		"| a | b |\n" +
		"| - | - |\n" +
		"| x [t](https://example.com/t) | y |\n"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 3 {
		t.Fatalf("got %d links, want 3: %+v", len(d.Links), d.Links)
	}
	wantSpans := []string{
		"[h](https://example.com/h)",
		"[i](https://example.com/i)",
		"[t](https://example.com/t)",
	}
	for i, want := range wantSpans {
		if got := span(t, md, d.Links[i]); got != want {
			t.Errorf("link %d: span = %q, want %q", i, got, want)
		}
	}
}

func TestParseLiteralBracketNumberText(t *testing.T) {
	md := "Notes [3] are cited.\n\nA [real](https://example.com/r) link.\n"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 1 {
		t.Fatalf("got %d links, want 1: %+v", len(d.Links), d.Links)
	}
	if got := span(t, md, d.Links[0]); got != "[real](https://example.com/r)" {
		t.Errorf("span = %q", got)
	}
	annotated := Annotate(md, d.Links, 0)
	if !strings.Contains(annotated, "Notes [3] are cited.") {
		t.Errorf("literal [3] text was corrupted:\n%q", annotated)
	}
	if !strings.Contains(annotated, "**[1]**[real](https://example.com/r)") {
		t.Errorf("marker missing or misplaced:\n%q", annotated)
	}
}

func TestParseUnicodeBeforeLinkOffsets(t *testing.T) {
	md := "héllo wörld — voilà [link](https://example.com/u) and 日本語 [two](https://example.com/2)"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 2 {
		t.Fatalf("got %d links, want 2", len(d.Links))
	}
	if got := span(t, md, d.Links[0]); got != "[link](https://example.com/u)" {
		t.Errorf("link 1 span = %q", got)
	}
	if got := span(t, md, d.Links[1]); got != "[two](https://example.com/2)" {
		t.Errorf("link 2 span = %q", got)
	}
}

func TestParseNestedParensAndTitle(t *testing.T) {
	md := `See [wiki](https://example.com/a_(b)_c) and [t](https://example.com/x "a )title").`
	d := Parse("https://example.com/", md)

	if len(d.Links) != 2 {
		t.Fatalf("got %d links, want 2: %+v", len(d.Links), d.Links)
	}
	if got := span(t, md, d.Links[0]); got != "[wiki](https://example.com/a_(b)_c)" {
		t.Errorf("nested-parens span = %q", got)
	}
	if d.Links[0].Dest != "https://example.com/a_(b)_c" {
		t.Errorf("nested-parens Dest = %q", d.Links[0].Dest)
	}
	if got := span(t, md, d.Links[1]); got != `[t](https://example.com/x "a )title")` {
		t.Errorf("titled span = %q", got)
	}
	if d.Links[1].Dest != "https://example.com/x" {
		t.Errorf("titled Dest = %q", d.Links[1].Dest)
	}
}

func TestParseEmphasizedLinkText(t *testing.T) {
	md := "read [**bold** docs](https://example.com/b) now"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 1 {
		t.Fatalf("got %d links, want 1", len(d.Links))
	}
	if got := span(t, md, d.Links[0]); got != "[**bold** docs](https://example.com/b)" {
		t.Errorf("span = %q", got)
	}
}

func TestAnnotateSplicesMarkers(t *testing.T) {
	md := "A [one](https://example.com/1) B [two](https://example.com/2)"
	d := Parse("https://example.com/", md)

	got := Annotate(md, d.Links, 0)
	want := "A **[1]**[one](https://example.com/1) B **[2]**[two](https://example.com/2)"
	if got != want {
		t.Errorf("Annotate =\n%q\nwant\n%q", got, want)
	}
}

func TestAnnotateIdenticalForAnySelected(t *testing.T) {
	md := "A [one](https://example.com/1) B [two](https://example.com/2) C <https://example.com/3>"
	d := Parse("https://example.com/", md)

	base := Annotate(md, d.Links, 0)
	for _, selected := range []int{-1, 1, 2, 3, 99} {
		if got := Annotate(md, d.Links, selected); got != base {
			t.Errorf("Annotate(selected=%d) differs:\n%q\nvs\n%q", selected, got, base)
		}
	}
}

func TestAnnotateSkipsReferenceLinksKeepsIndexes(t *testing.T) {
	md := "Ref [a][r] then [b](https://example.com/b).\n\n[r]: https://example.com/r\n"
	d := Parse("https://example.com/", md)

	if len(d.Links) != 2 {
		t.Fatalf("got %d links, want 2: %+v", len(d.Links), d.Links)
	}
	annotated := Annotate(md, d.Links, 0)
	if strings.Contains(annotated, "**[1]**") {
		t.Errorf("reference link must not get a marker:\n%q", annotated)
	}
	// Document-order index is preserved even when earlier links are skipped.
	if !strings.Contains(annotated, "**[2]**[b](https://example.com/b)") {
		t.Errorf("inline link marker missing or renumbered:\n%q", annotated)
	}
}

func TestResolveLink(t *testing.T) {
	d := Document{URL: "https://example.com/docs/page.md"}
	cases := []struct {
		name    string
		dest    string
		want    string
		wantErr bool
	}{
		{"relative sibling", "other.md", "https://example.com/docs/other.md", false},
		{"relative parent", "../top.md", "https://example.com/top.md", false},
		{"absolute path", "/abs.md", "https://example.com/abs.md", false},
		{"absolute URL", "https://other.example/x", "https://other.example/x", false},
		{"protocol relative", "//other.example/y", "https://other.example/y", false},
		{"mailto rejected", "mailto:a@example.com", "", true},
		{"ftp rejected", "ftp://example.com/f", "", true},
		{"javascript rejected", "javascript:alert(1)", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.ResolveLink(Link{Dest: tc.dest})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ResolveLink(%q) = %q, want error", tc.dest, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveLink(%q) error: %v", tc.dest, err)
			}
			if got != tc.want {
				t.Errorf("ResolveLink(%q) = %q, want %q", tc.dest, got, tc.want)
			}
		})
	}
}

func TestTruncateNoopWhenShorterThanMax(t *testing.T) {
	md := "short document\nwith lines\n"
	out, truncated := Truncate(md, len(md))
	if truncated {
		t.Error("truncated = true, want false")
	}
	if out != md {
		t.Errorf("out = %q, want input unchanged", out)
	}
}

func TestTruncateCutsAtLastNewline(t *testing.T) {
	md := "line1\nline2\nline3\n"
	out, truncated := Truncate(md, 9) // mid "line2"
	if !truncated {
		t.Fatal("truncated = false, want true")
	}
	if !strings.HasPrefix(out, "line1\n") {
		t.Errorf("out must start with the last complete line: %q", out)
	}
	if strings.Contains(out, "line2") {
		t.Errorf("partial line must be dropped: %q", out)
	}
	if !strings.Contains(out, TruncationNotice) {
		t.Errorf("missing truncation notice: %q", out)
	}
}

func TestTruncateMidFenceClosesFence(t *testing.T) {
	md := "intro\n```go\ncode1\ncode2\ncode3\n```\nafter\n"
	out, truncated := Truncate(md, 20) // lands inside the code block
	if !truncated {
		t.Fatal("truncated = false, want true")
	}
	fences := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			fences++
		}
	}
	if fences%2 != 0 {
		t.Errorf("odd fence count %d in output:\n%q", fences, out)
	}
	if !strings.Contains(out, "```go") {
		t.Errorf("opening fence lost: %q", out)
	}
	if !strings.Contains(out, TruncationNotice) {
		t.Errorf("missing truncation notice: %q", out)
	}
	if strings.Contains(out, "after") {
		t.Errorf("content past the cut leaked through: %q", out)
	}
}

func TestTruncateNoNewlineBeforeMaxKeepsRuneBoundary(t *testing.T) {
	md := strings.Repeat("é", 100) // 200 bytes, no newline
	out, truncated := Truncate(md, 101)
	if !truncated {
		t.Fatal("truncated = false, want true")
	}
	cut := strings.SplitN(out, "\n", 2)[0]
	if strings.ContainsRune(cut, '�') || !strings.HasPrefix(md, cut) {
		t.Errorf("cut broke a multi-byte rune: %q", cut)
	}
}

func TestParseDocumentFields(t *testing.T) {
	md := "# Title\n\n[a](https://example.com/a)\n"
	d := Parse("https://example.com/final", md)
	if d.URL != "https://example.com/final" {
		t.Errorf("URL = %q", d.URL)
	}
	if d.Markdown != md {
		t.Errorf("Markdown must be stored verbatim")
	}
}
