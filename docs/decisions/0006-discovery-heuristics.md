# ADR-0006 — Discovery: probe heuristics over a single convention

**Status**: Accepted
**Date**: 2026-07-17

## Context

There is no standard mapping from a page URL to its markdown twin.
In the wild: `/llms.txt` and `/llms-full.txt` at the root, `page.md`,
`page/index.md`, `.html.md`, and `Accept: text/markdown` content
negotiation all coexist. A browser that supports only one convention
would fail on most real sites.

## Decision

manul treats discovery as a first-class engine that probes candidates
in a fixed order:

1. Bare domain (empty path): `/llms.txt`, then `/llms-full.txt`.
2. URL already pointing at `.md`/`.txt`: fetch as-is.
3. Other deep URLs: content negotiation with `Accept: text/markdown`,
   then probe `<url>.md`, then `<url>/index.md`.

Responses are sniffed; HTML is never rendered (ADR-0007). The pattern
that worked is cached per host for the session so subsequent pages cost
one request. When every candidate fails, the UI shows a clear
"no markdown here" page offering to open the URL in a system browser —
never a silent dead-end.

## Consequences

- Hiding this mess is a core part of manul's value; the probe order is
  product behavior and changes to it deserve new ADRs.
- Probing costs extra requests on first contact with a host; the
  per-host cache bounds that cost.
- Sites with exotic mappings still fail; the fallback page is the
  contract.
