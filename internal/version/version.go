package version

import (
	_ "embed"
	"strings"
)

//go:embed version.txt
var Version string

func init() {
	Version = strings.TrimSpace(Version)
}
