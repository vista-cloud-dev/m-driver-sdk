package mdriver

// This file holds the shared payload shapes for the lifecycle and meta-doctor
// axes — the JSON m-cli reads identically from every driver. They live here (not
// per-driver) so m-ydb and m-iris cannot drift in field names or tags; a driver
// builds them from its own transport/connection facts.

// Check is one doctor preflight result (driver-contract.md §5.7, plan §3): a
// typed, named diagnostic with an optional human detail and a fix hint.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// DoctorResult is the `meta doctor` payload: the resolved transport, an overall
// ok (all checks green), and the typed checks. Exit code is carried by the
// envelope (0 all-green / 6 unreachable / 5 a check failed), not duplicated here.
type DoctorResult struct {
	Transport string  `json:"transport"`
	OK        bool    `json:"ok"`
	Checks    []Check `json:"checks"`
}

// Status is the `lifecycle status` payload (driver-contract.md §5.1) — the
// liveness/readiness snapshot plus engine identity. Namespaces/Version/Endpoint
// are omitempty so a driver that cannot cheaply report one simply omits it.
type Status struct {
	Transport  string   `json:"transport"`
	Running    bool     `json:"running"`
	Healthy    bool     `json:"healthy"`
	Version    string   `json:"version,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	LatencyMs  int64    `json:"latencyMs"`
	Endpoint   string   `json:"endpoint,omitempty"`
}

// StateResult is the lifecycle up/down/restart payload: the resulting state
// (e.g. "started"/"stopped"/"attached") and, where meaningful, the endpoint.
type StateResult struct {
	State    string `json:"state"`
	Endpoint string `json:"endpoint,omitempty"`
}
