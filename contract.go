// Package mdriver is the shared engine-driver SDK for the m toolchain: the
// vendor-neutral contract types (driver-contract.md v1.0) and the verb-level
// Transport seam that every m-<engine> driver implements and that m-cli depends
// on. It is the single coupling point between m-cli, m-ydb, and m-iris — m-cli
// speaks only the contract; all proprietary detail lives behind the Transport
// in each driver.
//
// Extracted at the Phase-0 reconciliation checkpoint by freezing the two driver
// Transport sketches (m-ydb's session-pipe and m-iris's Atelier-SQL) against one
// interface. Adding verbs/fields is a back-compatible minor bump; changing or
// removing one is a major bump (driver-contract.md §8).
//
// The package holds ONLY vendor-neutral types — no YottaDB or IRIS specifics.
// clikit (the toolchain envelope/styling layer) is intentionally NOT imported:
// clikit is vendored into every toolchain repo, including non-driver ones, so a
// clikit→SDK dependency would couple the whole toolchain to the driver SDK.
// EngineError is therefore defined here for Transport results; clikit keeps its
// own copy for the JSON envelope and drivers convert at the boundary.
package mdriver

// ContractVersion is the driver-contract major.minor this SDK encodes. A driver
// advertises it in caps.contract; m-cli refuses a driver whose major it does not
// understand, with an upgrade hint (driver-contract.md §8).
const ContractVersion = "1.0"

// Transport selectors (driver-contract.md §3). A driver advertises which it
// supports via caps.transports; YottaDB supports local+docker, IRIS adds remote.
const (
	TransportLocal  = "local"
	TransportDocker = "docker"
	TransportRemote = "remote"
)

// Caps is the capability document emitted by `meta caps` (driver-contract.md §4).
// m-cli probes it before optional verbs and adapts to exactly what is advertised;
// calling an unadvertised verb yields exit 7. Caps MUST be honest — advertise
// only axes/verbs actually wired in the build — and conformance enforces it.
type Caps struct {
	Engine     string   `json:"engine"`
	Contract   string   `json:"contract"`
	Transports []string `json:"transports"`
	Axes       Axes     `json:"axes"`
	Features   Features `json:"features"`
}

// Axes lists the advertised verbs per contract axis (driver-contract.md §5). It
// is a struct (not a map) so the JSON field order is the contract's logical
// order; each axis is omitempty so a driver advertising only its wired axes
// leaves the rest nil → absent (the honest-incremental model both drivers use).
type Axes struct {
	Lifecycle []string `json:"lifecycle,omitempty"`
	Sync      []string `json:"sync,omitempty"`
	Exec      []string `json:"exec,omitempty"`
	Data      []string `json:"data,omitempty"`
	Cover     []string `json:"cover,omitempty"`
	Admin     []string `json:"admin,omitempty"`
	Meta      []string `json:"meta,omitempty"`
}

// AxisVerbs pairs an axis name with its advertised verbs (for iteration).
type AxisVerbs struct {
	Name  string
	Verbs []string
}

// Wired returns the non-empty axes in the contract's logical order, so callers
// (caps text rendering, conformance) can iterate advertised axes without
// re-listing field names. Empty (unwired) axes are skipped.
func (a Axes) Wired() []AxisVerbs {
	var out []AxisVerbs
	for _, av := range []AxisVerbs{
		{"lifecycle", a.Lifecycle}, {"sync", a.Sync}, {"exec", a.Exec},
		{"data", a.Data}, {"cover", a.Cover}, {"admin", a.Admin}, {"meta", a.Meta},
	} {
		if len(av.Verbs) > 0 {
			out = append(out, av)
		}
	}
	return out
}

// Features advertises optional capabilities m-cli negotiates for graceful
// degradation (driver-contract.md §4, §10). All fields are always rendered
// (no omitempty): a false flag is meaningful information, not absence.
type Features struct {
	Remote          bool `json:"remote"`
	Prune           bool `json:"prune"`
	EphemeralPrefix bool `json:"ephemeralPrefix"`
	Snapshot        bool `json:"snapshot"`
}

// EngineError is the structured engine fault (driver-contract.md §7). On any
// compile/runtime fault, exec/cover verbs set ok=false AND populate this so a
// RED suite shows the real cause (e.g. a <NOROUTINE> at a line) instead of
// passed:0, failed:0. Mnemonic carries the engine code: %YDB-E-…/%GTM-E… for
// YottaDB (from $ZSTATUS), or the IRIS <…> code (from $ZERROR).
//
// A driver builds this from the transport result and maps it onto the clikit
// envelope's own EngineError field when rendering a failure.
type EngineError struct {
	Routine  string `json:"routine,omitempty"`
	Line     int    `json:"line,omitempty"`
	Mnemonic string `json:"mnemonic,omitempty"`
	Text     string `json:"text,omitempty"`
}
