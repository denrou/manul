# Keys

## Scrolling

- `j` / `k`, `↓` / `↑` — line down / up
- `Space`, `f`, `PgDn` — page down; `b`, `PgUp` — page up
- `d` / `u` — half page down / up
- `g`, `Home` — top; `G`, `End` — bottom

## Links

- `Tab` / `Shift+Tab` — select the next / previous link
- `Enter` — follow the selected link
- digits, then `Enter` — follow a link by number (e.g. `4` `2` `Enter`)

## Navigation

- `:` — open a URL or bare domain (manul discovers the markdown twin)
- `H`, `Backspace`, `[`, `B` — back; `L`, `]` — forward
  (`H`/`L` follow the vimium/tridactyl convention)
- `r` — reload, bypassing the per-host discovery cache
- `s` — start page (directory and bookmarks)

## Page actions

- `o` — open the current page's URL in your system browser
- `y` — yank the page URL (or the selected link's URL) to the
  clipboard via OSC 52
- `a` — bookmark the current page (title: first heading, else URL);
  bookmarks appear on the start page

## Link hints (vimium style)

- `f` — switch the visible links from numbers to letter labels; type a
  label to follow its link instantly. Single letters cover up to 26
  on-screen links, then fixed-length combinations kick in.
- `Esc` — back to numbers without following anything.
- Note: `f` no longer pages down (Space, PgDn, and `d` still do).

## Search in page

- `/` — search the rendered page (smart case: all-lowercase queries
  match case-insensitively, any uppercase makes the search exact;
  ASCII folding). All matches highlight; the view jumps to the first
  match at or below your position.
- `n` / `N` — next / previous match (wraps around)
- `Esc` — clear the search

## Pipe to a command

- `|` — run a shell command with the current page's raw markdown on
  stdin; the output opens as a page (go back with `H`). Example on a
  directory page: `| grep -c llms.txt` counts the listed sites.
- Outside the TUI, `manul --dump <url>` prints the resolved markdown
  to stdout for real shell pipelines:
  `manul --dump llmstxthub.com | grep -c llms.txt`

## Everything else

- `Esc` — one layer per press: close the URL prompt, cancel an
  in-flight load, clear the typed link number, clear the link
  selection. Esc never quits.
- `?` — this page
- `q`, `Ctrl+C` — quit

## Mouse

Wheel scrolling works when `mouse = true` is set in
`~/.config/manul/config.toml`. Tradeoff: capturing the mouse breaks
your terminal's native text selection — in most terminals, hold
`Shift` while dragging to select text anyway. With `mouse = false`
(the default), selection works normally and you scroll with the keys.

## Home page

By default a no-argument launch shows this built-in start page. Set
`home = "<url or domain>"` in `~/.config/manul/config.toml` to land
somewhere else — for example `home = "llmstxthub.com"`, a large
auto-generated markdown directory of sites implementing llms.txt.
The built-in start page stays one `s` away.
