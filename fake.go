package mdriver

import "context"

// FakeTransport is the injected Transport for driver unit tests (no engine). It
// records every call and returns canned results, so a command's behavior —
// envelope shape, engineError mapping, verb sequencing — is asserted without a
// real engine. Real transports appear only in the gated integration tier
// (plan §1, "No hidden engine in unit tests").
//
// Set the *Fn fields to script behavior; an unset verb returns a zero result.
// Calls records an ordered trace for sequencing/argument assertions.
type FakeTransport struct {
	HealthFn     func(ctx context.Context) (Health, error)
	LoadFn       func(ctx context.Context, req LoadRequest) (LoadResult, error)
	ExecFn       func(ctx context.Context, req ExecRequest) (ExecResult, error)
	ReadGlobalFn func(ctx context.Context, req GlobalRef) (GlobalNode, error)
	SetGlobalFn  func(ctx context.Context, ref, value string) error

	Calls []FakeCall
}

// FakeCall is one recorded interaction. Req holds the request struct (or, for
// SetGlobal, a [2]string{ref, value}).
type FakeCall struct {
	Verb string
	Req  any
}

var _ Transport = (*FakeTransport)(nil)

func (f *FakeTransport) Health(ctx context.Context) (Health, error) {
	f.Calls = append(f.Calls, FakeCall{Verb: "Health"})
	if f.HealthFn != nil {
		return f.HealthFn(ctx)
	}
	return Health{}, nil
}

func (f *FakeTransport) Load(ctx context.Context, req LoadRequest) (LoadResult, error) {
	f.Calls = append(f.Calls, FakeCall{Verb: "Load", Req: req})
	if f.LoadFn != nil {
		return f.LoadFn(ctx, req)
	}
	return LoadResult{}, nil
}

func (f *FakeTransport) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	f.Calls = append(f.Calls, FakeCall{Verb: "Exec", Req: req})
	if f.ExecFn != nil {
		return f.ExecFn(ctx, req)
	}
	return ExecResult{}, nil
}

func (f *FakeTransport) ReadGlobal(ctx context.Context, req GlobalRef) (GlobalNode, error) {
	f.Calls = append(f.Calls, FakeCall{Verb: "ReadGlobal", Req: req})
	if f.ReadGlobalFn != nil {
		return f.ReadGlobalFn(ctx, req)
	}
	return GlobalNode{}, nil
}

func (f *FakeTransport) SetGlobal(ctx context.Context, ref, value string) error {
	f.Calls = append(f.Calls, FakeCall{Verb: "SetGlobal", Req: [2]string{ref, value}})
	if f.SetGlobalFn != nil {
		return f.SetGlobalFn(ctx, ref, value)
	}
	return nil
}
