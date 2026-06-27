package api

import (
	"bytes"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

func newSPAHandler(directory string) http.Handler {
	root := os.DirFS(directory)
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestPath := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			requestPath = "index.html"
		}
		if !fs.ValidPath(requestPath) {
			http.NotFound(writer, request)
			return
		}
		info, err := fs.Stat(root, requestPath)
		if err != nil || info.IsDir() {
			requestPath = "index.html"
		}
		if requestPath == "index.html" {
			indexInfo, statErr := fs.Stat(root, "index.html")
			if statErr != nil {
				http.NotFound(writer, request)
				return
			}
			content, readErr := fs.ReadFile(root, "index.html")
			if readErr != nil {
				http.Error(writer, "WaveSight frontend unavailable", http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Cache-Control", "no-cache")
			http.ServeContent(writer, request, "index.html", indexInfo.ModTime(), bytes.NewReader(content))
			return
		}
		clone := request.Clone(request.Context())
		clone.URL.Path = "/" + requestPath
		files.ServeHTTP(writer, clone)
	})
}
