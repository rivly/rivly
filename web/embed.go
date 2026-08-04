package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var build embed.FS

func Dashboard() (fs.FS, bool) {
	dist, err := fs.Sub(build, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return nil, false
	}
	return dist, true
}
