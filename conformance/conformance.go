// Package conformance is the executable definition of the m engine-driver
// contract (driver-contract.md §9). It drives a driver *binary* over the same
// subprocess + JSON-envelope seam m-cli uses — `m-<engine> <axis> <verb>
// --output json` — and asserts the driver speaks the contract: well-formed
// envelopes, the exit-code ladder, an honest capability document, and reachable
// meta/lifecycle verbs. A driver is contract-complete only when it passes, which
// is what lets m-cli trust any engine sight unseen (contract §9, coherence C6).
//
// The suite is caps-driven: it reads `meta caps` and exercises only the verbs a
// driver advertises, so it is engine-agnostic and honest by construction. It is
// stdlib-only (the SDK stays dependency-free); the Runner seam lets unit tests
// drive it with canned envelopes and production drive a real binary via
// ExecRunner.
package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
)

// RawResult is one driver invocation's captured stdout and process exit code.
type RawResult struct {
	Stdout []byte
	Exit   int
}

// Runner invokes the driver with the given contract args (e.g. "meta","caps")
// and returns its stdout + process exit. A non-zero engine exit is a result, not
// a Go error; an error means the driver could not be launched at all.
type Runner func(ctx context.Context, args ...string) (RawResult, error)

// Envelope is the clikit JSON envelope every driver writes (contract §2). data is
// decoded per-verb into the SDK shapes.
type Envelope struct {
	SchemaVersion string               `json:"schemaVersion"`
	Command       string               `json:"command"`
	OK            bool                 `json:"ok"`
	Exit          int                  `json:"exit"`
	Data          json.RawMessage      `json:"data"`
	Error         *EnvError            `json:"error,omitempty"`
	EngineError   *mdriver.EngineError `json:"engineError,omitempty"`
}

