// Package webui serves the embedded browser UI for master-agent.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

// Handler returns an http.Handler that serves embedded static assets.
func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("webui: embed static: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
