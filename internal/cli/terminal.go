package cli

import (
	"os"
	"syscall"
	"unsafe"
)

// isStdinTerminal returns true if stdin is connected to a real terminal.
// Returns false for /dev/null, pipes, and redirected input.
func isStdinTerminal() bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, os.Stdin.Fd(), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}
