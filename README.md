# manul

> A grumpy little browser for the markdown web.

No HTML rendering. No CSS. No JavaScript. Just markdown, rendered your
way — identically on every site.

## Why

A growing slice of the web publishes agent-readable markdown alongside
HTML: an [`/llms.txt`](https://llmstxt.org/) index file at the root and
`.md` twins of individual pages. This content is small, fast to fetch,
and free of trackers and ads — built for machines, but it is the web
many humans wished they had. manul is a terminal reader for that web,
in the lineage of [Lynx](https://lynx.invisible-island.net/): type a
domain, read its markdown, follow links.

The name: the manul (Pallas's cat) is a small, famously grumpy wild
cat — a fitting successor to the lynx, and just as unimpressed by
JavaScript.

## Usage

```sh
manul                     # welcome page
manul docs.anthropic.com  # bare domain → fetches /llms.txt
manul https://example.com/docs/page.md
```

Keys: `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn` to scroll · `q` or `Esc` to quit.

## Build

Requires Go 1.26+.

```sh
make build   # produces ./manul
make test
```

## Roadmap

- [x] v0.0 — walking skeleton: fetch one document, render, scroll
- [ ] v0.2 — the core loop: llms.txt discovery probing, selectable
      links, follow/back/forward history, URL prompt
- [ ] v0.3 — shareable: themes via config file, bookmarks, seeded
      start page, graceful fallback for HTML-only sites, brew tap

Out of scope for the MVP: crawling/indexing, HTML→markdown conversion,
tabs, images, scripting of any kind (that last one is permanent).

## License

MIT — see [LICENSE](LICENSE).
