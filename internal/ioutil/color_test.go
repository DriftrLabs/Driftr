package ioutil

import (
	"os"
	"strings"
	"sync"
	"testing"
)

// resetColorCache resets the memoized TTY state so tests get a clean slate.
func resetColorCache() {
	ttyOnce = sync.Once{}
	isTTY = false
}

func TestColorize_AppliesANSI(t *testing.T) {
	got := colorize("hello", "32")
	if got != "\033[32mhello\033[0m" {
		t.Errorf("colorize() = %q, want ANSI-wrapped string", got)
	}
}

func TestColorDisabled_NO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if Green("x") != "x" {
		t.Error("Green() should return plain text when NO_COLOR is set")
	}
	if Bold("x") != "x" {
		t.Error("Bold() should return plain text when NO_COLOR is set")
	}
	if Dim("x") != "x" {
		t.Error("Dim() should return plain text when NO_COLOR is set")
	}
}

func TestColorDisabled_NonTTY(t *testing.T) {
	// Reset cached TTY state so this test controls when it fires.
	resetColorCache()
	t.Cleanup(resetColorCache)

	// Guarantee non-TTY stdout via a pipe, regardless of test runner environment.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = old
		r.Close()
		w.Close()
	})

	// Ensure NO_COLOR is unset so we test TTY detection, not the NO_COLOR path.
	prev, prevOK := os.LookupEnv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	t.Cleanup(func() {
		if prevOK {
			os.Setenv("NO_COLOR", prev)
		}
	})

	if strings.Contains(Green("x"), "\033") {
		t.Error("Green() should not emit ANSI codes when stdout is not a TTY")
	}
	if strings.Contains(Bold("x"), "\033") {
		t.Error("Bold() should not emit ANSI codes when stdout is not a TTY")
	}
	if strings.Contains(Dim("x"), "\033") {
		t.Error("Dim() should not emit ANSI codes when stdout is not a TTY")
	}
}
