// Package diagnostics provides an opt-in, loopback-only profiling listener.
// It deliberately uses a separate mux: no diagnostic handlers reach the public API.
package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Server struct {
	http *http.Server
	Addr string
}

// StartFromEnv returns nil when PERF_ADDR is unset. Profiling sampling also
// stays disabled unless explicitly requested for a diagnostic run.
func StartFromEnv() (*Server, error) {
	addr := os.Getenv("PERF_ADDR")
	if addr == "" {
		return nil, nil
	}
	mutex, err := sampleRate("PERF_MUTEX_FRACTION")
	if err != nil {
		return nil, err
	}
	block, err := sampleRate("PERF_BLOCK_RATE")
	if err != nil {
		return nil, err
	}
	return start(addr, mutex, block)
}

func sampleRate(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a nonnegative integer", key)
	}
	return n, nil
}

func start(addr string, mutex, block int) (*Server, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("PERF_ADDR: %w", err)
	}
	// Require a literal address to avoid DNS resolution changing the interface.
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("PERF_ADDR must bind a loopback IP, e.g. 127.0.0.1:6060")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("profiling listener: %w", err)
	}
	mux := newMux()
	s := &Server{http: &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}, Addr: listener.Addr().String()}
	runtime.SetMutexProfileFraction(mutex)
	runtime.SetBlockProfileRate(block)
	go func() { _ = s.http.Serve(listener) }()
	return s, nil
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/perf", func(w http.ResponseWriter, _ *http.Request) {
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"goVersion": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
			"numCPU": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0),
			"goroutines": runtime.NumGoroutine(), "heapAllocBytes": memory.HeapAlloc,
			"heapSysBytes": memory.HeapSys, "totalAllocBytes": memory.TotalAlloc,
			"gcCycles": memory.NumGC, "gcPauseTotalNs": memory.PauseTotalNs,
		})
	})
	return mux
}

func (s *Server) Close() {
	if s == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.http.Shutdown(ctx)
	runtime.SetMutexProfileFraction(0)
	runtime.SetBlockProfileRate(0)
}
