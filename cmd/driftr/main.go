package main

import (
	"fmt"
	"os"

	"github.com/DriftrLabs/driftr/internal/cli"
	"github.com/DriftrLabs/driftr/internal/process"
	"github.com/DriftrLabs/driftr/internal/resolver"
)

func main() {
	// Fast path: bypass Cobra entirely for shim invocations.
	// Shim scripts call "driftr shim <tool> [args...]".
	if len(os.Args) >= 3 && os.Args[1] == "shim" {
		tool := os.Args[2]
		binPath, err := resolver.ResolveBinary(tool, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "driftr: %s\n", err)
			os.Exit(1)
		}
		if err := process.Exec(binPath, os.Args[3:]); err != nil {
			fmt.Fprintf(os.Stderr, "driftr: %s\n", err)
			os.Exit(1)
		}
	}

	cli.Execute()
}
