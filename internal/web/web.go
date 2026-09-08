// Package web serves the editor's single-page app, which Vite builds into dist
// and the binary embeds. A .gitkeep keeps the directory embeddable before the
// frontend has been built, so the Go side compiles and tests on its own.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Handler serves the built app. Unknown paths fall back to index.html so the
// app's own routing works on a full page load.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FS(sub)
	server := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		if f, err := files.Open(p); err == nil {
			f.Close()
			if !strings.HasSuffix(p, "/") {
				server.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/"
		server.ServeHTTP(w, r)
	})
}
