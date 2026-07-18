// Package fetch retrieves markdown documents from the llms.txt web.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodySize caps how much of a response we read; llms.txt files are
// small by design, llms-full.txt can reach a few megabytes.
const maxBodySize = 10 << 20 // 10 MiB

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

// Markdown fetches rawURL and returns its body as a string. It asks for
// markdown via content negotiation but accepts whatever text comes back;
// deciding how to render is the caller's concern.
func Markdown(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/markdown, text/plain;q=0.9, */*;q=0.1")
	req.Header.Set("User-Agent", "manul/0.0 (+https://github.com/denrou/manul)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", rawURL, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
