package version

import (
	"regexp"
	"testing"
)

func TestVersionIsValidSemver(t *testing.T) {
	t.Parallel()
	if Version == "" {
		t.Fatal("Version is empty")
	}
	matched, err := regexp.MatchString(`^\d+\.\d+\.\d+$`, Version)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("Version = %q, want semver format (X.Y.Z)", Version)
	}
}

func TestVersionHasNoWhitespace(t *testing.T) {
	t.Parallel()
	for _, c := range Version {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			t.Fatalf("Version contains whitespace: %q", Version)
		}
	}
}
