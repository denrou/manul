# ADR-0007 — Render published markdown only, never convert HTML

**Status**: Accepted
**Date**: 2026-07-17

## Context

When a site has no markdown twin, manul could scrape the HTML and
convert it to markdown (as Jina Reader, Markdowner, and reader modes
do). That would raise coverage dramatically.

## Decision

manul renders **only markdown that publishers intentionally serve**.
No HTML→markdown conversion, no reader-mode extraction, in the MVP or
later. HTML-only targets get a graceful fallback page with a one-key
hand-off to the system browser.

## Consequences

- Coverage is limited to sites that opted into the markdown web; early
  on, manul is mostly a documentation browser. Accepted.
- The product story stays clean: what you read is what the site chose
  to publish, rendered faithfully — manul is a client, not a scraper.
- Publishers keep the incentive to publish real markdown; a converter
  would remove it.
- JavaScript execution is permanently out of scope; this is identity,
  not a phase.
