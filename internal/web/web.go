// Package web embeds the built Viewer SPA so `agenttape serve` ships as a single
// self-contained binary — no external dist directory to carry around.
//
// The frontend builds into internal/web/dist (see frontend/vite.config.ts
// build.outDir). Only a placeholder index.html is committed (so `go build` works on
// a fresh checkout before the frontend is built); the real bundle — index.html +
// assets/ — is produced by `npm run build` and overwrites it. The canonical release
// build is therefore: build the frontend, then `go build`.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

// Dist returns the embedded dist directory as a root filesystem (index.html at the
// top), suitable for http serving.
func Dist() fs.FS {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic("web: embedded dist missing: " + err.Error())
	}
	return sub
}
