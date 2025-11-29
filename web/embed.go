package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns the embedded frontend filesystem
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
