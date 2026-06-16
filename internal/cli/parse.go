package cli

import (
	"strings"

	"github.com/DriftrLabs/driftr/internal/platform"
)

// parseToolVersion splits "node@24.0.0" into ("node", "24.0.0").
//
// When no "@" is present the token is ambiguous: it can be a tool name
// ("pnpm") or a bare node version ("24", "latest"). A token that matches a
// known tool resolves to that tool with an empty version (callers default the
// empty version to "latest"); anything else is treated as a node version.
func parseToolVersion(spec string) (string, string) {
	if tool, ver, ok := strings.Cut(spec, "@"); ok {
		return tool, ver
	}
	if _, ok := platform.LookupTool(spec); ok {
		return spec, ""
	}
	return "node", spec
}
