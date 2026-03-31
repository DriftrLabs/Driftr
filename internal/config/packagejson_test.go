package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writePackageJSON(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test package.json: %v", err)
	}
}

func TestLoadPackageJSON_DriftrNode(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp", "driftr": {"node": "20.11.0"}}`)

	pkg, err := LoadPackageJSON(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg == nil {
		t.Fatal("expected non-nil result")
	}
	if pkg.Driftr.Node != "20.11.0" {
		t.Errorf("got %q, want %q", pkg.Driftr.Node, "20.11.0")
	}
}

func TestLoadPackageJSON_NoDriftr(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp", "version": "1.0.0"}`)

	pkg, err := LoadPackageJSON(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != nil {
		t.Errorf("expected nil for package.json without driftr, got %+v", pkg)
	}
}

func TestLoadPackageJSON_EmptyDriftrNode(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"driftr": {"node": ""}}`)

	pkg, err := LoadPackageJSON(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != nil {
		t.Errorf("expected nil for empty driftr.node, got %+v", pkg)
	}
}

func TestLoadPackageJSON_NoFile(t *testing.T) {
	dir := t.TempDir()

	pkg, err := LoadPackageJSON(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != nil {
		t.Errorf("expected nil when no package.json exists, got %+v", pkg)
	}
}

func TestLoadPackageJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, "{invalid")

	_, err := LoadPackageJSON(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSavePackageJSON_AddsKey(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp", "version": "1.0.0"}`)

	if err := SavePackageJSON(dir, "22.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON after save: %v", err)
	}

	// Verify driftr key was added.
	driftrRaw, ok := raw["driftr"]
	if !ok {
		t.Fatal("driftr key missing from package.json")
	}
	var dc DriftrConfig
	if err := json.Unmarshal(driftrRaw, &dc); err != nil {
		t.Fatalf("failed to unmarshal driftr config: %v", err)
	}
	if dc.Node != "22.14.0" {
		t.Errorf("got %q, want %q", dc.Node, "22.14.0")
	}

	// Verify existing keys are preserved.
	if _, ok := raw["name"]; !ok {
		t.Error("name key was lost")
	}
	if _, ok := raw["version"]; !ok {
		t.Error("version key was lost")
	}
}

func TestSavePackageJSON_UpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp", "driftr": {"node": "20.0.0"}}`)

	if err := SavePackageJSON(dir, "22.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkg, err := LoadPackageJSON(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg.Driftr.Node != "22.14.0" {
		t.Errorf("got %q, want %q", pkg.Driftr.Node, "22.14.0")
	}
}

func TestSavePackageJSON_NoFile(t *testing.T) {
	dir := t.TempDir()

	err := SavePackageJSON(dir, "22.14.0")
	if err == nil {
		t.Fatal("expected error when package.json doesn't exist")
	}
}

func TestRemoveDriftrFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp", "driftr": {"node": "20.0.0"}}`)

	if err := RemoveDriftrFromPackageJSON(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := raw["driftr"]; ok {
		t.Error("driftr key should have been removed")
	}
	if _, ok := raw["name"]; !ok {
		t.Error("name key was lost")
	}
}

func TestRemoveDriftrFromPackageJSON_NoFile(t *testing.T) {
	dir := t.TempDir()

	if err := RemoveDriftrFromPackageJSON(dir); err != nil {
		t.Fatalf("expected nil error when no package.json, got: %v", err)
	}
}

func TestRemoveDriftrFromPackageJSON_NoDriftrKey(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, `{"name": "myapp"}`)

	if err := RemoveDriftrFromPackageJSON(dir); err != nil {
		t.Fatalf("expected nil error when no driftr key, got: %v", err)
	}
}
