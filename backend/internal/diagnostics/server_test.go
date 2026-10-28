package diagnostics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDisabledByDefault(t *testing.T) {
	t.Setenv("PERF_ADDR", "")
	s, err := StartFromEnv()
	if s != nil || err != nil {
		t.Fatalf("disabled diagnostics: %v, %v", s, err)
	}
}

func TestRejectPublicListener(t *testing.T) {
	for _, addr := range []string{":6060", "0.0.0.0:6060", "[::]:6060", "192.168.1.2:6060", "localhost:6060", "invalid"} {
		if s, err := start(addr, 0, 0); err == nil {
			s.Close()
			t.Fatalf("accepted %s", addr)
		}
	}
}

func TestSamplingConfiguration(t *testing.T) {
	t.Setenv("PERF_ADDR", "127.0.0.1:0")
	for _, key := range []string{"PERF_MUTEX_FRACTION", "PERF_BLOCK_RATE"} {
		t.Setenv(key, "-1")
		if _, err := StartFromEnv(); err == nil {
			t.Fatalf("accepted invalid %s", key)
		}
		t.Setenv(key, "0")
	}
}

func TestDiagnosticEndpoints(t *testing.T) {
	mux := newMux()
	for _, path := range []string{"/debug/perf", "/debug/pprof/heap", "/debug/pprof/mutex", "/debug/pprof/block", "/debug/pprof/goroutine"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, w.Code)
		}
		if path == "/debug/perf" {
			var result map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result["goVersion"] == nil {
				t.Fatalf("invalid runtime metrics: %s", w.Body)
			}
		}
	}
}

func TestLoopbackListenerLifecycle(t *testing.T) {
	s, err := start("127.0.0.1:0", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + s.Addr + "/debug/perf")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("status %d", response.StatusCode)
	}
}
