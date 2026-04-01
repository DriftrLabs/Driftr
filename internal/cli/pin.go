package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/resolver"
)

// pinFormat represents the config storage format.
type pinFormat int

const (
	formatNone        pinFormat = iota
	formatTOML                  // .driftr.toml
	formatPackageJSON           // package.json driftr key
)

func newPinCmd() *cobra.Command {
	var migrate bool

	cmd := &cobra.Command{
		Use:   "pin <tool@version>",
		Short: "Pin a tool version to the current project",
		Long: `Pin a tool version to the current project.

The version is stored in .driftr.toml or package.json (driftr key).
On first use, you'll be asked which format to use. Subsequent pins
reuse the detected format automatically.

Examples:
  driftr pin node@22.14.0
  driftr pin node@22.14.0 --migrate   # switch storage format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool, versionSpec := parseToolVersion(args[0])

			versionStr, _, err := resolver.RequireToolInstalled(tool, versionSpec)
			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine current directory: %w", err)
			}

			current := detectPinFormat(cwd)

			if migrate {
				return migratePin(cwd, tool, versionStr, current)
			}

			format := current
			if format == formatNone {
				var err error
				format, err = promptPinFormat(cmd)
				if err != nil {
					return err
				}
			}

			return savePin(cwd, tool, versionStr, format)
		},
	}

	cmd.Flags().BoolVar(&migrate, "migrate", false, "migrate config to the other storage format")

	return cmd
}

// detectPinFormat checks which config format exists in the directory.
// .driftr.toml takes priority if both exist.
func detectPinFormat(dir string) pinFormat {
	tomlPath := filepath.Join(dir, config.ProjectConfigFile)
	if _, err := os.Stat(tomlPath); err == nil {
		return formatTOML
	}

	pkg, _ := config.LoadPackageJSON(dir)
	if pkg != nil {
		return formatPackageJSON
	}

	return formatNone
}

// promptPinFormat asks the user to choose a storage format.
func promptPinFormat(cmd *cobra.Command) (pinFormat, error) {
	// Non-interactive: default to .driftr.toml.
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return formatTOML, nil
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "No existing project config found. How should the version be stored?")
	fmt.Fprintln(cmd.ErrOrStderr(), "  1) .driftr.toml (recommended)")
	fmt.Fprintln(cmd.ErrOrStderr(), "  2) package.json (driftr key)")
	fmt.Fprint(cmd.ErrOrStderr(), "Choose [1/2]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return formatTOML, err
	}

	switch strings.TrimSpace(input) {
	case "2":
		return formatPackageJSON, nil
	default:
		return formatTOML, nil
	}
}

// savePin writes the version in the given format.
func savePin(dir, tool, versionStr string, format pinFormat) error {
	switch format {
	case formatPackageJSON:
		if err := config.SavePackageJSONTool(dir, tool, versionStr); err != nil {
			return err
		}
		fmt.Printf("Pinned %s %s in %s/package.json\n", tool, versionStr, dir)
	default:
		cfg, err := config.LoadProject(dir)
		if err != nil {
			return err
		}
		if cfg == nil {
			cfg = &config.ProjectConfig{}
		}
		cfg.Tools.SetTool(tool, versionStr)
		if err := config.SaveProject(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Pinned %s %s in %s/%s\n", tool, versionStr, dir, config.ProjectConfigFile)
	}
	return nil
}

// migratePin switches from the current format to the other one.
func migratePin(dir, tool, versionStr string, current pinFormat) error {
	switch current {
	case formatTOML:
		// Migrate TOML → package.json.
		if err := config.SavePackageJSONTool(dir, tool, versionStr); err != nil {
			return err
		}
		os.Remove(filepath.Join(dir, config.ProjectConfigFile))
		fmt.Printf("Migrated %s %s from %s to package.json\n", tool, versionStr, config.ProjectConfigFile)

	case formatPackageJSON:
		// Migrate package.json → TOML.
		cfg := &config.ProjectConfig{}
		cfg.Tools.SetTool(tool, versionStr)
		if err := config.SaveProject(dir, cfg); err != nil {
			return err
		}
		if err := config.RemoveDriftrFromPackageJSON(dir); err != nil {
			return err
		}
		fmt.Printf("Migrated %s %s from package.json to %s\n", tool, versionStr, config.ProjectConfigFile)

	case formatNone:
		return fmt.Errorf("no existing config to migrate. Run `driftr pin %s@<version>` first", tool)
	}
	return nil
}
