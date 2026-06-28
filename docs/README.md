# m-driver-sdk docs

`m-driver-sdk` is the coordination sync point (D0) for the m-engine-drivers
effort — it owns the neutral contract types and is the orchestrator that keeps
`m-ydb` and `m-iris` parallel and consistent. This folder holds the live
cross-repo status; the normative contract itself lives in the central `docs`
repo (see below).

## Layout

- [`implementation-tracker.md`](implementation-tracker.md) — **live** Tier-D
  tracker: the single source of truth for per-repo, per-milestone status across
  `m-driver-sdk` / `m-ydb` / `m-iris`. Reconcile it on every cross-repo change.
- `archive/` — retired docs from this repo (kept, never deleted):
  - [`archive/continue-implementation.md`](archive/continue-implementation.md) —
    archived fresh-session kickoff prompt to resume the implementation.

## Normative spec (read-only, central `docs` repo)

`~/vista-cloud-dev/docs/m-engine-drivers/` — `driver-contract.md` (v1.0),
`driver-implementation-plan.md` (§4 m-ydb, §5 m-iris), `engine-command-survey.md`,
`coordination-model.md`, `phase-0-reconciliation.md`.

Shared coordination memory is **not** kept here — it lives in the central `docs`
repo's `docs/memory/` (recalled by the coordinator session).
