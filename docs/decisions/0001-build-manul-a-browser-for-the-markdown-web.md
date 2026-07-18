# ADR-0001 — Build manul, a browser for the markdown web

**Status**: Accepted
**Date**: 2026-07-17

## Context

A growing slice of the web publishes agent-readable markdown alongside
HTML: an `/llms.txt` index at the root (llmstxt.org, proposed by
Answer.AI in September 2024) and `.md` twins of individual pages. As of
mid-2026, roughly 5–10% of top sites publish it, growth is ~5x per
year, and platforms (Shopify, Mintlify, Fern) generate it
automatically. This content is small, fast, and free of trackers and
ads — built for machines, yet it is the web many humans wished they
had. No human-facing client exists for it: browsers show these files as
raw unrendered text. The closest prior art, the Gemini protocol,
required publishers to adopt a separate protocol and server, which
capped it at hobbyist scale.

## Decision

Build manul: a reader/browser dedicated to the markdown web. It fetches
only markdown over plain HTTPS, renders it uniformly with the reader's
own theme, and never executes anything. The content supply grows on its
own (publishers add markdown for AI visibility); manul only has to
build the demand side.

## Consequences

- Early coverage skews to developer documentation; the empty-state and
  fallback experience is a first-class concern, not an edge case.
- The project bets on the llms.txt ecosystem surviving; if publishers
  abandon it, manul loses its content supply.
- No HTML, CSS, or JavaScript support — ever. See ADR-0007.
