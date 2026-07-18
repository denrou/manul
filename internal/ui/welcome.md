# manul

> A grumpy little browser for the markdown web.

No HTML. No CSS. No JavaScript. Just markdown, rendered *your* way —
the same on every site.

A growing part of the web publishes agent-readable markdown alongside
HTML: an `/llms.txt` index at the root and `.md` twins of each page.
That content is fast, light, and free of trackers and ads. It was made
for machines, but it is the web many humans wished they had. manul is
a terminal reader for it.

## Try it

```sh
manul docs.anthropic.com
```

manul normalizes what you type: a bare domain fetches its `/llms.txt`,
a full URL is fetched as-is.

## Keys

- `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn` — scroll
- `q` / `Esc` — quit

## Status

This is a walking skeleton (v0.0): fetch one document, render it,
scroll. Link navigation, history, discovery heuristics, themes, and
bookmarks are on the roadmap — see the README.
