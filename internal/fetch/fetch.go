// Package fetch retrieves markdown documents from the llms.txt web.
package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodySize caps how much of a response we read; llms.txt files are
// small by design, llms-full.txt can reach a few megabytes.
const maxBodySize = 10 << 20 // 10 MiB

// Timeouts for the two fetch situations: short-lived discovery probes
// that may fail fast, and the committed fetch of a known-good URL.
const (
	ProbeTimeout = 8 * time.Second
	FetchTimeout = 30 * time.Second
)

// ErrHTML reports that the server answered with an HTML page instead of
// markdown. Match it with errors.Is; manul never renders HTML (ADR-0007).
var ErrHTML = errors.New("response is HTML, not markdown")

// Normalize turns user input into a fetchable URL:
//   - a bare domain gets an https:// scheme
//   - an empty path defaults to /llms.txt
func Normalize(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty URL")
	}
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", input, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("missing host in %q", input)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/llms.txt"
	}
	return u.String(), nil
}

// Markdown fetches rawURL and returns the final URL after redirects
// alongside the body. It asks for markdown via content negotiation and
// rejects HTML responses with ErrHTML; any other text comes back as-is
// and deciding how to render is the caller's concern. A non-positive
// timeout falls back to FetchTimeout.
func Markdown(ctx context.Context, rawURL string, timeout time.Duration) (finalURL, body string, err error) {
	if timeout <= 0 {
		timeout = FetchTimeout
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "text/markdown, text/plain;q=0.9, */*;q=0.1")
	req.Header.Set("User-Agent", "manul/0.0 (+https://github.com/denrou/manul)")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	finalURL = resp.Request.URL.String()

	if resp.StatusCode != http.StatusOK {
		return finalURL, "", fmt.Errorf("GET %s: %s", finalURL, resp.Status)
	}
	if isHTMLContentType(resp.Header.Get("Content-Type")) {
		return finalURL, "", fmt.Errorf("%s: %w", finalURL, ErrHTML)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return finalURL, "", err
	}
	if sniffHTML(raw) {
		return finalURL, "", fmt.Errorf("%s: %w", finalURL, ErrHTML)
	}
	return finalURL, string(raw), nil
}

func isHTMLContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "text/html" || mediaType == "application/xhtml+xml"
}

// sniffHTML detects an HTML body by its prefix, tolerating a UTF-8 BOM
// and leading whitespace, case-insensitively.
func sniffHTML(body []byte) bool {
	trimmed := bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	trimmed = bytes.TrimLeft(trimmed, " \t\r\n\f\v")
	const longestPrefix = len("<!doctype")
	if len(trimmed) > longestPrefix {
		trimmed = trimmed[:longestPrefix]
	}
	lower := bytes.ToLower(trimmed)
	return bytes.HasPrefix(lower, []byte("<!doctype")) || bytes.HasPrefix(lower, []byte("<html"))
}
