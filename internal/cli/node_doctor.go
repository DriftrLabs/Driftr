package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/nodeenv"
)

// nodeDoctorInfo is the gathered state rendered by `driftr node doctor`.
type nodeDoctorInfo struct {
	projectDir         string
	packageManager     nodeenv.PackageManager
	nodeVersion        string // "" when node not found
	corepackAvailable  bool
	pnpmInstalled      bool
	pnpmVersion        string
	nodeModulesSize    int64
	storePath          string // "" when pnpm missing/unknown
	globalVirtualStore bool
}

func newNodeDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Analyze the Node.js dependency environment",
		Long:  "Inspect the current project's package manager and pnpm shared-store configuration, and recommend optimizations.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			r := nodeenv.NewExecRunner()
			info := gatherNodeDoctor(r, dir)
			renderNodeDoctor(cmd.OutOrStdout(), info)
			return nil
		},
	}
}

// gatherNodeDoctor collects environment state. It never fails on missing
// tooling — absence is reported, not treated as an error.
func gatherNodeDoctor(r nodeenv.Runner, projectDir string) nodeDoctorInfo {
	p := nodeenv.NewPnpm(r)
	info := nodeDoctorInfo{
		projectDir:        projectDir,
		packageManager:    nodeenv.DetectPackageManager(projectDir),
		corepackAvailable: p.CorepackAvailable(),
		pnpmInstalled:     p.Installed(),
	}

	if v, err := r.Run("node", "--version"); err == nil {
		info.nodeVersion = strings.TrimPrefix(v, "v")
	}

	if size, err := nodeenv.DirSize(filepath.Join(projectDir, "node_modules")); err == nil {
		info.nodeModulesSize = size
	}

	if info.pnpmInstalled {
		if v, err := p.Version(); err == nil {
			info.pnpmVersion = v
		}
		if path, err := p.StorePath(); err == nil {
			info.storePath = path
		}
		if val, err := p.ConfigGet(nodeenv.GlobalVirtualStoreKey); err == nil {
			info.globalVirtualStore = val == "true"
		}
	}

	return info
}

func renderNodeDoctor(w io.Writer, info nodeDoctorInfo) {
	fmt.Fprintln(w, "Driftr Node Doctor")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Project:         %s\n", info.projectDir)
	fmt.Fprintf(w, "Package manager: %s\n", info.packageManager)
	fmt.Fprintf(w, "Node.js:         %s\n", orNotFound(info.nodeVersion))
	fmt.Fprintf(w, "Corepack:        %s\n", availability(info.corepackAvailable))

	if info.pnpmInstalled {
		fmt.Fprintf(w, "pnpm:            %s\n", orNotFound(info.pnpmVersion))
	} else {
		fmt.Fprintf(w, "pnpm:            not found\n")
	}

	if info.nodeModulesSize > 0 {
		fmt.Fprintf(w, "node_modules:    %s\n", formatSize(info.nodeModulesSize))
	} else {
		fmt.Fprintf(w, "node_modules:    none\n")
	}

	fmt.Fprintf(w, "pnpm store:      %s\n", orUnknown(info.storePath))
	fmt.Fprintf(w, "global virtual store: %s\n", enabledDisabled(info.globalVirtualStore))

	if !info.pnpmInstalled {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Recommendation:")
		fmt.Fprintln(w, "pnpm is not installed. Enable corepack to get it, then optimize storage.")
		fmt.Fprintln(w, "Run: corepack enable && driftr node optimize")
		return
	}

	if !info.globalVirtualStore {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Recommendation:")
		fmt.Fprintln(w, "Enable pnpm shared storage to reduce duplicated dependency data.")
		fmt.Fprintln(w, "Run: driftr node optimize")
	}
}

func orNotFound(s string) string {
	if s == "" {
		return "not found"
	}
	return s
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func availability(ok bool) string {
	if ok {
		return "available"
	}
	return "not found"
}

func enabledDisabled(on bool) string {
	if on {
		return "enabled"
	}
	return "disabled"
}
