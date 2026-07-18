package discover

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const htmlPage = "<!doctype html><html><body>not markdown</body></html>"

// countingServer records how many times each path was requested.
type countingServer struct {
	*httptest.Server

	mu     sync.Mutex
	counts map[string]int
}

func newCountingServer(t *testing.T, handler http.HandlerFunc) *countingServer {
	t.Helper()
	cs := &countingServer{counts: make(map[string]int)}
	cs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cs.mu.Lock()
		cs.counts[r.URL.Path]++
		cs.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(cs.Close)
	return cs
}

func (cs *countingServer) count(path string) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.counts[path]
}

func (cs *countingServer) total() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	sum := 0
	for _, n := range cs.counts {
		sum += n
	}
	return sum
}

func serveMarkdown(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/markdown")
	_, _ = w.Write([]byte(body))
}

func serveHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlPage))
}

func TestResolveLLMSTxtHit(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/llms.txt" {
			serveMarkdown(w, "# Site index")
			return
		}
		http.NotFound(w, r)
	})

	var r Resolver
	finalURL, markdown, err := r.Resolve(context.Background(), cs.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if markdown != "# Site index" {
		t.Errorf("markdown = %q, want %q", markdown, "# Site index")
	}
	if want := cs.URL + "/llms.txt"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
	if n := cs.total(); n != 1 {
		t.Errorf("request count = %d, want 1", n)
	}
}

func TestResolveLLMSFullFallback(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/llms-full.txt" {
			serveMarkdown(w, "# Full index")
			return
		}
		http.NotFound(w, r)
	})

	var r Resolver
	finalURL, markdown, err := r.Resolve(context.Background(), cs.URL+"/")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if markdown != "# Full index" {
		t.Errorf("markdown = %q, want %q", markdown, "# Full index")
	}
	if want := cs.URL + "/llms-full.txt"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
	if n := cs.count("/llms.txt"); n != 1 {
		t.Errorf("/llms.txt probed %d times, want 1", n)
	}
}

func TestResolveMarkdownURLAsIs(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/docs/guide.md" {
			serveMarkdown(w, "# Guide")
			return
		}
		http.NotFound(w, r)
	})

	var r Resolver
	finalURL, markdown, err := r.Resolve(context.Background(), cs.URL+"/docs/guide.md")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if markdown != "# Guide" {
		t.Errorf("markdown = %q, want %q", markdown, "# Guide")
	}
	if want := cs.URL + "/docs/guide.md"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
	if n := cs.total(); n != 1 {
		t.Errorf("request count = %d, want 1 (as-is, no extra probes)", n)
	}
}

func TestResolveContentNegotiation(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/docs/guide" && strings.Contains(r.Header.Get("Accept"), "text/markdown") {
			serveMarkdown(w, "# Negotiated")
			return
		}
		http.NotFound(w, r)
	})

	var r Resolver
	finalURL, markdown, err := r.Resolve(context.Background(), cs.URL+"/docs/guide")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if markdown != "# Negotiated" {
		t.Errorf("markdown = %q, want %q", markdown, "# Negotiated")
	}
	if want := cs.URL + "/docs/guide"; finalURL != want {
		t.Errorf("finalURL = %q, want %q", finalURL, want)
	}
	if n := cs.total(); n != 1 {
		t.Errorf("request count = %d, want 1", n)
	}
}

func TestResolveHTMLOnlyReturnsNotMarkdownError(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		serveHTML(w)
	})

	input := cs.URL + "/docs/guide"
	var r Resolver
	_, _, err := r.Resolve(context.Background(), input)

	var nme *NotMarkdownError
	if !errors.As(err, &nme) {
		t.Fatalf("err = %v, want *NotMarkdownError", err)
	}
	if nme.URL != input {
		t.Errorf("NotMarkdownError.URL = %q, want original input %q", nme.URL, input)
	}
	for _, candidate := range []string{
		cs.URL + "/docs/guide",
		cs.URL + "/docs/guide.md",
		cs.URL + "/docs/guide/index.md",
	} {
		if !strings.Contains(nme.Error(), candidate) {
			t.Errorf("Error() = %q, missing probed candidate %q", nme.Error(), candidate)
		}
	}
	if n := cs.total(); n != 3 {
		t.Errorf("request count = %d, want 3", n)
	}
}

func TestResolveCrossHostRedirect(t *testing.T) {
	target := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/llms.txt" {
			serveMarkdown(w, "# Moved here")
			return
		}
		http.NotFound(w, r)
	})
	origin := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusFound)
	})

	var r Resolver
	finalURL, markdown, err := r.Resolve(context.Background(), origin.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if want := target.URL + "/llms.txt"; finalURL != want {
		t.Errorf("finalURL = %q, want post-redirect %q", finalURL, want)
	}
	if markdown != "# Moved here" {
		t.Errorf("markdown = %q, want %q", markdown, "# Moved here")
	}

	targetHost := mustHost(t, target.URL)
	originHost := mustHost(t, origin.URL)
	r.mu.Lock()
	_, cachedTarget := r.patterns[targetHost]
	_, cachedOrigin := r.patterns[originHost]
	r.mu.Unlock()
	if !cachedTarget {
		t.Errorf("pattern not cached under final host %q", targetHost)
	}
	if cachedOrigin {
		t.Errorf("pattern cached under pre-redirect host %q", originHost)
	}
}

func TestResolveUsesCachedPatternAndForget(t *testing.T) {
	cs := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".md") {
			serveMarkdown(w, "# Page "+r.URL.Path)
			return
		}
		serveHTML(w)
	})

	var r Resolver

	// First contact: negotiation on /page fails (HTML), /page.md works.
	if _, _, err := r.Resolve(context.Background(), cs.URL+"/page"); err != nil {
		t.Fatalf("first Resolve returned error: %v", err)
	}
	if n := cs.total(); n != 2 {
		t.Fatalf("first resolve made %d requests, want 2", n)
	}

	// Second page on the same host: cached append-.md pattern goes
	// straight to /other.md, one request, no negotiation retry.
	finalURL, _, err := r.Resolve(context.Background(), cs.URL+"/other")
	if err != nil {
		t.Fatalf("second Resolve returned error: %v", err)
	}
	if want := cs.URL + "/other.md"; finalURL != want {
		t.Errorf("finalURL = %q, want cached-pattern %q", finalURL, want)
	}
	if n := cs.total(); n != 3 {
		t.Errorf("second resolve total = %d requests, want 3 (exactly one more)", n)
	}
	if n := cs.count("/other"); n != 0 {
		t.Errorf("/other negotiated %d times, want 0 (cache should skip it)", n)
	}

	// Forget drops the cache: the next resolve re-probes from scratch.
	r.Forget(mustHost(t, cs.URL))
	if _, _, err := r.Resolve(context.Background(), cs.URL+"/third"); err != nil {
		t.Fatalf("post-Forget Resolve returned error: %v", err)
	}
	if n := cs.count("/third"); n != 1 {
		t.Errorf("/third negotiated %d times after Forget, want 1 (full re-probe)", n)
	}
	if n := cs.total(); n != 5 {
		t.Errorf("post-Forget total = %d requests, want 5", n)
	}
}

func TestResolveInvalidInput(t *testing.T) {
	var r Resolver
	for _, input := range []string{"", "ftp://example.com/x.md", "https://"} {
		if _, _, err := r.Resolve(context.Background(), input); err == nil {
			t.Errorf("Resolve(%q) succeeded, want error", input)
		}
	}
}

func mustHost(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing %q: %v", rawURL, err)
	}
	return u.Host
}
