package resolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
)

// setupFakeInstall creates a fake tool installation with a binary file.
func setupFakeInstall(t *testing.T, home, tool, version string) {
	t.Helper()
	dir, _ := platform.ToolVersionDir(tool, version)
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	// Determine binary name from tool map.
	entry, ok := platform.LookupTool(tool)
	binName := tool
	if ok {
		binName = entry.Binary
	}
	if err := os.WriteFile(filepath.Join(binDir, binName), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}
}

func TestRequireToolInstalled_ExactVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")

	ver, binPath, err := RequireToolInstalled("node", "22.14.0")
	if err != nil {
		t.Fatalf("RequireToolInstalled() error: %v", err)
	}
	if ver != "22.14.0" {
		t.Errorf("version = %q, want %q", ver, "22.14.0")
	}
	if binPath == "" {
		t.Error("binPath is empty")
	}
}

func TestRequireToolInstalled_NotInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, _, err := RequireToolInstalled("node", "99.0.0")
	if err == nil {
		t.Fatal("expected error for uninstalled version, got nil")
	}
}

func TestRequireToolInstalled_PartialVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")
	setupFakeInstall(t, home, "node", "22.13.0")

	ver, _, err := RequireToolInstalled("node", "22")
	if err != nil {
		t.Fatalf("RequireToolInstalled() error: %v", err)
	}
	if ver != "22.14.0" {
		t.Errorf("version = %q, want %q (latest 22.x)", ver, "22.14.0")
	}
}

func TestRequireToolInstalled_Latest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "20.11.0")
	setupFakeInstall(t, home, "node", "22.14.0")

	ver, _, err := RequireToolInstalled("node", "latest")
	if err != nil {
		t.Fatalf("RequireToolInstalled() error: %v", err)
	}
	if ver != "22.14.0" {
		t.Errorf("version = %q, want %q (latest)", ver, "22.14.0")
	}
}

func TestRequireToolInstalled_PartialNoMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "20.11.0")

	_, _, err := RequireToolInstalled("node", "22")
	if err == nil {
		t.Fatal("expected error for no matching version, got nil")
	}
}

func TestResolveFromProject_DriftrToml(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")

	// Create project with .driftr.toml.
	projectDir := t.TempDir()
	cfg := &config.ProjectConfig{}
	cfg.Tools.SetTool("node", "22.14.0")
	if err := config.SaveProject(projectDir, cfg); err != nil {
		t.Fatalf("SaveProject() error: %v", err)
	}

	// Chdir to project.
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "22.14.0" {
		t.Errorf("version = %q, want %q", res.Version, "22.14.0")
	}
	if res.Source != SourceProject {
		t.Errorf("source = %v, want %v", res.Source, SourceProject)
	}
}

func TestResolveFromProject_PackageJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")

	// Create project with package.json.
	projectDir := t.TempDir()
	pkgJSON := `{"name": "test", "driftr": {"node": "22.14.0"}}`
	os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(pkgJSON), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "22.14.0" {
		t.Errorf("version = %q, want %q", res.Version, "22.14.0")
	}
	if res.Source != SourcePackageJSON {
		t.Errorf("source = %v, want %v", res.Source, SourcePackageJSON)
	}
}

func TestResolveFromProject_TomlTakesPriority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "20.0.0")
	setupFakeInstall(t, home, "node", "22.14.0")

	// Create project with both .driftr.toml and package.json.
	projectDir := t.TempDir()
	cfg := &config.ProjectConfig{}
	cfg.Tools.SetTool("node", "20.0.0")
	config.SaveProject(projectDir, cfg)
	pkgJSON := `{"name": "test", "driftr": {"node": "22.14.0"}}`
	os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(pkgJSON), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	// .driftr.toml should win.
	if res.Version != "20.0.0" {
		t.Errorf("version = %q, want %q (.driftr.toml should take priority)", res.Version, "20.0.0")
	}
	if res.Source != SourceProject {
		t.Errorf("source = %v, want %v", res.Source, SourceProject)
	}
}

func TestResolveFromGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := platform.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	setupFakeInstall(t, home, "node", "22.14.0")

	// Set global default.
	globalCfg, _ := config.LoadGlobal()
	globalCfg.Default.SetTool("node", "22.14.0")
	config.SaveGlobal(globalCfg)

	// Chdir to a directory with no project config.
	emptyDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(emptyDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "22.14.0" {
		t.Errorf("version = %q, want %q", res.Version, "22.14.0")
	}
	if res.Source != SourceGlobal {
		t.Errorf("source = %v, want %v", res.Source, SourceGlobal)
	}
}

func TestResolveExplicit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "24.0.0")

	res, err := ResolveTool("node", "24.0.0", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "24.0.0" {
		t.Errorf("version = %q, want %q", res.Version, "24.0.0")
	}
	if res.Source != SourceExplicit {
		t.Errorf("source = %v, want %v", res.Source, SourceExplicit)
	}
}

func TestResolveNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	emptyDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(emptyDir)

	_, err := ResolveTool("node", "", false)
	if err == nil {
		t.Fatal("expected error when no config exists, got nil")
	}
}

func TestResolveTool_PnpmIndependent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "pnpm", "9.15.0")

	// Create project with pnpm pinned.
	projectDir := t.TempDir()
	cfg := &config.ProjectConfig{}
	cfg.Tools.SetTool("pnpm", "9.15.0")
	config.SaveProject(projectDir, cfg)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("pnpm", "", false)
	if err != nil {
		t.Fatalf("ResolveTool(pnpm) error: %v", err)
	}
	if res.Tool != "pnpm" {
		t.Errorf("tool = %q, want %q", res.Tool, "pnpm")
	}
	if res.Version != "9.15.0" {
		t.Errorf("version = %q, want %q", res.Version, "9.15.0")
	}
}

