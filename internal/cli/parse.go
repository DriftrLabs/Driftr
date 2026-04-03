package cli

import "strings"

// parseToolVersion splits "node@24.0.0" into ("node", "24.0.0").
// If no "@" is present, defaults to tool "node".
func parseToolVersion(spec string) (string, string) {
	if tool, ver, ok := strings.Cut(spec, "@"); ok {
		return tool, ver
	}
	return "node", spec
}
