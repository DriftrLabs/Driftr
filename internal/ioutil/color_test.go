package ioutil

import (
	"os"
	"strings"
	"testing"
)

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
	// Tests always run with stdout as a pipe (non-TTY).
	os.Unsetenv("NO_COLOR")

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
