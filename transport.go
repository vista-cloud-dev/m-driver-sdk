package mdriver

import "context"

// Transport is the frozen verb-level seam between a driver's vendor logic and
// its engine. It is deliberately NOT a low-level run(argv): the two driver
// shapes it must fit cannot share an argv seam (risk B1) —
//
//   - m-ydb local/docker: pipe M into a `yottadb` session (stdin → stdout),
//     compile implicit;
//   - m-iris local/docker: pipe ObjectScript into `iris session -U NS`, compile
//     via $SYSTEM.OBJ.Load; m-iris remote: Atelier PUT + action/compile + a SQL
//     action/query into a role-gated runner class — NO raw "run" endpoint, no
//     stdout, results returned through a result global the transport reads.
//
// A verb-level interface lets each transport implement its own strategy while
// the rest of the driver stays transport-agnostic. Frozen at the Phase-0
// checkpoint against both shapes.
type Transport interface {
	// Health is the readiness/liveness probe behind `lifecycle status --probe`
	// and `wait` (driver-contract.md §3, plan §3). YDB: `%XCMD 'write 1'` → 1;
	// IRIS remote: GET /api/atelier/v1/ → 200 + version.
	Health(ctx context.Context) (Health, error)

	// Load stages routine source and compiles it (exec.load: stage + compile).
	// YDB: copy .m onto $ydb_routines (compile is implicit on first run);
	// IRIS: $SYSTEM.OBJ.Load(path,"ck") / Atelier PUT + action/compile.
	Load(ctx context.Context, req LoadRequest) (LoadResult, error)

	// Exec runs an entryref, evaluates a command, or runs a direct-mode script
	// (exec.run / exec.eval). On a compile/runtime fault it returns the fault in
	// ExecResult.EngineError, NOT as a Go error — the fault is data (§7); a Go
	// error means the transport itself failed (could not reach/launch).
	Exec(ctx context.Context, req ExecRequest) (ExecResult, error)

	// ReadGlobal reads a global node, or a subtree per Depth/Order — data.get /
	// data.query, and the result-global read that backs exec/cover orchestration.
	ReadGlobal(ctx context.Context, req GlobalRef) (GlobalNode, error)

	// SetGlobal sets a single global node (data.set), used to seed fixtures. It
	// is a first-class verb (not Exec of a "set" command) so the IRIS remote
	// substrate can route it through a parameterized, role-gated runner method
	// rather than splicing values into ObjectScript.
	SetGlobal(ctx context.Context, ref, value string) error
}

// Health is the probe result (driver-contract.md §3 health probes).
type Health struct {
	Running   bool   `json:"running"`
	Healthy   bool   `json:"healthy"`
	Version   string `json:"version,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}

// LoadRequest stages source for exec (contract input `<paths…> | <dir>`). Paths
// are explicit files; Dir is an alternative directory of source. Prefix (e.g.
// zzt<runid>) namespaces an ephemeral run so teardown is scoped to that prefix.
type LoadRequest struct {
	Paths  []string
	Dir    string
	Prefix string
}

// LoadResult reports what was staged + compiled; EngineError carries a compile
// fault (§7) when staging compiled routines fails.
type LoadResult struct {
	Loaded      []string     `json:"loaded"`
	EngineError *EngineError `json:"engineError,omitempty"`
}

// ExecRequest runs one of three shapes, selected by which field is set rather
// than an explicit mode enum (the union of both drivers' needs). Precedence when
// more than one is set: Script, then EntryRef, then Command.
//
//   - Script  → a multi-line M/ObjectScript script (YDB `-direct`; the
//     transport appends the terminating halt). Engines without a direct mode
//     ignore it.
//   - EntryRef→ run an entryref (`yottadb -run <ref>` / IRIS `do <ref>`); Args
//     become $ZCMDLINE / the entry's formallist.
//   - Command → evaluate a single M command (YDB `%XCMD` / IRIS `xecute`).
type ExecRequest struct {
	Script   string
	EntryRef string
	Args     []string
	Command  string
	Stdin    string // optional principal-device input
	Prefix   string // ephemeral-run prefix (zzt<runid>)
}

// ExecResult is the unified outcome. Stdout is the captured device output
// (session transports) or the runner's result-global text (IRIS remote).
// EngineError, when non-nil, is the §7 structured fault — set instead of a Go
// error so the caller can render a RED-with-cause envelope.
type ExecResult struct {
	Stdout      string       `json:"stdout"`
	Status      int          `json:"status"`
	EngineError *EngineError `json:"engineError,omitempty"`
}

// GlobalRef addresses a global for a read. Ref is the full global reference
// (leading ^ optional), e.g. `^ycov("RTN",12)`. Order/Depth shape a subtree
// query (data.query); the zero value (empty Order, Depth 0) is a single-node get.
type GlobalRef struct {
	Ref   string
	Order string // "forward" | "reverse"
	Depth int    // 0 = this node only
}

// GlobalNode is a global value, with children for a subtree read.
type GlobalNode struct {
	Ref   string       `json:"ref"`
	Value string       `json:"value,omitempty"`
	Nodes []GlobalNode `json:"nodes,omitempty"`
}
