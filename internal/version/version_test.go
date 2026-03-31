package version

import (
	"testing"
)

func TestParse_Valid(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		patch   int
		str     string
		partial bool
	}{
		{"24", 24, 0, 0, "24.0.0", true},
		{"24.0", 24, 0, 0, "24.0.0", true},
		{"24.0.1", 24, 0, 1, "24.0.1", false},
		{"v24.0.1", 24, 0, 1, "24.0.1", false},
		{"node@24", 24, 0, 0, "24.0.0", true},
		{"node@v24.0.1", 24, 0, 1, "24.0.1", false},
		{"0.12.18", 0, 12, 18, "0.12.18", false},
		{"v22", 22, 0, 0, "22.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch {
				t.Errorf("Parse(%q) = {%d, %d, %d}, want {%d, %d, %d}",
					tt.input, v.Major, v.Minor, v.Patch, tt.major, tt.minor, tt.patch)
			}
			if got := v.String(); got != tt.str {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, got, tt.str)
			}
			if got := v.IsPartial(); got != tt.partial {
				t.Errorf("Parse(%q).IsPartial() = %v, want %v", tt.input, got, tt.partial)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"24.abc",
		"24.0.abc",
		"node@",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", input)
			}
		})
	}
}

func TestMajorMinor(t *testing.T) {
	v, _ := Parse("24.1.3")
	if got := v.MajorMinor(); got != "24.1" {
		t.Errorf("MajorMinor() = %q, want %q", got, "24.1")
	}
}

func TestMatchesMajor(t *testing.T) {
	a, _ := Parse("24.0.0")
	b, _ := Parse("24.1.3")
	c, _ := Parse("22.0.0")

	if !a.MatchesMajor(b) {
		t.Error("expected 24.0.0 to match major with 24.1.3")
	}
	if a.MatchesMajor(c) {
		t.Error("expected 24.0.0 to NOT match major with 22.0.0")
	}
}
