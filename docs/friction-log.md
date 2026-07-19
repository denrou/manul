# Friction log

Raw notes from dogfooding the MVP. One entry per friction, however
small — a miss, a surprise, a wish. No polish; volume beats quality
here. v2 gets scoped from this file.

Entry format (copy the block):

```
## <date> — <site or action>
- expected:
- got:
- hurt (1=shrug, 3=annoying, 5=rage-quit):
- idea (optional):
```

---

## 2026-07-18 — example (delete me)

- expected: `manul docs.astro.build` to show their docs
- got: "No markdown here" — their llms.txt 404s despite being listed in a directory
- hurt: 2
- idea: start-page directory entries should be re-verified periodically

## 2026-07-18 — directory.llmstxt.cloud as landing page

- expected: the llms.txt aggregator directories to be browsable in manul
- got: directory.llmstxt.cloud publishes only a 4-line stub llms.txt — its
  actual site list is HTML-only; llms-txt.io and llmsdirectory.com are
  similar stubs. Only llmstxthub.com serves a real markdown directory
  (649 KB, auto-generated). Worked around with the new `home` config key.
- hurt: 3
- idea: strong evidence for the annuaire brick — the markdown web has
  almost no markdown-native directory; manul could generate/host its own

## 2026-07-18 — no way back after following a link (reported while dogfooding)

- expected: `H`/`L` to go back/forward, as in the Firefox vim plugins
  (vimium/tridactyl muscle memory)
- got: seemingly no history navigation. It existed (`Backspace`/`[`/`B`
  back, `]` forward) but under different keys AND invisible — the
  statusbar short help never mentioned it, so it read as missing.
- hurt: 4
- resolved: `H`/`L` added as back/forward aliases; back now shown in the
  statusbar short help
- idea: user-configurable keybindings in config.toml (remap navigation to
  taste) — real v2 item; also audit which other bindings are invisible
  outside the `?` page

## 2026-07-18 — no way to post-process a page (reported while dogfooding)

- expected: pipe the page through a shell command, e.g. count the links
  on the directory landing page with `grep llms.txt | wc -l`
- got: no escape hatch from the TUI to the shell in either direction
- hurt: 3
- resolved: `|` runs a shell command with the page source on stdin and
  opens the output as a page (back with `H`); `manul --dump <url>` prints
  resolved markdown to stdout for real pipelines. First live use:
  `manul --dump llmstxthub.com | grep -c llms.txt` → 2580
- idea: the pipe-result-as-page pattern composes (pipe a pipe result);
  worth exploring saved commands/aliases if this gets heavy use

## 2026-07-19 — no search in page (reported while dogfooding)

- expected: `/` to search within the current page, pager-style
- got: nothing — search-in-page was consciously cut from the MVP
  (ADR-0008) and dogfooding promoted it immediately
- hurt: 4
- resolved: `/` searches the rendered page (smart case, ASCII fold,
  matches highlighted, ANSI-aware so styled text matches), `n`/`N`
  cycle with wrap, Esc clears; match counter lives in the statusbar;
  query survives resizes, clears on navigation
- idea: `/` on the huge directory page + `|` pipes make manul a decent
  llms.txt exploration tool already; regex search only if asked for

## 2026-07-19 — following by number is slower than vimium's letters

- expected: vimium's `f` hint flow — letters are faster to type than
  numbers (better finger access, 26 symbols vs 10 before going
  multi-char)
- got: numbers only; fine for precision, slow for rapid hopping
- hurt: 3
- resolved: `f` switches visible links to letter hints (home-row-first
  alphabet, single letters up to 26 on-screen links, fixed-width combos
  beyond); typing a label follows instantly; Esc returns to numbers;
  labels are padded to the numeric marker's width so the layout doesn't
  shift. Trade-off accepted: `f` no longer pages down (Space/PgDn/d
  remain).
- idea: numbers and hints now coexist as two speeds (precise vs fast);
  if configurable keybindings land, the hint alphabet should be
  configurable too (e.g. azerty home row)
