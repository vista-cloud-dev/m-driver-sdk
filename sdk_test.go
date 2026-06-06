package mdriver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCaps_OmitsUnwiredAxes(t *testing.T) {
	// The honest-incremental model: a driver advertising only its wired axes
	// leaves the rest nil, and they must not appear in the JSON.
	c := Caps{
		Engine:     "ydb",
		Contract:   ContractVersion,
		Transports: []string{TransportLocal, TransportDocker},
		Axes:       Axes{Meta: []string{"caps", "version", "schema"}},
		Features:   Features{Prune: true, EphemeralPrefix: true},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"meta":["caps","version","schema"]`) {
		t.Errorf("meta axis missing: %s", s)
	}
	for _, unwired := range []string{`"lifecycle"`, `"sync"`, `"exec"`, `"data"`, `"cover"`, `"admin"`} {
		if strings.Contains(s, unwired) {
			t.Errorf("unwired axis %s leaked into caps JSON: %s", unwired, s)
		}
	}
	// Features never omit: a false flag is meaningful (remote:false is a fact).
	if !strings.Contains(s, `"remote":false`) || !strings.Contains(s, `"snapshot":false`) {
		t.Errorf("features must render false flags: %s", s)
	}
}

func TestM1Shapes_JSONTags(t *testing.T) {
	// These shapes are the cross-driver contract m-cli reads; pin their JSON.
	checks := []struct {
		v    any
		want string
	}{
		{Check{Name: "binary", OK: true}, `{"name":"binary","ok":true}`},
		{DoctorResult{Transport: "local", OK: true, Checks: []Check{}}, `{"transport":"local","ok":true,"checks":[]}`},
		{StateResult{State: "started"}, `{"state":"started"}`},
		{Status{Transport: "docker", Running: true, Healthy: true, LatencyMs: 3}, `{"transport":"docker","running":true,"healthy":true,"latencyMs":3}`},
	}
	for _, c := range checks {
		b, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("marshal %T: %v", c.v, err)
		}
		if string(b) != c.want {
			t.Errorf("%T JSON = %s, want %s", c.v, b, c.want)
		}
	}
}

func TestCoverResult_JSONTag(t *testing.T) {
	// The `cover trace` payload (driver-contract.md §5.5) — the shape m-cli reads
	// identically from m-ydb (view "TRACE":1:"^ycov") and m-iris
	// (%Monitor.System.LineByLine) to aggregate LCOV and apply --min-percent.
	// Pin its JSON so the two drivers cannot drift.
	cr := CoverResult{
		LCOV:         "TN:\nSF:MATH.m\nDA:1,1\nLF:2\nLH:1\nend_of_record\n",
		CoveredLines: 1,
		TotalLines:   2,
		LinePercent:  50,
	}
	b, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"lcov":"TN:\nSF:MATH.m\nDA:1,1\nLF:2\nLH:1\nend_of_record\n","coveredLines":1,"totalLines":2,"linePercent":50}`
	if string(b) != want {
		t.Errorf("CoverResult JSON = %s, want %s", b, want)
	}
	// Counts always render — a zero is meaningful (0% covered is a fact, not
	// absence), the same convention as Status.LatencyMs and Features flags.
	z, _ := json.Marshal(CoverResult{})
	if string(z) != `{"lcov":"","coveredLines":0,"totalLines":0,"linePercent":0}` {
		t.Errorf("zero CoverResult must render all fields: %s", z)
	}
}

func TestAxes_WiredOrderAndSkipsEmpty(t *testing.T) {
	a := Axes{Meta: []string{"caps"}, Lifecycle: []string{"up", "down"}}
	w := a.Wired()
	if len(w) != 2 || w[0].Name != "lifecycle" || w[1].Name != "meta" {
		t.Fatalf("Wired() = %+v, want [lifecycle, meta] in contract order", w)
	}
}

func TestFakeTransport_RecordsAndScripts(t *testing.T) {
	var f Transport = &FakeTransport{
		ExecFn: func(_ context.Context, req ExecRequest) (ExecResult, error) {
			return ExecResult{Stdout: "out:" + req.Command}, nil
		},
		HealthFn: func(context.Context) (Health, error) {
			return Health{Running: true, Healthy: true, Version: "r2.02"}, nil
		},
	}
	res, err := f.Exec(context.Background(), ExecRequest{Command: "write 1"})
	if err != nil || res.Stdout != "out:write 1" {
		t.Fatalf("exec = %q, %v", res.Stdout, err)
	}
	if err := f.SetGlobal(context.Background(), "^x", "1"); err != nil {
		t.Fatalf("setglobal: %v", err)
	}
	h, _ := f.Health(context.Background())
	if h.Version != "r2.02" {
		t.Errorf("health version = %q", h.Version)
	}

	fake := f.(*FakeTransport)
	wantVerbs := []string{"Exec", "SetGlobal", "Health"}
	if len(fake.Calls) != len(wantVerbs) {
		t.Fatalf("recorded %d calls, want %d: %+v", len(fake.Calls), len(wantVerbs), fake.Calls)
	}
	for i, v := range wantVerbs {
		if fake.Calls[i].Verb != v {
			t.Errorf("call %d = %q, want %q", i, fake.Calls[i].Verb, v)
		}
	}
	if got := fake.Calls[1].Req.([2]string); got != [2]string{"^x", "1"} {
		t.Errorf("SetGlobal recorded %v", got)
	}
}
