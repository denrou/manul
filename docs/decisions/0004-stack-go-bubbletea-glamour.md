# ADR-0004 — Stack: Go + bubbletea + glamour

**Status**: Accepted
**Date**: 2026-07-17

## Context

Three stacks were evaluated for a TUI markdown browser:

- **Go + bubbletea + glamour** (charmbracelet): glamour is a terminal
  markdown renderer with a JSON-based theming system, proven by glow;
  Go compiles to a single static binary.
- **Python + Textual**: fastest prototyping (built-in Markdown widget
  with clickable links) but heavier distribution (pipx/uv) and would
  require rebuilding the theming glamour ships with.
- **Rust + ratatui**: most effort on the least differentiated layer
  (markdown rendering/theming is immature there).

## Decision

Go with **Go 1.26+, bubbletea (TUI runtime), bubbles (widgets),
glamour (markdown rendering + themes), lipgloss (styling)**. Additional
dependencies are added reluctantly; goldmark (already glamour's parser)
is used directly for AST work, BurntSushi/toml for config.

## Consequences

- Theming (the product differentiator, ADR-0009) comes nearly free via
  glamour styles.
- Single-binary distribution enables the brew-first strategy (ADR-0003).
- We accept glamour's rendering limits (e.g., link presentation) and
  work within them rather than writing a custom renderer.
