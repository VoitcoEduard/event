package main

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/user/go-expo-sample/backend/utils"
)

func newSPAHandler(distDir string) http.HandlerFunc {
	files := os.DirFS(distDir)
	server := http.FileServerFS(files)
	return func(w http.ResponseWriter, r *http.Request) {
		utils.DebugLog("Static request: %s %s", r.Method, r.URL.Path)
		enableCors(w, r)
		// Resolve the existence check inside the same filesystem used to serve
		// assets. Never concatenate an untrusted URL onto a filesystem path.
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		if _, err := fs.Stat(files, name); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		server.ServeHTTP(w, r)
	}
}
