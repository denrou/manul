# Decision records

This directory is manul's decision log. Every significant product or
technical choice is recorded here as an Architecture Decision Record
(ADR), numbered in the order it was made.

## Convention: records are immutable

A decision is a fact about the past. Once an ADR is merged with status
**Accepted**, its content is never edited (typo fixes excepted). If a
decision changes, write a *new* ADR that supersedes the old one and
update the old record's status line to `Superseded by ADR-NNNN` — that
single status line is the only permitted amendment.

This convention exists on purpose: choices are fixed in time, and the
log must stay trustworthy as a history of *why* the project looks the
way it does. A GitHub wiki was considered for this log and rejected
(see [ADR-0005](0005-decision-records-versioned-in-repo.md)).

## Format

Each record: **Status** / **Date** / **Context** / **Decision** /
**Consequences**. Keep them short; link related records.

## Index

- [ADR-0001](0001-build-manul-a-browser-for-the-markdown-web.md) — Build manul, a browser for the markdown web
- [ADR-0002](0002-name-the-project-manul.md) — Name the project "manul"
- [ADR-0003](0003-tui-first.md) — TUI first, the terminal is the platform
- [ADR-0004](0004-stack-go-bubbletea-glamour.md) — Stack: Go + bubbletea + glamour
- [ADR-0005](0005-decision-records-versioned-in-repo.md) — Decision records versioned in-repo, not a wiki
- [ADR-0006](0006-discovery-heuristics.md) — Discovery: probe heuristics over a single convention
- [ADR-0007](0007-render-published-markdown-only.md) — Render published markdown only, never convert HTML
- [ADR-0008](0008-mvp-scope-cut.md) — MVP scope cut
- [ADR-0009](0009-reader-controlled-theming.md) — Reader-controlled theming
- [ADR-0010](0010-llms-txt-treated-as-plain-markdown.md) — llms.txt treated as plain markdown
