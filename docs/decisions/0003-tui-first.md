# ADR-0003 — TUI first, the terminal is the platform

**Status**: Accepted
**Date**: 2026-07-17

## Context

The viewer could be a desktop app (like Lagrange for Gemini), a browser
extension, or a terminal UI. The early markdown web skews heavily
toward developer documentation, so the early audience is developers who
live in terminals and adopt tools via `brew install`.

## Decision

Build manul as a TUI. The terminal is not a stepping stone; it is the
product's home. GUI or web front-ends are possible later but are not a
goal of the MVP.

## Consequences

- Distribution is a single binary + brew tap; no app stores, no code
  signing.
- Keyboard-only interaction; the keymap must follow the idioms the
  audience already knows (less/vim/w3m).
- Rendering constraints (no inline images in MVP) are accepted as
  features of the medium, not gaps.
