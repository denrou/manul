// Package discover maps arbitrary web URLs to their markdown twins by
// probing candidates in a fixed order (ADR-0006).
package discover

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/denrou/manul/internal/fetch"
)

// NotMarkdownError reports that no probe for a URL yielded markdown.
// URL is the original normalized input URL (what the user asked for),
// not the last probed candidate, so the fallback page and the "open in
// browser" action target the page the user meant.
type NotMarkdownError struct {
	URL string

	candidates []string
}

func (e *NotMarkdownError) Error() string {
	if len(e.candidates) == 0 {
		return fmt.Sprintf("no markdown found at %s", e.URL)
	}
	return fmt.Sprintf("no markdown found at %s (probed: %s)", e.URL, strings.Join(e.candidates, ", "))
}

// pattern identifies which probe strategy produced markdown for a host.
type pattern int

const (
	patternLLMS pattern = iota
	patternLLMSFull
	patternAsIs
	patternNegotiate
	patternAppendMD
	patternIndexMD
)

type probe struct {
	url string
	pat pattern
}

// Resolver resolves user input to markdown, remembering per host which
// probe strategy worked so later pages on that host cost one request.
// The zero value is ready to use and safe for concurrent use.
type Resolver struct {
	mu       sync.Mutex
	patterns map[string]pattern
}

// Resolve normalizes input, probes candidates in ADR-0006 order, and
// returns the final URL (post-redirect) with the markdown body. When
// every candidate is HTML or missing it returns a *NotMarkdownError.
func (r *Resolver) Resolve(ctx context.Context, input string) (finalURL, markdown string, err error) {
	u, err := normalizeInput(input)
	if err != nil {
		return "", "", err
	}

	probes := candidatesFor(u)
	// A single candidate means we are not exploring: the URL already
	// names a markdown-looking resource, so give it the full timeout.
	committed := len(probes) == 1

	if pat, ok := r.lookup(u.Host); ok {
		if cached, ok := applyPattern(u, pat); ok {
			probes = promote(probes, cached)
			committed = true
		}
	}

	var probed []string
	for i, p := range probes {
		timeout := fetch.ProbeTimeout
		if committed && i == 0 {
			timeout = fetch.FetchTimeout
		}
		final, body, err := fetch.Markdown(ctx, p.url, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return "", "", ctx.Err()
			}
			probed = append(probed, p.url)
			continue
		}
		r.remember(hostOf(final, u.Host), p.pat)
		return final, body, nil
	}
	return "", "", &NotMarkdownError{URL: u.String(), candidates: probed}
}

// Forget drops the cached pattern for host, forcing the next Resolve
// touching it to re-probe (used by the reload key).
func (r *Resolver) Forget(host string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.patterns, host)
}

func (r *Resolver) lookup(host string) (pattern, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pat, ok := r.patterns[host]
	return pat, ok
}

func (r *Resolver) remember(host string, pat pattern) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.patterns == nil {
		r.patterns = make(map[string]pattern)
	}
	r.patterns[host] = pat
}

// normalizeInput adds a scheme to bare domains and validates the URL.
// Unlike fetch.Normalize it keeps an empty path as-is so Resolve can
// distinguish "bare domain" (probe llms.txt variants) from an explicit
// /llms.txt request.
func normalizeInput(input string) (*url.URL, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("empty URL")
	}
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", input, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host in %q", input)
	}
	return u, nil
}

// candidatesFor lists probe URLs for u in ADR-0006 order.
func candidatesFor(u *url.URL) []probe {
	switch {
	case u.Path == "" || u.Path == "/":
		return []probe{
			{withPath(u, "/llms.txt"), patternLLMS},
			{withPath(u, "/llms-full.txt"), patternLLMSFull},
		}
	case strings.HasSuffix(u.Path, ".md") || strings.HasSuffix(u.Path, ".txt"):
		return []probe{{u.String(), patternAsIs}}
	default:
		trimmed := strings.TrimSuffix(u.Path, "/")
		return []probe{
			{u.String(), patternNegotiate},
			{withPath(u, trimmed+".md"), patternAppendMD},
			{withPath(u, trimmed+"/index.md"), patternIndexMD},
		}
	}
}

// applyPattern builds the candidate URL a cached pattern implies for u,
// or reports false when the pattern does not fit u's shape (for example
// a cached llms.txt pattern applied to a deep URL).
func applyPattern(u *url.URL, pat pattern) (probe, bool) {
	bare := u.Path == "" || u.Path == "/"
	markdownish := strings.HasSuffix(u.Path, ".md") || strings.HasSuffix(u.Path, ".txt")
	trimmed := strings.TrimSuffix(u.Path, "/")

	switch pat {
	case patternLLMS:
		if !bare {
			return probe{}, false
		}
		return probe{withPath(u, "/llms.txt"), pat}, true
	case patternLLMSFull:
		if !bare {
			return probe{}, false
		}
		return probe{withPath(u, "/llms-full.txt"), pat}, true
	case patternAsIs:
		if !markdownish {
			return probe{}, false
		}
		return probe{u.String(), pat}, true
	case patternNegotiate:
		if bare || markdownish {
			return probe{}, false
		}
		return probe{u.String(), pat}, true
	case patternAppendMD:
		if bare || markdownish {
			return probe{}, false
		}
		return probe{withPath(u, trimmed+".md"), pat}, true
	case patternIndexMD:
		if bare || markdownish {
			return probe{}, false
		}
		return probe{withPath(u, trimmed+"/index.md"), pat}, true
	}
	return probe{}, false
}

// promote moves p to the front of probes, deduplicating by URL.
func promote(probes []probe, p probe) []probe {
	out := []probe{p}
	for _, q := range probes {
		if q.url != p.url {
			out = append(out, q)
		}
	}
	return out
}

func withPath(u *url.URL, path string) string {
	clone := *u
	clone.Path = path
	clone.RawQuery = ""
	clone.Fragment = ""
	return clone.String()
}

func hostOf(rawURL, fallback string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fallback
	}
	return u.Host
}
