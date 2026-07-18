# ADR-0010 — llms.txt treated as plain markdown

**Status**: Accepted
**Date**: 2026-07-17

## Context

The llms.txt spec defines a semi-formal structure (H1 title, blockquote
summary, H2 link sections). manul could parse that structure into a
navigable site menu, or render the file as ordinary markdown. Real
files in the wild are frequently sloppy or only loosely compliant.

## Decision

The MVP renders `llms.txt` as **plain markdown**, exactly like any
other page. No structural parsing, no special llms.txt view. Since the
format is markdown links in sections, ordinary link navigation already
makes it a usable site menu.

## Consequences

- Compliant and sloppy files get the same, predictable treatment.
- A structured "site map" view remains open as a post-MVP enhancement
  and would be introduced by a superseding ADR.
