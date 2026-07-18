# ADR-0008 — MVP scope cut

**Status**: Accepted
**Date**: 2026-07-17

## Context

The core product loop to validate: type a domain → auto-discover its
markdown → read it themed → follow links → back/forward. Everything
that does not serve that loop dilutes the MVP. Definition of done:
dogfood a real docs site end-to-end (browse, follow links, return),
hit a markdown-less site, and get a clean fallback — without touching a
regular browser or seeing raw markdown syntax.

## Decision

**In scope**: discovery engine (ADR-0006), themed rendering
(ADR-0009), keyboard link navigation, back/forward history, URL
prompt, graceful HTML fallback, bookmarks (stored as a markdown page
manul itself renders), seeded start page, config file, CI.

**Out of scope for the MVP**: crawler/index ("annuaire" — the seeded
start page stands in for it), HTML→markdown conversion (permanently,
ADR-0007), tabs, images (sixel/kitty later), search-in-page, disk
cache, mermaid diagrams.

## Consequences

- The index/directory ambition is deferred; existing public llms.txt
  directories seed the start page instead.
- Feature requests belonging to the out list are answered with "after
  the MVP" or "never" (per ADR-0007), by pointing at this record.
