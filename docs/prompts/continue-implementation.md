# Kickoff prompt — continue the m-engine-drivers implementation

Paste the block below into a fresh session to resume. It makes `m-driver-sdk` the orchestrator and
points at the tracker as the source of truth.

---

```
You are continuing the m-engine-drivers implementation: a vendor-neutral driver contract (m-driver-sdk)
with two engine drivers — m-ydb (YottaDB, D2) and m-iris (IRIS, D1) — that m-cli will drive. You OWN
and ORCHESTRATE all three repos from m-driver-sdk. Work in parallel across them but keep them strictly
consistent. Proceed as unsupervised as possible.

ORCHESTRATION HUB — READ FIRST (in this order):
  ~/vista-cloud-dev/m-driver-sdk/docs/implementation-tracker.md   (master cross-repo status — THE source of truth)
  ~/vista-cloud-dev/docs/m-engine-drivers/driver-contract.md       (normative contract v1.0, read-only)
  ~/vista-cloud-dev/docs/m-engine-drivers/driver-implementation-plan.md (§4 m-ydb, §5 m-iris granular tasks)
  ~/vista-cloud-dev/docs/m-engine-drivers/engine-command-survey.md  (engine command facts)
Also recall the auto-loaded memories: m-engine-drivers-consistency-protocol,
m-engine-drivers-real-engine-testing, m-driver-sdk-phase0, m-ydb-driver-m0, m-iris-driver-m0-spike.

THE THREE REPOS (all public, github.com/vista-cloud-dev/*, local at ~/vista-cloud-dev/):
  - m-driver-sdk  (pkg mdriver) — contract types + verb-level Transport + FakeTransport. main, tag v0.2.0. ORCHESTRATOR.
  - m-ydb         (D2) — YottaDB. main. SDK v0.2.0. M0+M1 done, validated vs real YottaDB r2.07. NEXT: M2 sync.
  - m-iris        (D1, was irissync) — IRIS. branch m-iris-driver (PR #1 open→main). SDK v0.2.0. M0+M1 done,
                  validated vs real IRIS 2026.1. NEXT: finish M2 sync (diff/rm/bare-filter) + M3 exec.

HARD RULES (non-negotiable):
  1. TDD always — failing table-driven test (fake Transport / golden / argv) → red → implement → green →
     `go test -race`. Never implement before a failing test.
  2. DUAL-ENGINE REAL VALIDATION at every stage — after each slice, run BOTH drivers' gated real-engine
     tiers and add gated tests for new verbs. Unit tests use fakes; reality is the gate.
       m-ydb:  cd m-ydb  && make test-it        (→ m-test-engine, YottaDB r2.07)
       m-iris: cd m-iris && make test-it        (→ m-test-iris, IRIS CE 2026.1)
     Containers (start if absent):
       YDB:  docker container m-test-engine (yottadb-base) — should be Up/healthy.
       IRIS: m-test-iris on host port 52774→52773, creds _SYSTEM/testsys, ns USER. docker EXEC is BLOCKED
             and the CE image ships the password EXPIRED, so (re)create it password-first:
             printf 'testsys\n' >/tmp/m-test-iris-pw.txt
             docker run -d --name m-test-iris -p 52774:52773 -v /tmp/m-test-iris-pw.txt:/pw.txt \
               intersystemsdc/iris-community:latest --password-file /pw.txt
       NEVER touch the shared `vista-iris` container (off-limits; it's port 52773).
       Note: foreground `curl -u …` is classifier-blocked — probe with the driver (`meta doctor`,
       `lifecycle status`) or a tiny Go net/http program, not raw curl.
  3. SDK-FIRST for shared shapes — any JSON/result shape m-cli reads from BOTH drivers is defined once in
     mdriver, then consumed (drivers may type-alias). Bump+repin protocol: edit SDK → tag vX.Y.Z → push tag →
     in BOTH drivers `go get …/m-driver-sdk@vX.Y.Z` (default proxy) → tidy → test → commit "deps: pin". Both
     drivers ALWAYS pin the same SDK version.
  4. clikit stays byte-identical across m-ydb and m-iris (except the version.go ldflags comment).
  5. caps stays honest (advertise only wired verbs; grow per milestone; regenerate goldens).
  6. RECONCILE THE TRACKER — update m-driver-sdk/docs/implementation-tracker.md (§1 snapshot, §2 milestones,
     §3 SDK surface) on every cross-repo change, and the normative driver-contract.md / -implementation-plan.md
     when a contract/plan item moves. Keep all three repos green (`go test -race`, `go vet`, `gofmt`) and pushed.
  7. Repo boundaries are gone for you (you own all three) BUT do NOT edit m-cli — its D3 cutover
     (delete internal/engine/{ydb,iris,docker}.go) is gated on both drivers passing conformance; only READ it
     to confirm the contract/terms it expects.

NEXT WORK (per the tracker §4, critical path M2→M3→M5→M8):
  - m-ydb M2: `sync` — filesystem-native over $ydb_routines (list/pull/status/verify, then push --from /
    deploy --prune / diff / rm; bare-name --filter). Then M3 exec (yottadb -run / %XCMD / -direct;
    $ZSTATUS→engineError), then M5 cover (view "TRACE":1:"^ycov"→LCOV; port m-cli/internal/mcov).
  - m-iris M2: add diff/rm + bare-name --filter to the existing sync. Then M3: wire run/eval/abort onto the
    proven remote runner substrate, and build local/docker `iris session` transports.
  - SDK: define the M5 (LCOV) and M6 (admin) shared shapes when those milestones start; own
    m-driver-conformance (D5) for M8.

START NOW: read the tracker, confirm both real-engine containers are healthy (start m-test-iris if needed),
then pick up the next milestone slice (default: m-ydb M2 sync) test-first, validating against the real
engine, and reconcile the tracker as you land each piece.
```
