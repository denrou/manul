package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "bare domain", input: "example.com", want: "https://example.com/llms.txt"},
		{name: "domain with slash", input: "example.com/", want: "https://example.com/llms.txt"},
		{name: "full URL kept", input: "https://example.com/docs/page.md", want: "https://example.com/docs/page.md"},
		{name: "http kept", input: "http://example.com/llms.txt", want: "http://example.com/llms.txt"},
		{name: "surrounding spaces", input: "  example.com  ", want: "https://example.com/llms.txt"},
		{name: "empty", input: "", wantErr: true},
		{name: "unsupported scheme", input: "ftp://example.com", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) = %q, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSniffHTML(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{name: "doctype lowercase", body: "<!doctype html><html>", want: true},
		{name: "doctype uppercase", body: "<!DOCTYPE HTML>", want: true},
		{name: "html tag", body: "<html lang=\"en\">", want: true},
		{name: "html tag uppercase", body: "<HTML>", want: true},
		{name: "leading whitespace", body: "  \n\t<!doctype html>", want: true},
		{name: "utf8 bom then html", body: "\xef\xbb\xbf<html>", want: true},
		{name: "markdown heading", body: "# Title\n\nBody", want: false},
		{name: "html mentioned mid-text", body: "escape <html> in docs", want: false},
		{name: "empty body", body: "", want: false},
		{name: "short non-html", body: "<h1>", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sniffHTML([]byte(tc.body)); got != tc.want {
				t.Fatalf("sniffHTML(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestMarkdownReturnsFinalURLAndBody(t *testing.T) {
	const page = "# Hello\n\nSome markdown."
	mux := http.NewServeMux()
	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "" {
			t.Error("Accept header not sent")
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header not sent")
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte(page))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	finalURL, body, err := Markdown(context.Background(), server.URL+"/llms.txt", 0)
	if err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}
	if body != page {
		t.Errorf("body = %q, want %q", body, page)
	}
	if want := server.URL + "/llms.txt"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
}

func TestMarkdownFollowsRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# Landed"))
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/final.md", http.StatusFound)
	}))
	defer origin.Close()

	finalURL, body, err := Markdown(context.Background(), origin.URL+"/llms.txt", 0)
	if err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}
	if want := target.URL + "/final.md"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
	if body != "# Landed" {
		t.Errorf("body = %q, want %q", body, "# Landed")
	}
}

func TestMarkdownRejectsHTMLContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("does not even look like html"))
	}))
	defer server.Close()

	_, _, err := Markdown(context.Background(), server.URL+"/page", 0)
	if !errors.Is(err, ErrHTML) {
		t.Fatalf("err = %v, want ErrHTML", err)
	}
}

func TestMarkdownRejectsSniffedHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("\n  <!DOCTYPE html><html><body>hi</body></html>"))
	}))
	defer server.Close()

	_, _, err := Markdown(context.Background(), server.URL+"/page", 0)
	if !errors.Is(err, ErrHTML) {
		t.Fatalf("err = %v, want ErrHTML", err)
	}
}

func TestMarkdownNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, _, err := Markdown(context.Background(), server.URL+"/missing.md", 0)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if errors.Is(err, ErrHTML) {
		t.Fatalf("404 misreported as ErrHTML: %v", err)
	}
}
