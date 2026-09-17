// Package assets provides the read-only framework and module files bundled
// into the executable at build time.
package assets

import (
	"embed"
	"io/fs"
)

// Directory patterns include nested files. Keep the README placeholders so
// both patterns remain valid before real payloads are added.
//
//go:embed frameworks modules
var bundled embed.FS

// Files returns the embedded filesystem. Paths use forward slashes and start
// with frameworks/ or modules/, regardless of the host operating system.
func Files() fs.FS { return bundled }