// EnvError is the envelope's error object (contract §2, ok=false).
type EnvError struct {
	Code    string `json:"code"`
	Exit    int    `json:"exit"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

// Result is one conformance assertion outcome.
type Result struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
}

// Report is the suite outcome for one driver+transport.
type Report struct {
	Transport string   `json:"transport"`
	Engine    string   `json:"engine,omitempty"`
	Results   []Result `json:"results"`
	Pass      int      `json:"pass"`
	Fail      int      `json:"fail"`
}

// validExit is the contract's exit-code ladder (contract §2).
var validExit = map[int]bool{0: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true}

type suite struct {
	run       Runner
	transport string
	results   []Result
	caps      *mdriver.Caps
}

// Run executes the conformance suite against a driver via run, for transport.
func Run(ctx context.Context, run Runner, transport string) Report {
	s := &suite{run: run, transport: transport}
	s.checkCaps(ctx)
	// The remaining checks are caps-driven: only advertised verbs are exercised.
	if s.caps != nil {
		if has(s.caps.Axes.Meta, "version") {
			s.checkVersion(ctx)
		}
		if has(s.caps.Axes.Meta, "doctor") {
			s.checkDoctor(ctx)
		}
		if has(s.caps.Axes.Lifecycle, "status") {
			s.checkLifecycleStatus(ctx)
		}
	}
	return s.report()
}

// call invokes a verb, decodes + validates the universal envelope discipline
// (contract §2), records a pass/fail for that discipline, and returns the parsed
// envelope (nil if the call could not be made or the envelope was unusable).
func (s *suite) call(ctx context.Context, label string, args ...string) *Envelope {
	raw, err := s.run(ctx, args...)
	if err != nil {
		s.fail(label+": launch", fmt.Sprintf("driver could not be launched: %v", err))
		return nil
	}
	var env Envelope
	if jerr := json.Unmarshal(raw.Stdout, &env); jerr != nil {
		s.fail(label+": envelope", fmt.Sprintf("stdout is not a JSON envelope: %v", jerr))
		return nil
	}
	// Universal envelope discipline (contract §2).
	var problems []string
	if env.SchemaVersion == "" {
		problems = append(problems, "missing schemaVersion")
	}
	if env.Command == "" {
		problems = append(problems, "missing command")
	}
	if !validExit[env.Exit] {
		problems = append(problems, fmt.Sprintf("exit %d not on the ladder", env.Exit))
	}
	if env.Exit != raw.Exit {
		problems = append(problems, fmt.Sprintf("envelope.exit %d != process exit %d", env.Exit, raw.Exit))
	}
	if env.OK != (env.Exit == 0) {
		problems = append(problems, fmt.Sprintf("ok=%v inconsistent with exit=%d", env.OK, env.Exit))
	}
	// A non-ok envelope must explain itself — via an error object (the Fail path)
	// OR a data payload (the doctor/lint "data + non-zero exit" report). Bare
	// ok=false with neither is unexplained. A JSON `null` data field (how a nil
	// RawMessage round-trips, since the envelope's data has no omitempty) counts
	// as no data.
	if !env.OK && env.Error == nil && isBlankJSON(env.Data) {
		problems = append(problems, "ok=false but neither error nor data")
	}
	if len(problems) > 0 {
		s.fail(label+": envelope", strings.Join(problems, "; "))
	} else {
		s.pass(label + ": envelope")
	}
	return &env
}

// isBlankJSON reports whether a RawMessage carries no payload: empty, all
// whitespace, or a literal `null` (a nil json.RawMessage marshals to `null`).
func isBlankJSON(r json.RawMessage) bool {
	t := bytes.TrimSpace(r)
	return len(t) == 0 || string(t) == "null"
}

func (s *suite) checkCaps(ctx context.Context) {
	env := s.call(ctx, "caps", "meta", "caps")
	if env == nil {
		return
	}
	if !env.OK {
		s.fail("caps: ok", "meta caps returned ok=false")
		return
	}
	var c mdriver.Caps
	if err := json.Unmarshal(env.Data, &c); err != nil {
		s.fail("caps: parse", fmt.Sprintf("caps data not a capability document: %v", err))
		return
	}
	s.caps = &c
	s.report0("caps: engine", c.Engine != "", "engine name is empty")
	s.report0("caps: contract major",
		major(c.Contract) == major(mdriver.ContractVersion),
		fmt.Sprintf("contract %q major != SDK %q", c.Contract, mdriver.ContractVersion))
	s.report0("caps: transports non-empty", len(c.Transports) > 0, "transports list is empty")
	s.report0("caps: advertises tested transport",
		has(c.Transports, s.transport),
		fmt.Sprintf("transport %q not in advertised %v", s.transport, c.Transports))
	s.report0("caps: features.remote honest",
		c.Features.Remote == has(c.Transports, mdriver.TransportRemote),
		fmt.Sprintf("features.remote=%v but transports remote=%v", c.Features.Remote, has(c.Transports, mdriver.TransportRemote)))
	s.report0("caps: meta self-listing",
		has(c.Axes.Meta, "caps") && has(c.Axes.Meta, "version"),
		"meta axis must list at least caps and version")
	// Honest axes: no axis may be present-but-empty. A JSON `[]` unmarshals to a
	// non-nil empty slice (absent → nil), so non-nil && len 0 = dishonest. (Can't
	// use Axes.Wired(), which filters len>0 and would hide exactly this.)
	emptyAxis := ""
	for _, av := range []struct {
		name string
		v    []string
	}{
		{"lifecycle", c.Axes.Lifecycle}, {"sync", c.Axes.Sync}, {"exec", c.Axes.Exec},
		{"data", c.Axes.Data}, {"cover", c.Axes.Cover}, {"admin", c.Axes.Admin}, {"meta", c.Axes.Meta},
	} {
		if av.v != nil && len(av.v) == 0 {
			emptyAxis = av.name
		}
	}
	s.report0("caps: no empty axes", emptyAxis == "", "axis advertised but empty: "+emptyAxis)
}

func (s *suite) checkVersion(ctx context.Context) {
	env := s.call(ctx, "version", "meta", "version")
	if env == nil || !env.OK {
		s.fail("version: ok", "meta version did not return ok")
		return
	}
	// Only engine + contract are cross-checked against caps (the m-cli-relevant
	// invariant); driver/build shape is left free.
	var v struct {
		Engine   string `json:"engine"`
		Contract string `json:"contract"`
	}
	if err := json.Unmarshal(env.Data, &v); err != nil {
		s.fail("version: parse", err.Error())
		return
	}
	s.report0("version: engine matches caps",
		s.caps == nil || v.Engine == s.caps.Engine,
		fmt.Sprintf("version.engine %q != caps.engine", v.Engine))
	s.report0("version: contract matches caps",
		s.caps == nil || v.Contract == s.caps.Contract,
		fmt.Sprintf("version.contract %q != caps.contract", v.Contract))
}

func (s *suite) checkDoctor(ctx context.Context) {
	env := s.call(ctx, "doctor", "meta", "doctor")
	if env == nil {
		return
	}
	// doctor may legitimately be ok=false (a failed check / unreachable): exit
	// 0/5/6 only. The envelope discipline already checked the ladder + ok⟺exit.
	s.report0("doctor: exit", env.Exit == 0 || env.Exit == 5 || env.Exit == 6,
		fmt.Sprintf("doctor exit %d, want 0/5/6", env.Exit))
	var d mdriver.DoctorResult
	if err := json.Unmarshal(env.Data, &d); err != nil {
		s.fail("doctor: parse", err.Error())
		return
	}
	s.report0("doctor: has checks", len(d.Checks) > 0, "doctor returned no checks")
}

func (s *suite) checkLifecycleStatus(ctx context.Context) {
	env := s.call(ctx, "status", "lifecycle", "status")
	if env == nil {
		return
	}
	var st mdriver.Status
	if err := json.Unmarshal(env.Data, &st); err != nil {
		s.fail("status: parse", err.Error())
		return
	}
	// Healthy must imply running — a driver cannot be healthy-but-down.
	s.report0("status: healthy implies running", !st.Healthy || st.Running,
		"status reports healthy=true but running=false")
}

// --- result bookkeeping ------------------------------------------------------

func (s *suite) pass(name string) { s.results = append(s.results, Result{Name: name, Pass: true}) }
func (s *suite) fail(name, detail string) {
	s.results = append(s.results, Result{Name: name, Pass: false, Detail: detail})
}
func (s *suite) report0(name string, ok bool, failDetail string) {
	if ok {
		s.pass(name)
	} else {
		s.fail(name, failDetail)
	}
}

func (s *suite) report() Report {
	r := Report{Transport: s.transport, Results: s.results}
	if s.caps != nil {
		r.Engine = s.caps.Engine
	}
	for _, res := range s.results {
		if res.Pass {
			r.Pass++
		} else {
			r.Fail++
		}
	}
	return r
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func major(version string) string {
	if i := strings.IndexByte(version, '.'); i >= 0 {
		return version[:i]
	}
	return version
}

// ExecRunner returns a Runner that invokes a real driver binary at path, passing
// `<args…> --transport <transport> --output json` and inheriting the process
// environment (the driver reads its M_<ENGINE>_* connection from there). Stdout
// is captured; stderr is discarded (diagnostics, contract §2).
func ExecRunner(path, transport string) Runner {
	return func(ctx context.Context, args ...string) (RawResult, error) {
		full := append([]string{}, args...)
		full = append(full, "--transport", transport, "--output", "json")
		cmd := exec.CommandContext(ctx, path, full...)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return RawResult{Stdout: out.Bytes(), Exit: ee.ExitCode()}, nil
			}
			return RawResult{}, err
		}
		return RawResult{Stdout: out.Bytes(), Exit: 0}, nil
	}
}
