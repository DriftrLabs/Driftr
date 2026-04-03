package installer

import (
	"crypto/sha512"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/DriftrLabs/driftr/internal/version"
)

func TestParseSRI_Valid(t *testing.T) {
	// A known sha512 hash in SRI format.
	rawHash := make([]byte, 64) // 512 bits = 64 bytes
	rawHash[0] = 0xAB
	rawHash[63] = 0xCD
	b64 := base64.StdEncoding.EncodeToString(rawHash)
	sri := "sha512-" + b64

	algo, hash, err := parseSRI(sri)
	if err != nil {
		t.Fatalf("parseSRI() error: %v", err)
	}
	if algo != "sha512" {
		t.Errorf("algo = %q, want %q", algo, "sha512")
	}
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
	if hash[0] != 0xAB || hash[63] != 0xCD {
		t.Error("hash content mismatch")
	}
}

func TestParseSRI_Invalid(t *testing.T) {
	tests := []string{
		"",
		"sha512",
		"nohyphen",
		"sha512-!!!invalid-base64!!!",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, _, err := parseSRI(input)
			if err == nil {
				t.Errorf("parseSRI(%q) expected error, got nil", input)
			}
		})
	}
}

func TestVerifyIntegrity_Valid(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello npm registry")
	path := filepath.Join(dir, "test.tgz")

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := sha512.Sum512(content)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(h[:])

	if err := VerifyIntegrity(path, integrity); err != nil {
		t.Fatalf("VerifyIntegrity() error: %v", err)
	}
}

func TestVerifyIntegrity_Mismatch(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello npm registry")
	path := filepath.Join(dir, "test.tgz")

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Use a wrong hash.
	wrongHash := make([]byte, 64)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(wrongHash)

	err := VerifyIntegrity(path, integrity)
	if err == nil {
		t.Fatal("expected integrity mismatch error, got nil")
	}
}

func TestVerifyIntegrity_UnsupportedAlgo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tgz")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := VerifyIntegrity(path, "sha256-AAAA")
	if err == nil {
		t.Fatal("expected unsupported algorithm error, got nil")
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int // positive if a > b, negative if a < b, 0 if equal
	}{
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			a, err := version.Parse(tt.a)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.a, err)
			}
			b, err := version.Parse(tt.b)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.b, err)
			}
			if got := versionCompare(a, b); got != tt.want {
				t.Errorf("versionCompare(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