func TestResolveBinary_NpmResolveViaNode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := platform.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	setupFakeInstall(t, home, "node", "22.14.0")
	// npm binary is inside the node installation.
	npmBinDir := filepath.Join(home, ".driftr", "tools", "node", "22.14.0", "bin")
	os.WriteFile(filepath.Join(npmBinDir, "npm"), []byte("#!/bin/sh\n"), 0o755)

	globalCfg, _ := config.LoadGlobal()
	globalCfg.Default.SetTool("node", "22.14.0")
	config.SaveGlobal(globalCfg)

	emptyDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(emptyDir)

	binPath, err := ResolveBinary("npm", "")
	if err != nil {
		t.Fatalf("ResolveBinary(npm) error: %v", err)
	}
	if binPath == "" {
		t.Error("binPath is empty")
	}
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{SourceExplicit, "explicit override"},
		{SourceProject, "project config"},
		{SourcePackageJSON, "package.json (driftr)"},
		{SourceNvmrc, ".nvmrc"},
		{SourceNodeVersion, ".node-version"},
		{SourceGlobal, "global default"},
		{Source(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.source.String(); got != tt.want {
			t.Errorf("Source(%d).String() = %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestNotInstalledError(t *testing.T) {
	e := &NotInstalledError{Tool: "node", Version: "24.1.0", Context: "pinned in /home/user/project"}
	msg := e.Error()
	if !strings.Contains(msg, "node 24.1.0") {
		t.Errorf("error should contain tool and version, got: %s", msg)
	}
	if !strings.Contains(msg, "pinned in /home/user/project") {
		t.Errorf("error should contain context, got: %s", msg)
	}
	if !strings.Contains(msg, "driftr install node@24.1.0") {
		t.Errorf("error should contain install hint, got: %s", msg)
	}

	// Without context.
	e2 := &NotInstalledError{Tool: "pnpm", Version: "9.0.0"}
	msg2 := e2.Error()
	if strings.Contains(msg2, "(") {
		t.Errorf("error without context should not contain parens, got: %s", msg2)
	}
}

func TestNotInstalledError_ErrorsAs(t *testing.T) {
	orig := &NotInstalledError{Tool: "node", Version: "22.0.0"}
	wrapped := fmt.Errorf("resolution failed: %w", orig)

	var target *NotInstalledError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should unwrap NotInstalledError from wrapped error")
	}
	if target.Tool != "node" || target.Version != "22.0.0" {
		t.Errorf("unwrapped error has wrong fields: %+v", target)
	}
}

func TestRequireToolBinaryExists_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := requireToolBinaryExists("node", "99.99.99", "global default")
	if err == nil {
		t.Fatal("expected error for non-existent version")
	}

	var notInstalled *NotInstalledError
	if !errors.As(err, &notInstalled) {
		t.Fatalf("expected NotInstalledError, got %T: %v", err, err)
	}
	if notInstalled.Tool != "node" || notInstalled.Version != "99.99.99" || notInstalled.Context != "global default" {
		t.Errorf("unexpected fields: %+v", notInstalled)
	}
}

func TestResolveFromProject_Nvmrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")

	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, ".nvmrc"), []byte("22.14.0\n"), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "22.14.0" {
		t.Errorf("version = %q, want %q", res.Version, "22.14.0")
	}
	if res.Source != SourceNvmrc {
		t.Errorf("source = %v, want %v", res.Source, SourceNvmrc)
	}
}

func TestResolveFromProject_NodeVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "22.14.0")

	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, ".node-version"), []byte("22.14.0\n"), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "22.14.0" {
		t.Errorf("version = %q, want %q", res.Version, "22.14.0")
	}
	if res.Source != SourceNodeVersion {
		t.Errorf("source = %v, want %v", res.Source, SourceNodeVersion)
	}
}

func TestResolveFromProject_NvmrcPriority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	setupFakeInstall(t, home, "node", "20.0.0")
	setupFakeInstall(t, home, "node", "22.14.0")

	// .nvmrc should lose to .driftr.toml
	projectDir := t.TempDir()
	cfg := &config.ProjectConfig{}
	cfg.Tools.SetTool("node", "20.0.0")
	config.SaveProject(projectDir, cfg)
	os.WriteFile(filepath.Join(projectDir, ".nvmrc"), []byte("22.14.0\n"), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	res, err := ResolveTool("node", "", false)
	if err != nil {
		t.Fatalf("ResolveTool() error: %v", err)
	}
	if res.Version != "20.0.0" {
		t.Errorf("version = %q, want %q (.driftr.toml should beat .nvmrc)", res.Version, "20.0.0")
	}
	if res.Source != SourceProject {
		t.Errorf("source = %v, want %v", res.Source, SourceProject)
	}
}

func TestResolveFromProject_NvmrcIgnoredForNonNode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create .nvmrc but resolve pnpm — should not match.
	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, ".nvmrc"), []byte("22.14.0\n"), 0o644)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(projectDir)

	// pnpm resolution should fall through to global (and fail with no global set).
	_, err := ResolveTool("pnpm", "", false)
	if err == nil {
		t.Fatal("expected error for pnpm with only .nvmrc, got nil")
	}
}
