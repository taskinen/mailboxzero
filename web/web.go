// Package web holds the page template and the static files. They are built
// into the binary, so it runs from any directory.
package web

import (
	"embed"
	"io/fs"
)

//go:embed templates static
var files embed.FS

var (
	// Templates holds the HTML templates (index.html).
	Templates = sub("templates")
	// Static holds the files served under /static/ (app.js, style.css).
	Static = sub("static")
)

func sub(dir string) fs.FS {
	f, err := fs.Sub(files, dir)
	if err != nil {
		panic(err) // dir is a constant that names an embedded directory
	}
	return f
}
