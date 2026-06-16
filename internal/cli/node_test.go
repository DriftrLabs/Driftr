package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DriftrLabs/driftr/internal/nodeenv"
)

// stubRunner implements nodeenv.Runner for command-level tests.
type stubRunner struct {
	calls   []string
	outputs map[string]string
	missing map[string]bool
}

func newStubRunner() *stubRunner {
	return &stubRunner{outputs: map[string]string{}, missing: map[string]bool{}}
}

func (s *stubRunner) Run(name string, args ...string) (string, error) {
	key := strings.TrimSpace(name + " " + strings.Join(args, " "))
	s.calls = append(s.calls, key)
	return s.outputs[key], nil
}

func (s *stubRunner) LookPath(name string) (string, error) {
	if s.missing[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func (s *stubRunner) called(cmd string) bool {
	for _, c := range s.calls {
		if c == cmd {
			return true
		}
	}
	return false
}

func TestRunNodeOptimize_SetsUnconfiguredKeys(t *testing.T) {
	s := newStubRunner()
	// both keys read back empty → both should be set
	var buf bytes.Buffer
	if err := runNodeOptimize(nodeenv.NewPnpm(s), "/store/pnpm", false, &buf); err != nil {
		t.Fatal(err)
	}
	if !s.called("pnpm config set store-dir /store/pnpm") {
		t.Errorf("expected store-dir to be set; calls=%v", s.calls)
	}
	if !s.called("pnpm config set enable-global-virtual-store true") {
		t.Errorf("expected virtual-store to be set; calls=%v", s.calls)
	}
}

func TestRunNodeOptimize_Idempotent(t *testing.T) {
	s := newStubRunner()
	s.outputs["pnpm config get store-dir"] = "/store/pnpm"
	s.outputs["pnpm config get enable-global-virtual-store"] = "true"

	var buf bytes.Buffer
	if err := runNodeOptimize(nodeenv.NewPnpm(s), "/store/pnpm", false, &buf); err != nil {
		t.Fatal(err)
	}
	for _, c := range s.calls {
		if strings.HasPrefix(c, "pnpm config set") {
			t.Errorf("expected no config set when already configured, got %q", c)
		}
	}
	if !strings.Contains(buf.String(), "already") {
		t.Errorf("expected 'already' in output, got: %s", buf.String())
	}
}

func TestRunNodeOptimize_PnpmMissing(t *testing.T) {
	s := newStubRunner()
	s.missing["pnpm"] = true
	err := runNodeOptimize(nodeenv.NewPnpm(s), "/store/pnpm", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "corepack enable") {
		t.Errorf("expected actionable pnpm-missing error, got: %v", err)
	}
}

func TestRunNodeClean_DryRunMutatesNothing(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "f"), make([]byte, 10), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newStubRunner()
	var buf bytes.Buffer
	if err := runNodeClean(nodeenv.NewPnpm(s), dir, false, &buf); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); err != nil {
		t.Errorf("dry-run must not remove node_modules: %v", err)
	}
	if len(s.calls) != 0 {
		t.Errorf("dry-run must not run any command, got %v", s.calls)
	}
	if !strings.Contains(buf.String(), "Dry run") {
		t.Errorf("expected dry-run notice, got: %s", buf.String())
	}
}

func TestRunNodeClean_ExecuteRemovesAndPrunes(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "f"), make([]byte, 10), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newStubRunner()
	if err := runNodeClean(nodeenv.NewPnpm(s), dir, true, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); !os.IsNotExist(err) {
		t.Errorf("node_modules should be removed, stat err=%v", err)
	}
	if !s.called("pnpm install") {
		t.Errorf("expected pnpm install, got %v", s.calls)
	}
	if !s.called("pnpm store prune") {
		t.Errorf("expected pnpm store prune, got %v", s.calls)
	}
}

func TestGatherNodeDoctor_RecommendsOptimizeWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newStubRunner()
	s.outputs["node --version"] = "v22.14.0"
	s.outputs["pnpm --version"] = "9.1.0"
	s.outputs["pnpm store path"] = "/store/pnpm"
	s.outputs["pnpm config get enable-global-virtual-store"] = "undefined"

	info := gatherNodeDoctor(s, dir)
	if info.packageManager != nodeenv.PnpmManager {
		t.Errorf("packageManager = %q, want pnpm", info.packageManager)
	}
	if info.nodeVersion != "22.14.0" {
		t.Errorf("nodeVersion = %q, want 22.14.0", info.nodeVersion)
	}
	if info.globalVirtualStore {
		t.Error("globalVirtualStore should be false for 'undefined'")
	}

	var buf bytes.Buffer
	renderNodeDoctor(&buf, info)
	if !strings.Contains(buf.String(), "driftr node optimize") {
		t.Errorf("expected optimize recommendation, got: %s", buf.String())
	}
}

func TestRunNodeReport_ShowsSizes(t *testing.T) {
	dir := t.TempDir()
	store := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "node_modules.placeholder"), make([]byte, 5), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "x"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "blob"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newStubRunner()
	s.outputs["pnpm store path"] = store

	var buf bytes.Buffer
	if err := runNodeReport(nodeenv.NewPnpm(s), dir, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Project node_modules:") || !strings.Contains(out, "Shared pnpm store:") {
		t.Errorf("report missing size lines: %s", out)
	}
}

func TestNodeCmd_HelpRegistered(t *testing.T) {
	if err := runCmd(t, "node", "--help"); err != nil {
		t.Errorf("node --help failed: %v", err)
	}
}
