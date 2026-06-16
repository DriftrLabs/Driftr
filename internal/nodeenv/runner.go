package nodeenv

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Runner abstracts external command execution so pnpm/corepack interactions can
// be driven by a fake in tests. The production implementation is execRunner.
type Runner interface {
	// Run executes name with args, returning trimmed stdout. On failure the
	// error includes captured stderr for actionable messages.
	Run(name string, args ...string) (string, error)
	// LookPath reports the resolved path of name, or an error if not on PATH.
	LookPath(name string) (string, error)
}

// execRunner is the os/exec-backed Runner used in production.
type execRunner struct{}

// NewExecRunner returns a Runner that shells out to real binaries.
func NewExecRunner() Runner { return execRunner{} }

func (execRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (execRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
