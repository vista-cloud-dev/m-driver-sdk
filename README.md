# m-driver-sdk — the shared engine-driver SDK for the `m` toolchain

`m-driver-sdk` (Go package `mdriver`) is the **single coupling point** between
m-cli and the engine drivers. It encodes the vendor-neutral
[engine-driver contract](../docs/m-engine-drivers/driver-contract.md) v1.0 as Go
types and the **verb-level `Transport`** interface every `m-<engine>` driver
implements. m-cli speaks only the contract; all proprietary detail lives behind
the `Transport` in each driver (`m-ydb`, `m-iris`).

It was extracted at the **Phase-0 reconciliation checkpoint** by freezing the two
driver `Transport` sketches against one interface — see
[`phase-0-reconciliation.md`](../docs/m-engine-drivers/phase-0-reconciliation.md).

## Orchestration

This repo is the **owner and orchestrator** of the m-engine-drivers implementation. Cross-repo
status and the protocols that keep m-ydb + m-iris parallel and consistent live here:

- [`docs/implementation-tracker.md`](docs/implementation-tracker.md) — the single source of truth
  for per-repo, per-milestone status. Reconcile it on every cross-repo change.
- [`docs/prompts/continue-implementation.md`](docs/prompts/continue-implementation.md) — paste into
  a fresh session to resume work.

## What's in it

| Symbol | Purpose |
|---|---|
| `Transport` | the frozen verb-level seam: `Health · Load · Exec · ReadGlobal · SetGlobal` |
| `ExecRequest`/`ExecResult` | field-dispatched exec (Script > EntryRef > Command); `EngineError` carries §7 faults |
| `LoadRequest`/`LoadResult`, `GlobalRef`/`GlobalNode`, `Health` | the request/result types |
| `EngineError` | the §7 structured engine fault (mnemonic/routine/line/text) |
| `Caps`/`Axes`/`Features`, `ContractVersion`, `Transport*` consts | the capability document (§4); `Axes.Wired()` iterates advertised axes in contract order |
| `FakeTransport` | the function-field fake for driver unit tests (no engine) |

## Design rules

- **Verb-level, not `run(argv)`** (risk B1). A low-level argv seam cannot model
  both YottaDB's session-pipe (stdin→stdout) and IRIS's Atelier-SQL remote
  (HTTP PUT + SQL; results via a result global). Each transport implements its
  own strategy; the rest of a driver is transport-agnostic.
- **Vendor-neutral only.** No YottaDB/IRIS specifics here.
- **No clikit dependency.** clikit (the toolchain envelope/styling layer) is
  vendored into every toolchain repo, including non-driver ones; a clikit→SDK
  edge would couple the whole toolchain to the driver SDK. `EngineError` is
  therefore defined here for transport results, and clikit keeps its own copy
  for the JSON envelope — drivers convert at the boundary.
- **Versioning.** Adding verbs/fields is a back-compatible minor bump; changing
  or removing one is a major bump (contract §8). `ContractVersion` is the
  handshake m-cli negotiates on.

## Consuming it (local development)

Until the module is published, drivers depend on it via a local replace:

```
require github.com/vista-cloud-dev/m-driver-sdk v0.0.0
replace github.com/vista-cloud-dev/m-driver-sdk => ../m-driver-sdk
```

Drop the `replace` and pin a tagged version once it is published.
