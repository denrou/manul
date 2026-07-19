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
- `Backspace`, `[`, `B` — back; `]` — forward
- `r` — reload, bypassing the per-host discovery cache
- `s` — start page (directory and bookmarks)

## Page actions

- `o` — open the current page's URL in your system browser
- `y` — yank the page URL (or the selected link's URL) to the
  clipboard via OSC 52
- `a` — bookmark the current page (title: first heading, else URL);
  bookmarks appear on the start page

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
