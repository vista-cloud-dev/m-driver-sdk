# m-driver-sdk — the coordination sync point (D0). Repo rules.

Adds to the org rules (`~/vista-cloud-dev/CLAUDE.md`). This repo is the **only
coupling** between `m-iris` and `m-ydb`: it owns the neutral contract types, the
`Transport` interface, `EngineError`, `Caps`/`Axes`/`Features`, and the
`ContractVersion`. A change here ripples to **both** drivers and to m-cli. Read
[[coordination-model]] (`docs/m-engine-drivers/coordination-model.md` in `docs`).

## This is a COORDINATOR session — serialize SDK changes
- Editing this repo is a **coordination event**, never done concurrently with a
  driver spike that will consume the change. Do SDK work in a coordinator session
  (org-root or here), with no parallel iris/ydb session mid-flight on the new shape.
- **Releases are tagged and batched.** Don't dribble shared shapes out continuously:
  collect what both drivers need for the next milestone, land it once, then
  `git tag vX.Y.Z` + push the tag, and repin BOTH drivers
  (`go get github.com/vista-cloud-dev/m-driver-sdk@vX.Y.Z` → tidy → test) together.
  Semver: add types/fields = minor; change/remove = major (contract §8). Now v0.2.0.
- **Keep `clikit` byte-identical** across m-ydb and m-iris (it is vendored per-repo;
  m-cli has its own copy too). `EngineError` lives here for `ExecResult`; clikit keeps
  its own copy for the JSON envelope — convert at the boundary, never make clikit
  import the SDK.
- Update `docs/m-engine-drivers/driver-contract.md` whenever a shared shape/verb
  changes, and roll up the shared plan §4/§5 trackers — the coordinator owns those.

## Increment Protocol here
Run it per the org rules. Branch off `main` for SDK feature work; merge to `main`
then tag the release (merging/PRs stay explicit user actions). Coordinator/SDK
memory is **shared** — it lives in the `docs` repo's `docs/memory/` (an SDK session's
recall path is symlinked there), not in this repo. Gate before committing:
`go test -race ./...`, `go vet`, `gofmt`.
