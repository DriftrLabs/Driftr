package pathsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setHome creates a temp dir, sets HOME to it, and returns the path.
func setHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("ZDOTDIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	return dir
}

func TestDetectShell(t *testing.T) {
	tests := []struct {
		sh   string
		want Shell
	}{
		{"/bin/zsh", ShellZsh},
		{"/usr/bin/bash", ShellBash},
		{"/opt/homebrew/bin/fish", ShellFish},
		{"/bin/dash", ShellUnknown},
		{"", ShellUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.sh, func(t *testing.T) {
			t.Setenv("SHELL", tc.sh)
			if got := DetectShell(); got != tc.want {
				t.Errorf("DetectShell(%q) = %v, want %v", tc.sh, got, tc.want)
			}
		})
	}
}

func TestTargetProfile_Zsh(t *testing.T) {
	home := setHome(t)
	got, err := TargetProfile(ShellZsh)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".zshenv")
	if got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestTargetProfile_ZshHonorsZDOTDIR(t *testing.T) {
	setHome(t)
	custom := t.TempDir()
	t.Setenv("ZDOTDIR", custom)
	got, err := TargetProfile(ShellZsh)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(custom, ".zshenv")
	if got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestTargetProfile_Bash(t *testing.T) {
	home := setHome(t)
	got, err := TargetProfile(ShellBash)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".bash_profile")
	if got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestTargetProfile_Fish(t *testing.T) {
	home := setHome(t)
	got, err := TargetProfile(ShellFish)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "fish", "conf.d", "driftr.fish")
	if got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestTargetProfile_FishHonorsXDG(t *testing.T) {
	setHome(t)
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := TargetProfile(ShellFish)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "fish", "conf.d", "driftr.fish")
	if got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestDetect_EmptyEnv(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")

	binDir := filepath.Join(home, ".driftr", "bin")
	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Shell != ShellZsh {
		t.Errorf("shell = %v, want zsh", r.Shell)
	}
	if r.InTarget {
		t.Errorf("InTarget = true, want false (.zshenv does not exist)")
	}
	if len(r.StaleFiles) != 0 {
		t.Errorf("StaleFiles = %v, want empty", r.StaleFiles)
	}
	if r.InProcessPATH {
		t.Errorf("InProcessPATH = true, want false")
	}
	if !r.NeedsFix() {
		t.Errorf("NeedsFix = false, want true")
	}
}

func TestDetect_StaleInZshrc(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")
	zshrc := filepath.Join(home, ".zshrc")
	writeFile(t, zshrc, "# legacy\nexport PATH=\""+binDir+":$PATH\"\n")

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if r.InTarget {
		t.Errorf("InTarget = true, want false")
	}
	if len(r.StaleFiles) != 1 || r.StaleFiles[0] != zshrc {
		t.Errorf("StaleFiles = %v, want [%s]", r.StaleFiles, zshrc)
	}
	if !r.NeedsFix() {
		t.Errorf("NeedsFix = false, want true")
	}
}

func TestDetect_AlreadyInTarget(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")
	zshenv := filepath.Join(home, ".zshenv")
	writeFile(t, zshenv, "export PATH=\""+binDir+":$PATH\"\n")

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InTarget {
		t.Errorf("InTarget = false, want true")
	}
	if r.NeedsFix() {
		t.Errorf("NeedsFix = true, want false")
	}
}

func TestApply_CreatesZshenv(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")
	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}

	wrote, file, err := Apply(r)
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("Apply did not write")
	}
	want := filepath.Join(home, ".zshenv")
	if file != want {
		t.Errorf("wrote to %q, want %q", file, want)
	}
	content := readFile(t, file)
	if !strings.Contains(content, "export PATH=\""+binDir+":$PATH\"") {
		t.Errorf("content missing export line:\n%s", content)
	}
	if !strings.Contains(content, "# Driftr") {
		t.Errorf("content missing # Driftr marker:\n%s", content)
	}
}

func TestApply_NoOpWhenInTarget(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")
	zshenv := filepath.Join(home, ".zshenv")
	original := "export PATH=\"" + binDir + ":$PATH\"\n"
	writeFile(t, zshenv, original)

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	wrote, file, err := Apply(r)
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Errorf("Apply wrote when target already had entry")
	}
	if file != "" {
		t.Errorf("file = %q, want empty", file)
	}
	if got := readFile(t, zshenv); got != original {
		t.Errorf("file was modified:\n%s", got)
	}
}

func TestApply_AppendsWithLeadingNewline(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")
	zshenv := filepath.Join(home, ".zshenv")
	writeFile(t, zshenv, "export FOO=bar\n")

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Apply(r); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, zshenv)
	wantPrefix := "export FOO=bar\n\n# Driftr\n"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("content does not preserve prior line with newline separator:\n%s", got)
	}
}

func TestApply_Fish(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/opt/homebrew/bin/fish")

	binDir := filepath.Join(home, ".driftr", "bin")
	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	wrote, file, err := Apply(r)
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("Apply did not write")
	}
	want := filepath.Join(home, ".config", "fish", "conf.d", "driftr.fish")
	if file != want {
		t.Errorf("wrote to %q, want %q", file, want)
	}
	content := readFile(t, file)
	if !strings.Contains(content, "set -gx PATH "+binDir+" $PATH") {
		t.Errorf("content missing fish syntax:\n%s", content)
	}
}

func TestDetect_HomeRelativeForms(t *testing.T) {
	home := setHome(t)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(home, ".driftr", "bin")

	// Each rc file uses a different form of the home-relative path.
	writeFile(t, filepath.Join(home, ".zshenv"), "export PATH=\"$HOME/.driftr/bin:$PATH\"\n")
	writeFile(t, filepath.Join(home, ".zshrc"), "export PATH=\"~/.driftr/bin:$PATH\"\n")
	writeFile(t, filepath.Join(home, ".zprofile"), "export PATH=\"${HOME}/.driftr/bin:$PATH\"\n")

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InTarget {
		t.Errorf("InTarget = false, want true ($HOME form in .zshenv)")
	}
	// Both .zshrc (~) and .zprofile (${HOME}) should register as stale.
	if len(r.StaleFiles) != 2 {
		t.Errorf("StaleFiles = %v, want 2 entries (~ and ${HOME} forms)", r.StaleFiles)
	}
}

func TestDetect_InProcessPATH(t *testing.T) {
	home := setHome(t)
	binDir := filepath.Join(home, ".driftr", "bin")
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", binDir+":/usr/bin")

	r, err := Detect(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InProcessPATH {
		t.Errorf("InProcessPATH = false, want true")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
