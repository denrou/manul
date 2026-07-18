package ui

import (
	_ "embed"
	"strings"

	"github.com/denrou/manul/internal/discover"
)

//go:embed welcome.md
var WelcomeMarkdown string

//go:embed help.md
var helpMarkdown string

// Internal pages carry manul: pseudo-URLs; they are rendered from
// embedded or generated markdown and never fetched.
const (
	startURL = "manul:start"
	helpURL  = "manul:help"
)

func isInternal(url string) bool {
	return url == "" || strings.HasPrefix(url, "manul:")
}

// startMarkdown composes the start page: the embedded welcome plus a
// Bookmarks section when the user has any.
func (m *Model) startMarkdown() string {
	md := strings.TrimRight(WelcomeMarkdown, "\n")
	if m.bookmarks == nil {
		return md + "\n"
	}
	list := bookmarkList(m.bookmarks.Markdown())
	if list == "" {
		return md + "\n"
	}
	return md + "\n\n## Bookmarks\n\n" + list
}

// bookmarkList extracts the "- [title](url)" lines from the bookmarks
// page so they can be re-homed under a "## Bookmarks" section.
func bookmarkList(page string) string {
	var b strings.Builder
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- [") {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// fallbackMarkdown is the in-app page shown when discovery found no
// markdown for a URL. It enters history like any document; its URL is
// the original input so 'o' opens the page the user meant.
func fallbackMarkdown(e *discover.NotMarkdownError) string {
	var b strings.Builder
	b.WriteString("# No markdown here\n\n")
	b.WriteString("manul could not find a markdown version of:\n\n")
	b.WriteString("**" + e.URL + "**\n\n")
	b.WriteString(e.Error() + "\n\n")
	b.WriteString("This site probably publishes HTML only (manul never renders HTML).\n\n")
	b.WriteString("- Press `o` to open the page in your system browser.\n")
	b.WriteString("- Press `Backspace` to go back.\n")
	b.WriteString("- Press `:` to try another URL.\n")
	return b.String()
}

// firstH1 returns the text of the first level-1 heading, skipping fenced
// code blocks, or "" when the document has none.
func firstH1(markdown string) string {
	inFence := false
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}
