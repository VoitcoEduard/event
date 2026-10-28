package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAAssetsFallbackAndTraversal(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	if err := os.Mkdir(dist, 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"index.html": "SPA entry", "asset.js": "static asset"} {
		if err := os.WriteFile(filepath.Join(dist, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	handler := newSPAHandler(dist)
	request := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}
	for path, want := range map[string]string{"/": "SPA entry", "/asset.js": "static asset", "/client/route": "SPA entry"} {
		w := request(path)
		if w.Code != http.StatusOK || w.Body.String() != want {
			t.Fatalf("%s: %d %q, want 200 %q", path, w.Code, w.Body.String(), want)
		}
	}
	for _, path := range []string{"/../secret", "/%2e%2e/secret", "/nested/../../secret"} {
		before := request(path)
		if err := os.WriteFile(filepath.Join(root, "secret"), []byte("outside secret"), 0600); err != nil {
			t.Fatal(err)
		}
		after := request(path)
		if before.Code != after.Code || before.Body.String() != after.Body.String() {
			t.Fatalf("%s disclosed existence of an outside file: before=%d/%q after=%d/%q", path, before.Code, before.Body.String(), after.Code, after.Body.String())
		}
		if err := os.Remove(filepath.Join(root, "secret")); err != nil {
			t.Fatal(err)
		}
	}
}
