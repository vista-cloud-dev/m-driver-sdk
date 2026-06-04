# m-engine-drivers — implementation tracker

**`m-driver-sdk` is the OWNER and ORCHESTRATOR of the m-engine-drivers implementation.**
This document is the single source of truth for cross-repo status. Every change in any of the
three repos must reconcile here: when you land work, update the tables below in the SAME change
(or immediately after) so the three implementations stay parallel and consistent.

> Three repos, one system. m-cli speaks only the neutral contract; the two drivers must look
> identical to it. The SDK + this tracker are the coordination point — shared shapes change here
> first, both drivers pin the same SDK version, and every stage is validated against BOTH real
> engines.

Last reconciled: **2026-06-04** (m-ydb M2 read-side slice landed).

Legend: ☑ done · ◐ in progress / partial · ☐ not started · — n/a · 🔒 gated (needs a resource)

---

## 1. Repo snapshot

| Repo | Role | GitHub (public, vista-cloud-dev) | Branch | SDK pin | Head | Real-engine tier |
|---|---|---|---|---|---|---|
| **m-driver-sdk** | contract + Transport, **orchestrator** | `m-driver-sdk` | `main` (tags v0.1.0, **v0.2.0**) | — | `f5ad91f` | — |
| **m-ydb** | D2 — YottaDB driver | `m-ydb` | `main` | **v0.2.0** | _M2 read-side_ | `m-test-engine` (YottaDB r2.07) ☑ |
| **m-iris** | D1 — IRIS driver (was irissync) | `m-iris` | `m-iris-driver` (PR #1 → main, OPEN) | **v0.2.0** | `a502071` | `m-test-iris` (IRIS CE 2026.1) ☑ |

Local paths: `~/vista-cloud-dev/{m-driver-sdk,m-ydb,m-iris}`. Both drivers depend on the SDK by
tagged version through the public Go proxy (no `replace`).

---

## 2. Milestone × repo status (the master rollup)

Milestone ladder from `docs/m-engine-drivers/driver-implementation-plan.md` §2. The SDK column is
the shared shape/seam each milestone needs (defined once, consumed by both drivers).

| M | Axis / concern | m-driver-sdk (shared shape) | m-ydb | m-iris |
|---|---|---|---|---|
| **M0** | scaffold + `meta` (caps/version/info/schema) + transport seam | ☑ `Transport`, `Caps/Axes/Features`, `EngineError`, `FakeTransport`, `ContractVersion` | ☑ | ☑ |
| **M1** | `lifecycle` + health probes + `doctor` | ☑ `Status`, `StateResult`, `Check`, `DoctorResult`, `Axes.Wired()` (**v0.2.0**) | ☑ (real YDB) | ☑ (real IRIS) |
| **M2** | `sync` (source axis) | — (filesystem / Atelier; no shared payload) | ◐ read side ☑ (list/pull/status/verify + bare-name `--filter`, filesystem-native over `$ydb_routines`); write side (push/deploy/diff/rm) **NEXT** | ◐ verbs exist (list/pull/status/verify/push/deploy); add diff/rm + bare-name `--filter` |
| **M3** | `exec` (load/run/eval/abort) + `engineError` | ☑ `ExecRequest`/`ExecResult`, `EngineError`, `LoadRequest`/`LoadResult` | ☐ | ◐ remote substrate spike ☑ (real IRIS); wire exec/eval/abort commands |
| **M4** | `data` (get/set/kill/query/export/import) | ☑ `GlobalRef`/`GlobalNode`, `SetGlobal` (export/import shape TBD) | ☐ | ☐ |
| **M5** | `cover` (trace → LCOV) | ☐ LCOV result shape TBD | ☐ | ☐ |
| **M6** | `admin` (backup/restore/check/journal) | ☐ neutral result shape TBD | ☐ | ☐ |
| **M7** | `native` passthrough (full backend surface) | — (per-engine; not in contract) | ☐ | ☐ |
| **M8** | conformance green on all transports + CI matrix | ☐ `m-driver-conformance` (component D5) | ☐ | ☐ |

Per-driver critical path: **M0 → M1 → M2(staging) → M3 → M5 → M8**; M4/M6 after M3; M7 any time
after M0. (m-ydb: no `remote`. m-iris: `remote` attach-mode — provision/destroy unsupported there.)

---

## 3. SDK API surface (package `mdriver`) — what exists vs pending

`ContractVersion = "1.0"`. Current tag: **v0.2.0**.

| Symbol | Since | Notes |
|---|---|---|
| `Transport` (Health/Load/Exec/ReadGlobal/SetGlobal) | v0.1.0 | the frozen verb-level seam |
| `ExecRequest`/`ExecResult`, `LoadRequest`/`LoadResult` | v0.1.0 | field-dispatched exec (Script>EntryRef>Command) |
| `GlobalRef`/`GlobalNode`, `Health` | v0.1.0 | |
| `EngineError` | v0.1.0 | §7 fault; clikit keeps its own copy for the envelope (drivers convert at the boundary) |
| `Caps`/`Axes`/`Features`, transport consts, `FakeTransport` | v0.1.0 | `Axes` is struct+omitempty (honest caps) |
| `Axes.Wired()`, `Check`, `DoctorResult`, `Status`, `StateResult` | **v0.2.0** | M1 doctor + lifecycle payloads |
| LCOV/coverage result shape | — | **pending (M5)** |
| admin result shape | — | **pending (M6)** |

**Versioning / repin protocol:** add types/fields → minor bump; change/remove → major (contract §8).
To bump: edit SDK → commit → `git tag vX.Y.Z` + push tag → in BOTH drivers
`go get github.com/vista-cloud-dev/m-driver-sdk@vX.Y.Z` (default proxy, so go.sum is sumdb-verified
for CI) → `go mod tidy` → test → commit `deps: pin … vX.Y.Z`. Both drivers always pin the SAME version.

---

## 4. Per-repo next actions

**m-ydb (D2)** — M2 **read side done** (plan §4 task 6): `sync list/pull/status/verify` +
bare-name `--filter`, filesystem-native over `$ydb_routines` (new `internal/{manifest,mirror,source}`;
`source.Store` = host-fs `FileStore` (local) | container `ShellStore` (docker, over the session shell);
caps advertises `sync:[list,pull,status,verify]`; golden regenerated; validated vs real YottaDB r2.07
via `make test-it`). NEXT: **M2 write side** (plan §4 task 7) — `push --from`/`deploy --prune`/`diff`/`rm`
(copy `.m`; prune under common-prefix guard; reuse `manifest.Conflict`-style guard). Then M3 exec
(`yottadb -run`/`%XCMD`/`-direct`, `$ZSTATUS`→engineError), M5 cover (`view "TRACE"`→LCOV).
Deferred: `lifecycle logs` (kept off for parity with m-iris). Granular tasks: plan §4.

**m-iris (D1)** — M0+M1 done & real-IRIS-validated; next **finish M2 `sync`** (add diff/rm +
bare-name `--filter`; regroup is done) and **M3 exec** (wire run/eval/abort onto the proven remote
runner substrate; also build local/docker `iris session` transports). Granular tasks: plan §5.
Open: PR #1 (m-iris-driver → main) — merge when ready.

**m-driver-sdk (orchestrator)** — define the M5 (LCOV) and M6 (admin) shared shapes when those
milestones start; keep this tracker + the contract reconciled; own `m-driver-conformance` (D5).

---

## 5. Hard rules (every session honors these)

1. **TDD always** — failing table-driven test (fake Transport / golden / argv) → red → implement →
   green → `go test -race`. No implementation before a failing test.
2. **Dual-engine real validation each stage** — after a milestone slice, run that driver's gated
   real-engine tier (`make test-it`) against BOTH real engines and add gated tests for new verbs.
   See [`docs/prompts/continue-implementation.md`](prompts/continue-implementation.md) for the
   container setup. Unit tests use fakes; reality is the gate.
3. **Shared shapes live in the SDK** — any JSON/result shape m-cli reads from BOTH drivers is
   defined once in `mdriver`, then consumed (drivers may alias). Never hand-duplicate divergently.
4. **clikit stays byte-identical** across m-ydb and m-iris (except the version.go ldflags comment).
5. **caps stays honest** — advertise only wired verbs; grow per milestone; regenerate goldens.
6. **Reconcile here** — update §1/§2/§3 of this tracker on every cross-repo change, and the
   normative `driver-contract.md` / `driver-implementation-plan.md` when a contract/plan item moves.

---

## 6. Source-of-truth references

- Normative spec (read-only): `~/vista-cloud-dev/docs/m-engine-drivers/` — `driver-contract.md`
  (v1.0), `driver-implementation-plan.md` (§4 m-ydb, §5 m-iris tracking), `engine-command-survey.md`,
  `driver-plan-risk-assessment.md`, `phase-0-reconciliation.md`.
- This SDK repo: `README.md` (SDK overview), `docs/implementation-tracker.md` (this file),
  `docs/prompts/continue-implementation.md` (fresh-session kickoff).
- Session memory (auto-loaded): `m-engine-drivers-consistency-protocol`,
  `m-engine-drivers-real-engine-testing`, `m-driver-sdk-phase0`, `m-ydb-driver-m0`,
  `m-iris-driver-m0-spike`.
