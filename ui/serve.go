package ui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// 将 build 目录及其所有子目录和文件打包到二进制文件中，并将 files 作为其接口供外界访问
//
//go:embed build
var buildFiles embed.FS
var files, _ = fs.Sub(buildFiles, "build")

// Register registers the ui on the root path.
func Register(r *mux.Router) {
	r.Handle("/", serveFile("index.html", "text/html"))
	r.Handle("/index.html", serveFile("index.html", "text/html"))
	r.Handle("/assets/{resource}", http.FileServer(http.FS(files)))

	r.Handle("/favicon.ico", serveFile("favicon.ico", "image/x-icon"))
	r.Handle("/logo.svg", serveFile("logo.svg", "image/svg+xml"))
	r.Handle("/apple-touch-icon.png", serveFile("apple-touch-icon.png", "image/png"))
	r.Handle("/og-banner.png", serveFile("og-banner.png", "image/png"))
}

// serveFile serves a file from the embedded filesystem.
func serveFile(name, contentType string) http.HandlerFunc {
	file, err := files.Open(name)
	if err != nil {
		log.Panic().Err(err).Msgf("could not find %s", file)
	}
	defer func(file fs.File) {
		_ = file.Close()
	}(file)
	content, err := io.ReadAll(file)
	if err != nil {
		log.Panic().Err(err).Msgf("could not read %s", file)
	}

	return func(writer http.ResponseWriter, reg *http.Request) {
		writer.Header().Set("Content-Type", contentType)
		_, _ = writer.Write(content)
	}
}
