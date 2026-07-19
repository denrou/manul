# manul — project instructions for Claude

manul is a TUI browser for the markdown web (sites publishing
`/llms.txt` and `.md` page twins). Go + bubbletea + glamour.

## Roles and workflow

- **Claude develops manul end-to-end.** The repo owner plays the user:
  they build and run from `main` and report frictions, which land in
  `docs/friction-log.md`. v2+ scoping is driven by that file.
- **Standing authorization (owner, 2026-07-18): Claude pushes and
  merges itself.** Either push small changes (docs, fixes, minor
  features) directly to `main`, or open a PR and self-merge for larger
  work — whichever is cleanest. Do not wait for the owner to merge.
- The owner builds from `main`, so never leave finished work stranded
  on a feature branch — land it. If a PR is open, check its merge
  state before pushing more commits to its branch (twice already,
  commits pushed after a fast manual merge ended up stranded).
- Keep `main` green: `go build ./... && go vet ./... && go test ./...`
  must pass before anything lands on `main`.

## Conventions

- Decisions are immutable ADRs in `docs/decisions/` — supersede with a
  new numbered record, never edit an accepted one (see its README).
- No HTML rendering, ever (ADR-0007). No JS, no tracking — identity,
  not a phase.
- Dependency policy: charmbracelet suite + BurntSushi/toml + goldmark
  (pinned to glamour's version); add anything else reluctantly.
- Keybindings follow pager/vim/w3m idioms; every binding must be
  discoverable (statusbar short help or the `?` page).

## Commands

- `make build` / `make test` / `make vet`
- Quick smoke: `./manul --dump llmstxthub.com | head`
- TUI smoke under a pty: size it first
  (`script` ptys default to 0×0 in headless sessions).
