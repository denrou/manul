# ADR-0005 — Decision records versioned in-repo, not a wiki

**Status**: Accepted
**Date**: 2026-07-17

## Context

Project decisions must be recorded as fixed-in-time facts, not living
documents open to later modification. A GitHub wiki was the first
choice precisely because it sits outside the code review/versioning
flow. However, a GitHub wiki's git repository does not exist until a
first page is created interactively through the web UI, so it cannot be
bootstrapped or maintained from automation; wiki pages are also
silently editable by anyone with write access, which contradicts the
fixed-in-time requirement rather than serving it.

## Decision

Keep decision records **in the repository** under `docs/decisions/` as
numbered ADRs, with a strict convention (see the directory README):
once merged as Accepted, a record's content is never edited; a changed
decision gets a *new* superseding ADR, and the old record only gains a
`Superseded by` status line.

## Consequences

- Immutability is enforced by convention and code review, not by
  tooling; reviewers must reject diffs that rewrite accepted records.
- Decisions travel with the code (clones, forks, offline) and their
  history is auditable via git.
- The wiki remains unused; if GitHub ever allows non-interactive wiki
  bootstrap, this decision may be revisited by a superseding ADR.
