package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProjectConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ProjectConfigFile), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test project config: %v", err)
	}
}

func TestLoadProject_NoFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject() unexpected error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when no file exists, got %+v", cfg)
	}
}

func TestLoadProject_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, `[tools]
node = "22.14.0"
`)

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Tools.Node != "22.14.0" {
		t.Errorf("Tools.Node = %q, want %q", cfg.Tools.Node, "22.14.0")
	}
}

func TestLoadProject_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, `this is not valid toml {{{`)

	_, err := LoadProject(dir)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func TestSaveAndLoadProject(t *testing.T) {
	dir := t.TempDir()

	cfg := &ProjectConfig{}
	cfg.Tools.Node = "24.0.1"

	if err := SaveProject(dir, cfg); err != nil {
		t.Fatalf("SaveProject() error: %v", err)
	}

	loaded, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil config after save")
	}
	if loaded.Tools.Node != "24.0.1" {
		t.Errorf("Tools.Node = %q, want %q", loaded.Tools.Node, "24.0.1")
	}
}

func TestLoadProject_EmptyNode(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, `[tools]
`)

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject() error: %v", err)
	}
	// File exists but node is empty — still returns the config struct.
	if cfg == nil {
		t.Fatal("expected non-nil config for existing file with empty node")
	}
	if cfg.Tools.Node != "" {
		t.Errorf("Tools.Node = %q, want empty", cfg.Tools.Node)
	}
}
