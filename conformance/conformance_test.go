package conformance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// fakeDriver is a scriptable in-memory driver: it maps a verb path ("meta caps")
// to a canned RawResult, so the suite is exercised without a real binary.
type fakeDriver struct {
	envelopes map[string]RawResult
}

func (f fakeDriver) runner() Runner {
	return func(_ context.Context, args ...string) (RawResult, error) {
		key := strings.Join(args, " ")
		if r, ok := f.envelopes[key]; ok {
			return r, nil
		}
		// Unknown verb → a well-formed "unsupported" envelope (exit 7).
		return env(false, 7, `{}`, key), nil
	}
}

// env builds a RawResult carrying a well-formed clikit envelope.
func env(ok bool, exit int, data, command string) RawResult {
	e := Envelope{SchemaVersion: "1.0", Command: command, OK: ok, Exit: exit, Data: json.RawMessage(data)}
	if !ok {
		e.Error = &EnvError{Code: "X", Exit: exit, Message: "m"}
	}
	b, _ := json.Marshal(e)
	return RawResult{Stdout: b, Exit: exit}
}

// goodDriver is a conformant fake: honest caps + consistent version/doctor/status.
func goodDriver() fakeDriver {
	caps := `{"engine":"ydb","contract":"1.0","transports":["local","docker","remote"],` +
		`"axes":{"lifecycle":["status"],"meta":["caps","version","doctor"]},` +
		`"features":{"remote":true,"prune":true,"ephemeralPrefix":true,"snapshot":false}}`
	version := `{"driver":"m-ydb","engine":"ydb","contract":"1.0","build":"test"}`
	doctor := `{"transport":"local","ok":true,"checks":[{"name":"binary","ok":true}]}`
	status := `{"transport":"local","running":true,"healthy":true,"latencyMs":1}`
	return fakeDriver{envelopes: map[string]RawResult{
		"meta caps":        env(true, 0, caps, "meta caps"),
		"meta version":     env(true, 0, version, "meta version"),
		"meta doctor":      env(true, 0, doctor, "meta doctor"),
		"lifecycle status": env(true, 0, status, "lifecycle status"),
	}}
}

func TestRun_ConformantDriverPasses(t *testing.T) {
	rep := Run(context.Background(), goodDriver().runner(), "local")
	if rep.Fail != 0 {
		t.Fatalf("conformant driver had %d failures: %s", rep.Fail, failNames(rep))
	}
	if rep.Pass == 0 {
		t.Fatal("expected passing checks, got none")
	}
	if rep.Engine != "ydb" {
		t.Errorf("report engine = %q, want ydb", rep.Engine)
	}
}

func TestRun_DishonestCaps_RemoteFeatureMismatch(t *testing.T) {
	d := goodDriver()
	// transports omit remote but features.remote=true → dishonest.
	d.envelopes["meta caps"] = env(true, 0,
		`{"engine":"ydb","contract":"1.0","transports":["local","docker"],`+
			`"axes":{"meta":["caps","version"]},"features":{"remote":true}}`, "meta caps")
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "caps: features.remote honest")
}

func TestRun_ContractMajorMismatch(t *testing.T) {
	d := goodDriver()
	d.envelopes["meta caps"] = env(true, 0,
		`{"engine":"ydb","contract":"2.0","transports":["local"],`+
			`"axes":{"meta":["caps","version"]},"features":{"remote":false}}`, "meta caps")
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "caps: contract major")
}

func TestRun_TransportNotAdvertised(t *testing.T) {
	rep := Run(context.Background(), goodDriver().runner(), "ssh-typo")
	mustFail(t, rep, "caps: advertises tested transport")
}

func TestRun_EnvelopeDiscipline_OkExitMismatch(t *testing.T) {
	d := goodDriver()
	// ok=true but exit=5 — inconsistent.
	bad := env(true, 0, `{"transport":"local","running":true,"healthy":true}`, "lifecycle status")
	var e Envelope
	_ = json.Unmarshal(bad.Stdout, &e)
	e.Exit = 5
	b, _ := json.Marshal(e)
	d.envelopes["lifecycle status"] = RawResult{Stdout: b, Exit: 5}
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "status: envelope")
}

func TestRun_HealthyButNotRunning(t *testing.T) {
	d := goodDriver()
	d.envelopes["lifecycle status"] = env(true, 0,
		`{"transport":"local","running":false,"healthy":true,"latencyMs":1}`, "lifecycle status")
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "status: healthy implies running")
}

func TestRun_EmptyAdvertisedAxis(t *testing.T) {
	d := goodDriver()
	d.envelopes["meta caps"] = env(true, 0,
		`{"engine":"ydb","contract":"1.0","transports":["local"],`+
			`"axes":{"meta":["caps","version"],"exec":[]},"features":{"remote":false}}`, "meta caps")
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "caps: no empty axes")
}

func TestRun_NotJSON(t *testing.T) {
	d := goodDriver()
	d.envelopes["meta caps"] = RawResult{Stdout: []byte("not json"), Exit: 0}
	rep := Run(context.Background(), d.runner(), "local")
	mustFail(t, rep, "caps: envelope")
}

// --- helpers -----------------------------------------------------------------

func mustFail(t *testing.T, rep Report, checkName string) {
	t.Helper()
	for _, r := range rep.Results {
		if r.Name == checkName {
			if r.Pass {
				t.Fatalf("check %q passed, expected failure", checkName)
			}
			return
		}
	}
	t.Fatalf("check %q not found in report; got: %s", checkName, allNames(rep))
}

func failNames(rep Report) string {
	var n []string
	for _, r := range rep.Results {
		if !r.Pass {
			n = append(n, r.Name+" ("+r.Detail+")")
		}
	}
	return strings.Join(n, "; ")
}

func allNames(rep Report) string {
	var n []string
	for _, r := range rep.Results {
		n = append(n, r.Name)
	}
	return strings.Join(n, ", ")
}
