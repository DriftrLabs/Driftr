package installer

import "testing"

func TestInstallYarn_RejectsLTS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := InstallYarn("lts", false); err == nil {
		t.Fatal("InstallYarn(lts) expected error, got nil")
	}
}
