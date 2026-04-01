package cli

import "strings"

// parseToolVersion splits "node@24.0.0" into ("node", "24.0.0").
// If no "@" is present, defaults to tool "node".
func parseToolVersion(spec string) (string, string) {
	if i := strings.Index(spec, "@"); i >= 0 {
		return spec[:i], spec[i+1:]
	}
	return "node", spec
}
