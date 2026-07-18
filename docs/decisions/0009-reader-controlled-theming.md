# ADR-0009 — Reader-controlled theming

**Status**: Accepted
**Date**: 2026-07-17

## Context

On the HTML web, the publisher controls presentation. manul inverts
this: every site renders through the *reader's* chosen theme,
identically. This inversion is the product differentiator, so it must
exist in the MVP — but an in-app theme editor would blow the scope.

## Decision

Theming is configured through a file
(`${XDG_CONFIG_HOME:-~/.config}/manul/config.toml`): a named built-in
glamour style (`auto`, `dark`, `light`, `notty`) or a path to a custom
glamour style JSON, plus reading preferences such as maximum content
width. No in-app editor in the MVP; glamour's style JSON format is the
customization surface.

## Consequences

- Uniform reading experience across all sites — the point of manul.
- Users who want deep customization edit a documented JSON style file;
  power users are the audience, this is acceptable.
- Adopting glamour's style schema couples theming to glamour (accepted,
  consistent with ADR-0004).
