# m-engine-drivers — implementation tracker

**`m-driver-sdk` is the OWNER and ORCHESTRATOR of the m-engine-drivers implementation.**
This document is the single source of truth for cross-repo status. Every change in any of the
three repos must reconcile here: when you land work, update the tables below in the SAME change
(or immediately after) so the three implementations stay parallel and consistent.

> Three repos, one system. m-cli speaks only the neutral contract; the two drivers must look
> identical to it. The SDK + this tracker are the coordination point — shared shapes change here
> first, both drivers pin the same SDK version, and every stage is validated against BOTH real
> engines.

Last reconciled: **2026-06-06** (SDK M5 `CoverResult` shape defined on `coordination`,
gates green, untagged — staged for the v0.3.0 release that unblocks M5 cover on both drivers).

Legend: ☑ done · ◐ in progress / partial · ☐ not started · — n/a · 🔒 gated (needs a resource)

---

## 1. Repo snapshot

| Repo | Role | GitHub (public, vista-cloud-dev) | Branch | SDK pin | Head | Real-engine tier |
|---|---|---|---|---|---|---|
| **m-driver-sdk** | contract + Transport, **orchestrator** | `m-driver-sdk` | `main` (tags v0.1.0, **v0.2.0**) | — | `f5ad91f` | — |
| **m-ydb** | D2 — YottaDB driver | `m-ydb` | `main` | **v0.2.0** | _M3 exec done_ | `m-test-engine` (YottaDB r2.07) ☑ |
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
| **M2** | `sync` (source axis) | — (filesystem / Atelier; no shared payload) | ☑ full axis (list/pull/status/verify/diff/push/deploy/rm + bare-name `--filter`, filesystem-native over `$ydb_routines`; push conflict-checked, deploy `--prune` common-prefix guard) | ◐ verbs exist (list/pull/status/verify/push/deploy); add diff/rm + bare-name `--filter` |
| **M3** | `exec` (load/run/eval/abort) + `engineError` | ☑ `ExecRequest`/`ExecResult`, `EngineError`, `LoadRequest`/`LoadResult` | ☑ load/run/eval/abort + engineError (runtime via $ETRAP→$ZSTATUS over `%XCMD`; compile via ZLINK stderr listing; ephemeral `--prefix`; real r2.07) | ◐ remote substrate spike ☑ (real IRIS); wire exec/eval/abort commands |
| **M4** | `data` (get/set/kill/query/export/import) | ☑ `GlobalRef`/`GlobalNode`, `SetGlobal` (export/import shape TBD) | ☐ | ☐ |
| **M5** | `cover` (trace → LCOV) | ◐ `CoverResult` (lcov/coveredLines/totalLines/linePercent) defined on `coordination`, untagged | ☐ | ☐ |
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
| `CoverResult` (lcov/coveredLines/totalLines/linePercent) | **v0.3.0 (staged)** | M5 cover payload (contract §5.5); on `coordination`, untagged. No new Transport verb — driver composes `Exec`+`ReadGlobal` |
| admin result shape | — | **pending (M6)** |

**Versioning / repin protocol:** add types/fields → minor bump; change/remove → major (contract §8).
To bump: edit SDK → commit → `git tag vX.Y.Z` + push tag → in BOTH drivers
`go get github.com/vista-cloud-dev/m-driver-sdk@vX.Y.Z` (default proxy, so go.sum is sumdb-verified
for CI) → `go mod tidy` → test → commit `deps: pin … vX.Y.Z`. Both drivers always pin the SAME version.

---

## 4. Per-repo next actions

**m-ydb (D2)** — **M2 `sync` COMPLETE** (plan §4 tasks 6+7): full axis
`list/pull/status/verify/diff/push/deploy/rm` + bare-name `--filter`, filesystem-native over
`$ydb_routines`. `internal/{manifest,mirror,source,udiff}`; `source.Store` (List/Read/Write/Remove) =
host-fs `FileStore` (local) | container `ShellStore` (docker, over `transport.Session.Sh`; write via
base64→`base64 -d`). push is conflict-checked vs the manifest (`manifest.CheckConflict`, exit 4 unless
`--force`) and lands content in both mirror + instance; deploy installs a library with a common-prefix
prune guard (refuses prune with no common prefix); diff = LCS unified (`internal/udiff`); rm clears
instance+mirror+manifest. caps advertises all 8 verbs; golden regenerated. Validated vs real YottaDB
r2.07 via `make test-it` (ShellStore list/read + write/read/remove round-trip, arbitrary bytes).
**M3 exec ☑ COMPLETE** (plan §4 tasks 8,9,10): `load`/`run`/`eval`/`abort` + `engineError`.
`ExecTrapped` xecutes the work under a `$ETRAP` via `%XCMD` (clean non-direct context; `-direct` prints
`YDB>` prompts and bypasses the trap), capturing sentinel-delimited `$ZSTATUS` → §7 EngineError (mnemonic
by `%FAC-S-NAME` regex so a `%`-routine location isn't mis-read). The configured routine path is layered
onto `$ZROUTINES` at runtime (not the env) so staged routines resolve while `%XCMD` stays linked.
`Compile` (ZLINK) surfaces **compile** faults from the stderr listing (`parseCompileError`; ZLINK writes
the listing to stderr with exit 0, no trap). `exec load` stages (store.Write) + compiles; `run`/`eval`
→ {stdout,status}; `abort --prefix` greps the `;<prefix>` marker filtered to yottadb procs via
`/proc/<pid>/comm` → `mupip stop`. Validated vs real r2.07 (run staged→"HI42", `%YDB-E-LVUNDEF`,
`%YDB-E-INVCMD` compile, abort no-op). caps `exec:[load,run,eval,abort]`. NEXT: **M5 cover**
(`view "TRACE":1:"^ycov"`→LCOV; port m-cli/internal/mcov) and/or **M4 data**. Deferred: `lifecycle logs`
(parity with m-iris). Granular tasks: plan §4.

**m-iris (D1)** — M0+M1 done & real-IRIS-validated; next **finish M2 `sync`** (add diff/rm +
bare-name `--filter`; regroup is done) and **M3 exec** (wire run/eval/abort onto the proven remote
runner substrate; also build local/docker `iris session` transports). Granular tasks: plan §5.
Open: PR #1 (m-iris-driver → main) — merge when ready.

**m-driver-sdk (orchestrator)** — M5 (LCOV) shared shape **DONE on `coordination`** (untagged):
`CoverResult{lcov,coveredLines,totalLines,linePercent}` in `cover.go` (contract §5.5), gates green
(`go test -race`, vet, gofmt). No new Transport verb — `cover trace` composes the existing
`Exec`+`ReadGlobal` (YDB `view "TRACE"`→`zwrite ^ycov`; IRIS monitor→runner). NEXT (user-gated
release ceremony): merge `coordination`→`main`, `git tag v0.3.0` + push tag, then repin BOTH drivers
(`go get …@v0.3.0` → tidy → test) — this unblocks **M5 cover** on m-ydb and m-iris. Still pending:
define the M6 (admin) shape when that milestone starts; keep this tracker + the contract reconciled;
own `m-driver-conformance` (D5).

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
