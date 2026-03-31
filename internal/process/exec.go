package process

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Exec replaces the current process with the given binary.
// On Unix, this uses syscall.Exec for zero-overhead process replacement.
// Falls back to exec.Command on systems where Exec is not available.
func Exec(binary string, args []string) error {
	// Prepend binary as argv[0].
	argv := append([]string{binary}, args...)

	// Use syscall.Exec to replace the current process.
	// This preserves stdin/stdout/stderr and exit codes.
	return syscall.Exec(binary, argv, os.Environ())
}

// Run executes a binary as a child process and returns the exit code.
func Run(binary string, args []string) (int, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("failed to execute %s: %w", binary, err)
	}

	return 0, nil
}
