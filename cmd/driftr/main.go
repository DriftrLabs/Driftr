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
		rb, err := resolver.ResolveBinaryFull(tool, "")
		if err != nil {
			rb, err = cli.HandleShimError(err, tool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "driftr: %s\n", err)
				os.Exit(1)
			}
		}
		if rb.NodePath != "" {
			nodeArgs := append([]string{rb.ToolPath}, os.Args[3:]...)
			if err := process.Exec(rb.NodePath, nodeArgs); err != nil {
				fmt.Fprintf(os.Stderr, "driftr: %s\n", err)
				os.Exit(1)
			}
		}
		if err := process.Exec(rb.ToolPath, os.Args[3:]); err != nil {
			fmt.Fprintf(os.Stderr, "driftr: %s\n", err)
			os.Exit(1)
		}
	}

	cli.Execute()
}
