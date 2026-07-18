# ADR-0002 — Name the project "manul"

**Status**: Accepted
**Date**: 2026-07-17

## Context

Terminal browsers have a feline naming lineage: Lynx, the canonical
text-mode browser, is a wild cat. The name had to be short, fast to
type as a CLI command, memorable, and available. Candidates considered:
wisp, tern, gander (all taken by active dev tools), lire (French "to
read", pronunciation-ambiguous in English), margay, leaf, moth, skiff.

## Decision

Name the project **manul** — the Pallas's cat: a small, famously
grumpy, meme-beloved wild cat.

- Continues the Lynx wild-cat lineage, signalling "text browser
  successor" to the exact target audience.
- The grumpy-cat mascot fits the attitude: no JS, no CSS, no tracking.
- Starts with `man`, the most-typed reading command in Unix history.
- Five letters, alternating hands, hard to typo.

Availability checked 2026-07-17: Homebrew formula and cask free;
GitHub `denrou/manul` free (the bare `manul` user name is a dormant
account); existing same-name projects (a fuzzer, a dead pre-modules Go
vendoring tool) do not conflict in namespace or spirit. `manul.dev` and
`manul.app` are registered by third parties; `manul.sh` appeared free.

## Consequences

- Binary and module are `manul`; the brew formula name is ours to take.
- A future website cannot use `manul.dev`/`manul.app` without buying it.
