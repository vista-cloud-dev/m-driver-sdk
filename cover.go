package mdriver

// This file holds the shared payload shape for the cover axis — the JSON m-cli
// reads identically from every driver. It lives here (not per-driver) so m-ydb
// and m-iris cannot drift in field names or tags; each driver builds it from its
// own line-tracer output.

// CoverResult is the `cover trace` payload (driver-contract.md §5.5). The driver
// runs the entryref under the engine's line tracer, reconciles the raw per-line
// hit data against the executable-line set, and emits an LCOV tracefile plus the
// rolled-up line totals. Both engines converge on executable-line granularity:
//
//   - m-ydb:  view "TRACE":1:"^ycov" … zwrite ^ycov (label-relative offsets,
//     reconciled to absolute lines via the parse tree);
//   - m-iris: %Monitor.System.LineByLine → MLINE:<routine>:<line>:<count>
//     (already absolute); remote rides the runner class.
//
// The neutral shape is intentionally line-coverage only (LCOV DA/LF/LH) — neither
// engine natively emits function or branch coverage. m-cli aggregates LCOV across
// drivers and applies --min-percent; the counts here let it gate without re-parsing.
//
// All fields always render: a zero is meaningful (0% covered is a fact, not
// absence), the same convention as Status.LatencyMs and the Features flags.
type CoverResult struct {
	LCOV         string  `json:"lcov"`
	CoveredLines int     `json:"coveredLines"`
	TotalLines   int     `json:"totalLines"`
	LinePercent  float64 `json:"linePercent"`
}
