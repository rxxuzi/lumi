package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

// GetStaticFilesHandler returns an http.Handler that serves static files
func GetStaticFilesHandler() http.Handler {
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(fsys))
}
