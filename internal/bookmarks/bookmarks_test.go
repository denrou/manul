package bookmarks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownMissingFileReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	if got := s.Markdown(); got != "" {
		t.Errorf("Markdown() on missing file = %q, want \"\"", got)
	}
}

func TestAddCreatesFileAndRenders(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Add("Anthropic Docs", "https://docs.anthropic.com/llms.txt"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	want := "# Bookmarks\n\n- [Anthropic Docs](https://docs.anthropic.com/llms.txt)\n"
	if got := s.Markdown(); got != want {
		t.Errorf("Markdown() = %q, want %q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, "bookmarks.md"))
	if err != nil {
		t.Fatalf("bookmarks.md not written: %v", err)
	}
	if string(data) != want {
		t.Errorf("file content = %q, want %q", string(data), want)
	}
}

func TestAddCreatesMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "manul")
	s := New(dir)
	if err := s.Add("Example", "https://example.com/llms.txt"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if s.Markdown() == "" {
		t.Error("Markdown() empty after Add into missing directory")
	}
}

func TestAddDedupesByURL(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Add("First title", "https://example.com/a.md"); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("Second title", "https://example.com/a.md"); err != nil {
		t.Fatal(err)
	}
	got := s.Markdown()
	if n := strings.Count(got, "https://example.com/a.md"); n != 1 {
		t.Errorf("URL appears %d times, want 1:\n%s", n, got)
	}
	if !strings.Contains(got, "[First title]") {
		t.Errorf("original title lost on dedupe:\n%s", got)
	}
	if strings.Contains(got, "[Second title]") {
		t.Errorf("duplicate Add changed title:\n%s", got)
	}
}

func TestMarkdownRendersMultipleEntriesInOrder(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Add("A", "https://a.example/llms.txt"); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("B", "https://b.example/llms.txt"); err != nil {
		t.Fatal(err)
	}
	want := "# Bookmarks\n\n" +
		"- [A](https://a.example/llms.txt)\n" +
		"- [B](https://b.example/llms.txt)\n"
	if got := s.Markdown(); got != want {
		t.Errorf("Markdown() = %q, want %q", got, want)
	}
}

func TestMarkdownEmptyWhenFileHasNoEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bookmarks.md"), []byte("# Bookmarks\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	if got := s.Markdown(); got != "" {
		t.Errorf("Markdown() = %q, want \"\" for entry-less page", got)
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if want := filepath.Join(dir, "bookmarks.md"); s.Path() != want {
		t.Errorf("Path() = %q, want %q", s.Path(), want)
	}
}
