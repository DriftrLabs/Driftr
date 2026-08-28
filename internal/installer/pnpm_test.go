package installer

import "testing"

func TestInstallPnpm_RejectsLTS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := InstallPnpm("lts", false); err == nil {
		t.Fatal("InstallPnpm(lts) expected error, got nil")
	}
}
