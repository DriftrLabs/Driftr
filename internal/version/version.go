package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// Parse parses a version string like "24", "24.0", or "24.0.1".
// Supports optional "v" prefix and "node@" prefix.
func Parse(s string) (Version, error) {
	raw := s
	s = strings.TrimPrefix(s, "node@")
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)

	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	parts := strings.SplitN(s, ".", 3)

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}

	v := Version{Major: major, Raw: raw}

	if len(parts) >= 2 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
		}
		v.Minor = minor
	}

	if len(parts) == 3 {
		patch, err := strconv.Atoi(parts[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
		}
		v.Patch = patch
	}

	return v, nil
}

// IsPartial returns true if the version was specified without all three components.
func (v Version) IsPartial() bool {
	raw := strings.TrimPrefix(v.Raw, "node@")
	raw = strings.TrimPrefix(raw, "v")
	parts := strings.Split(raw, ".")
	return len(parts) < 3
}

// String returns the full semver string.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// MajorMinor returns "MAJOR.MINOR" format.
func (v Version) MajorMinor() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// MatchesMajor returns true if the other version has the same major version.
func (v Version) MatchesMajor(other Version) bool {
	return v.Major == other.Major
}
