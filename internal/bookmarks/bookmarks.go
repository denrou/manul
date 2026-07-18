// Package bookmarks persists the user's bookmarks as a real markdown
// page (<configdir>/bookmarks.md) that manul itself can render.
package bookmarks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const heading = "# Bookmarks"

// Bookmark is one saved page.
type Bookmark struct {
	Title string
	URL   string
}

// Store reads and writes the bookmarks.md page. Concurrency is not a
// concern: manul is a single-process application.
type Store struct {
	path string
}

// New returns a Store persisting to <dir>/bookmarks.md, where dir is
// typically config.Dir().
func New(dir string) *Store {
	return &Store{path: filepath.Join(dir, "bookmarks.md")}
}

// Path returns the location of the bookmarks page on disk.
func (s *Store) Path() string {
	return s.path
}

// Add appends a bookmark, creating the file (and its directory) if
// missing. Bookmarks are deduplicated by URL: adding an existing URL is
// a no-op, keeping the original title. The new entry is appended to the
// existing file content rather than re-rendering it, so hand-edited
// lines the parser does not recognize survive — the file is a real
// markdown page the user may edit.
func (s *Store) Add(title, url string) error {
	data, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	content := string(data)
	for _, b := range parse(content) {
		if b.URL == url {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	entry := fmt.Sprintf("- [%s](%s)\n", title, url)
	if strings.TrimSpace(content) == "" {
		content = heading + "\n\n"
	} else if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(s.path, []byte(content+entry), 0o644)
}

// Markdown returns the bookmarks page text, or "" when no bookmarks
// exist (missing file or a page with no entries). A read failure is
// reported instead of being mistaken for an empty store.
func (s *Store) Markdown() (string, error) {
	entries, err := s.list()
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	return render(entries), nil
}

func render(entries []Bookmark) string {
	var b strings.Builder
	b.WriteString(heading + "\n\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s](%s)\n", e.Title, e.URL)
	}
	return b.String()
}

// list parses the on-disk page back into entries. A missing file is an
// empty store, not an error.
func (s *Store) list() ([]Bookmark, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return parse(string(data)), nil
}

// parse extracts the recognized bookmark lines of a page.
func parse(content string) []Bookmark {
	var entries []Bookmark
	for _, line := range strings.Split(content, "\n") {
		b, ok := parseLine(line)
		if ok {
			entries = append(entries, b)
		}
	}
	return entries
}

// parseLine matches "- [title](url)". The separator is the LAST "](" on
// the line so titles containing "](" cannot truncate the URL.
func parseLine(line string) (Bookmark, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "- [") || !strings.HasSuffix(line, ")") {
		return Bookmark{}, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "- ["), ")")
	sep := strings.LastIndex(inner, "](")
	if sep < 0 {
		return Bookmark{}, false
	}
	return Bookmark{Title: inner[:sep], URL: inner[sep+2:]}, true
}
