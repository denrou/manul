# MVP spec — v1 (final, after two critique rounds)

Implementation spec for the manul MVP ("core loop" milestone). Grounded
in ADR-0001..0010; this document is the build contract. Sections marked
**[risk]** were flagged by review as estimate hotspots.

## Package contracts

### internal/fetch (extend existing)

- `Markdown` returns the **final URL after redirects**
  (`resp.Request.URL.String()`) alongside the body:
  `func Markdown(ctx, rawURL string) (finalURL, body string, err error)`.
  Callers must use the final URL for link resolution and cache keys
  (verified live: docs.anthropic.com/llms.txt redirects cross-host).
- HTML detection lives here, checked early: `Content-Type: text/html`
  or body prefix sniff (`<!doctype`, `<html`, case-insensitive,
  whitespace-tolerant) returns typed `ErrHTML` (sentinel or type —
  must be `errors.Is`/`errors.As`-matchable).
- Timeout becomes a parameter: probes use ~8s, the committed final
  fetch keeps 30s. Keep 10 MiB cap, Accept negotiation, User-Agent.

### internal/doc (new) — **[risk: the hardest pure-code task]**

```go
type Link struct {
    Index      int    // 1-based, document order
    Text       string
    Dest       string
    Start, End int    // byte offsets of the full link source span
}
type Document struct {
    URL      string // final URL (post-redirect)
    Markdown string // possibly truncated — see Truncate
    Links    []Link
}
func Parse(url, markdown string) Document
func (d Document) ResolveLink(l Link) (string, error)
func Annotate(markdown string, links []Link, selected int) string
func Truncate(markdown string, max int) (out string, truncated bool)
```

- Parse uses goldmark **with extension.GFM** (Linkify included) so the
  parse dialect matches glamour's render dialect — bare URLs become
  links in both or neither.
- Offsets: inline links (`[t](d)`) — scan forward from the text
  segment past the destination (nested parens, optional title);
  autolinks (`<url>` and GFM linkified bare URLs) — the AST segment
  gives the span directly. **Reference-style links (`[t][ref]`) are
  extracted as Links but skipped by Annotate in the MVP** (no marker);
  documented and tested.
- Annotate splices markers **by byte offsets, in reverse order**.
  Marker is always `**[n]**` (bold — constant rendered cell width
  regardless of selection). Selection is NOT conveyed in the source;
  see the render pipeline below. Table tests: duplicate links,
  reference links, autolinks, links in headings/list items/tables,
  document containing literal `[3]` text, unicode before offsets.
- ResolveLink: resolve relative dest against d.URL; reject non-http(s).
- Truncate (for the 1 MiB interactive render cap): cut at the last
  newline before max, then if the cut text contains an odd number of
  ``` fences, append a closing fence, then append a truncation notice
  line. **Truncation happens BEFORE Parse** so Markdown, Links, and
  render all describe the same text.

### internal/discover (new)

```go
type NotMarkdownError struct{ URL string } // the ORIGINAL normalized input URL, not the last probe
type Resolver struct{ /* per-host pattern cache, mutex-guarded */ }
func (r *Resolver) Resolve(ctx context.Context, input string) (finalURL, markdown string, err error)
func (r *Resolver) Forget(host string) // for 'r' reload bypass
```

- Probe order: bare domain/empty path → `/llms.txt`, `/llms-full.txt`;
  URL ending `.md`/`.txt` → as-is; other deep URLs → Accept
  negotiation on the URL itself, then `<url>.md`, `<url>/index.md`.
- Per-probe timeout ~8s (via fetch). `ErrHTML`/404/timeout → next
  candidate. All candidates HTML or missing → `*NotMarkdownError`
  carrying the original input URL (probed candidates listed in the
  error string for the fallback page body).
- Pattern cache keyed by **final-URL host** (post-redirect).
- httptest suite: llms.txt hit; llms-full fallback; .md probe;
  content negotiation; HTML-only → NotMarkdownError; cross-host
  redirect (final URL propagated); cache used on second request;
  Forget forces re-probe.

### internal/config (new)

- `Config{Theme string; MaxWidth int; StylePath string; Mouse bool}`;
  `Load()` from `${XDG_CONFIG_HOME:-~/.config}/manul/config.toml`
  (BurntSushi/toml); absent → defaults `{Theme: "auto", MaxWidth: 100,
  Mouse: false}`. Theme: auto|dark|light|notty, or StylePath to a
  glamour style JSON. Mouse default **false**: cell-motion capture
  breaks terminal-native text selection in a reader app.
- Tests: absent file, partial file, invalid TOML.

### internal/bookmarks (new)

- Store at `<configdir>/bookmarks.md` — a real markdown page
  (`# Bookmarks` + `- [title](url)`) that manul itself renders.
- `Add(title, url)` dedupes by URL; `Markdown()` returns page or "".
- Tests: add, dedupe, render, missing file.

## UI (internal/ui rework + cmd/manul)

### Render pipeline

- Glamour style resolved ONCE in main() before `tea.NewProgram`
  (config theme; auto → `lipgloss.HasDarkBackground`), passed into ui.
