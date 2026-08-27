package cli

import (
	"strings"

	"github.com/stackmade/driftr/internal/platform"
)

// parseToolVersion splits "node@24.0.0" into ("node", "24.0.0").
//
// When no "@" is present the token is ambiguous: it can be a tool name
// ("pnpm") or a bare node version ("24", "latest"). A token that matches a
// known tool resolves to that tool with an empty version; anything else is
// treated as a node version.
//
// An empty returned version is left for the caller to interpret — it does not
// imply "latest". Callers decide: install treats it as the latest release,
// while uninstall requires an explicit version. Note that a bare tool name and
// an explicit-but-empty spec ("pnpm" vs "pnpm@") both yield an empty version;
// callers that care about the difference must inspect the raw argument.
func parseToolVersion(spec string) (string, string) {
	if tool, ver, ok := strings.Cut(spec, "@"); ok {
		return tool, ver
	}
	if _, ok := platform.LookupTool(spec); ok {
		return spec, ""
	}
	return "node", spec
}
