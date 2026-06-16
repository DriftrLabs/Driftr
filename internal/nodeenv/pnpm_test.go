package nodeenv

import (
	"errors"
	"strings"
	"testing"
)

// fakeRunner records invocations and returns scripted output keyed by the joined
// command line. Missing keys return empty output and no error.
type fakeRunner struct {
	calls    []string
	outputs  map[string]string
	errs     map[string]error
	missing  map[string]bool // names that LookPath should fail for
	lookErrs error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}, errs: map[string]error{}, missing: map[string]bool{}}
}

func (f *fakeRunner) Run(name string, args ...string) (string, error) {
	key := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.calls = append(f.calls, key)
	return f.outputs[key], f.errs[key]
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.missing[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func (f *fakeRunner) called(cmd string) bool {
	for _, c := range f.calls {
		if c == cmd {
			return true
		}
	}
	return false
}

func TestPnpm_Installed(t *testing.T) {
	f := newFakeRunner()
	if !NewPnpm(f).Installed() {
		t.Error("expected Installed=true when pnpm on PATH")
	}
	f.missing["pnpm"] = true
	if NewPnpm(f).Installed() {
		t.Error("expected Installed=false when pnpm missing")
	}
}

func TestPnpm_ConfigSetCommand(t *testing.T) {
	f := newFakeRunner()
	if err := NewPnpm(f).ConfigSet(StoreDirKey, "/store"); err != nil {
		t.Fatal(err)
	}
	want := "pnpm config set store-dir /store"
	if !f.called(want) {
		t.Errorf("expected call %q, got %v", want, f.calls)
	}
}

func TestPnpm_ConfigGetReturnsOutput(t *testing.T) {
	f := newFakeRunner()
	f.outputs["pnpm config get enable-global-virtual-store"] = "true"
	got, err := NewPnpm(f).ConfigGet(GlobalVirtualStoreKey)
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("ConfigGet = %q, want %q", got, "true")
	}
}

func TestPnpm_CorepackEnableCommand(t *testing.T) {
	f := newFakeRunner()
	if err := NewPnpm(f).CorepackEnable(); err != nil {
		t.Fatal(err)
	}
	if !f.called("corepack enable") {
		t.Errorf("expected 'corepack enable', got %v", f.calls)
	}
}

func TestPnpm_StoreCommands(t *testing.T) {
	f := newFakeRunner()
	f.outputs["pnpm store path"] = "/store/pnpm"
	p := NewPnpm(f)
	if got, _ := p.StorePath(); got != "/store/pnpm" {
		t.Errorf("StorePath = %q", got)
	}
	if _, err := p.StorePrune(); err != nil {
		t.Fatal(err)
	}
	if !f.called("pnpm store prune") {
		t.Errorf("expected 'pnpm store prune', got %v", f.calls)
	}
}

func TestPnpm_RunErrorPropagates(t *testing.T) {
	f := newFakeRunner()
	f.errs["pnpm config set store-dir /store"] = errors.New("boom")
	if err := NewPnpm(f).ConfigSet(StoreDirKey, "/store"); err == nil {
		t.Error("expected error to propagate")
	}
}