- ONE cached renderer per clamped wrap width
  (`min(termWidth, cfg.MaxWidth)`); rebuilt only when the clamp
  changes; resize re-render skipped when unchanged, debounced ~100ms.
- Per-document render: Truncate → Parse → Annotate (all markers,
  no selection) → glamour render ONCE → cache the rendered string.
- **Selection highlight is post-render**: find the nth bold marker
  token in the cached rendered output and restyle it (reverse video),
  no glamour pass per Tab. Fallback if the ANSI signature proves
  unreliable during implementation: re-render via cached renderer per
  cycle (acceptable for llms.txt-scale docs; note it in code).
  Preserve viewport YOffset across SetContent.
- Chrome height centralized in one function of the input mode;
  viewport height = max(0, termHeight - chrome).

### Navigation & async

- `navigate(url)` → `tea.Cmd` running discover.Resolve →
  `docLoadedMsg` / `loadFailedMsg`. **Every message carries a
  monotonic `navSeq int`** incremented per navigate; Update drops
  messages whose seq != current (stale-fetch race guard — unit-test).
- Model holds per-navigation `context.CancelFunc`; new navigation,
  Esc, or quit cancels in-flight fetch. Spinner + `loading <host>…
  (esc cancels)` in statusbar. Initial CLI arg goes through this path
  (TUI appears instantly).
- Fetch failure keeps current page + history; error shown in
  statusbar until next keypress or successful navigation.
  NotMarkdownError → rendered in-app fallback page (enters history)
  showing the original URL + probed candidates, `o` opens it in the
  system browser (`open`/`xdg-open`).
- History: back/forward stacks of `{url, markdown, scrollFraction}`;
  fraction = clamp(ScrollPercent(), 0, 1), 0 when content fits
  (guard the NaN case); restore via
  `SetYOffset(round(frac * max(0, lines-height)))`. Stacks capped at
  50 entries. Unit-test the math.

### Link selection

- Tab selects the **first link at or below the current viewport top**;
  Shift-Tab the last above the bottom; subsequent presses cycle.
  After each change, scroll minimally to keep the selected marker
  visible (locate its rendered line — pure func, unit-tested).
- **Digit-buffer follow** (lynx model): typing digits shows
  `follow: 42_` in the statusbar; Enter follows link #42; Esc clears.
- Enter with no selection and empty digit buffer falls through to the
  viewport default (scroll).
- Followed links go through discover.Resolve (an HTML page URL with a
  .md twin still works).

### Keymap (bubbles/key.Binding + bubbles/help)

- Viewport defaults (j/k/arrows/space/b/d/u/pgup/pgdn) + g/G +
  Home/End top/bottom.
- Tab/Shift-Tab link cycle; digits+Enter follow-by-number.
- Backspace and `[` back; `]` forward.
- `:` URL prompt (bubbles/textinput above statusbar).
- `o` open current page URL externally; `y` yank current page URL (or
  selected link URL if selection active) via OSC 52 (termenv).
- `a` add bookmark (lynx binding — NOT `B`, which collides with w3m
  back; leave B as a back alias). Title = first H1 or URL.
- `s` start page; `r` reload (discover.Forget + refetch); `?` help
  page (embedded markdown); q/ctrl+c quit.
- **Esc NEVER quits.** Precedence, one layer per press: close prompt >
  cancel in-flight fetch > clear digit buffer > clear link selection.
- **Prompt mode routing**: while the textinput is focused, KeyMsgs go
  exclusively to it except Enter/Esc/ctrl+c (unit test: `q` typed in
  prompt must not quit). Chrome height accounts for the prompt row.
- Mouse: `tea.WithMouseCellMotion` only when `cfg.Mouse` is true;
  help page notes the text-selection tradeoff and shift-drag escape.
- Short help in statusbar, full help on `?`.

### Start page

- Rewritten welcome that teaches in-app keys (`:` to open a URL, Tab
  to cycle links, Enter to follow) — its own links are the tutorial.
- `## Directory` section: seed list of live-verified llms.txt sites.
- `## Bookmarks` section appended when non-empty.

### cmd/manul

- `--version` flag (const, goreleaser-ready); optional arg → initial
  navigate; else start page.

## CI

`.github/workflows/ci.yml`: build, vet, test on ubuntu-latest +
macos-latest, Go 1.26.

## Out of scope (ADR-0007/0008)

Crawler/index, HTML→markdown conversion, tabs, images, search-in-page,
disk cache, mermaid, in-app theme editor, streaming per-probe progress
messages (deferred — the 8s probe timeout bounds worst-case wait).

## Dependency policy

charmbracelet suite + BurntSushi/toml + goldmark, all pre-added as
direct deps by the orchestrator. Implementation agents must NOT touch
go.mod/go.sum.

## Demo acceptance

`manul docs.anthropic.com` → llms.txt renders themed, spinner-backed,
across the redirect; Tab → Enter follows a .md link; Backspace returns
with scroll restored; `:` example.com → clean fallback offering `o`;
`s` start page shows directory + bookmarks; config `theme = "dark"`
respected; Esc during a slow load cancels leaving the page intact;
typing `q` in the URL prompt does not quit; all tests green; CI file
present.
