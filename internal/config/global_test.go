package config

import (
	"testing"

	"github.com/DriftrLabs/driftr/internal/platform"
)

func TestLoadGlobal_NoDirExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal() unexpected error: %v", err)
	}
	if cfg.Default.Node != "" {
		t.Errorf("expected empty default node, got %q", cfg.Default.Node)
	}
}

func TestSaveAndLoadGlobal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := platform.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error: %v", err)
	}

	cfg := &GlobalConfig{Default: DefaultConfig{Node: "22.14.0"}}
	if err := SaveGlobal(cfg); err != nil {
		t.Fatalf("SaveGlobal() error: %v", err)
	}

	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal() error: %v", err)
	}
	if loaded.Default.Node != "22.14.0" {
		t.Errorf("LoadGlobal().Default.Node = %q, want %q", loaded.Default.Node, "22.14.0")
	}
}

func TestSaveGlobal_OverwritesExisting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := platform.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error: %v", err)
	}

	cfg1 := &GlobalConfig{Default: DefaultConfig{Node: "20.0.0"}}
	if err := SaveGlobal(cfg1); err != nil {
		t.Fatalf("SaveGlobal() error: %v", err)
	}

	cfg2 := &GlobalConfig{Default: DefaultConfig{Node: "22.14.0"}}
	if err := SaveGlobal(cfg2); err != nil {
		t.Fatalf("SaveGlobal() error: %v", err)
	}

	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal() error: %v", err)
	}
	if loaded.Default.Node != "22.14.0" {
		t.Errorf("LoadGlobal().Default.Node = %q, want %q", loaded.Default.Node, "22.14.0")
	}
}
