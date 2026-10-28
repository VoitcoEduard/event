package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiagnosticsDoNotReachPublicHandler(t *testing.T) {
	public := appHandler(http.NewServeMux(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	for _, path := range []string{"/debug/perf", "/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/profile"} {
		w := httptest.NewRecorder()
		public.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusTeapot {
			t.Fatalf("public route %s exposed a diagnostic handler: %d", path, w.Code)
		}
	}
}
