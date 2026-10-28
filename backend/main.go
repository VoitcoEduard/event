package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/filestore"
	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/internal/diagnostics"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/utils"
)

type Response struct {
	Message string `json:"message"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	utils.DebugLog("Starting server initialization...")
	if len(os.Args) > 1 {
		var err error
		if os.Args[1] == "migrate" {
			err = runMigrationCommand(os.Args[1:])
		} else {
			err = runAuthCommand(os.Args[1:])
		}
		if err != nil {
			return err
		}
		return nil
	}
	config, err := auth.ConfigFromEnv()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Migrate(ctx); err != nil {
		return err
	}
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		mediaRoot = "/app/data/media"
	}
	files, err := filestore.Open(mediaRoot, db)
	if err != nil {
		return err
	}
	store := auth.Store{DB: db}
	authService, err := auth.NewService(ctx, config, store)
	if err != nil {
		return err
	}
	// Build the HTTP and websocket dependencies once, after PostgreSQL is ready.
	events := services.NewEventService(db)
	teams := database.NewTeamRepository(db)
	quizResults := &services.QuizResultService{
		Results:     database.NewQuizResultRepository(db),
		Teams:       teams,
		TeamResults: database.NewTeamResultRepository(db),
	}
	quizHub := hub.NewQuizHub(hub.Dependencies{
		DB:          db,
		AuthService: authService,
		Events:      events,
		Teams:       teams,
		QuizResults: quizResults,
	})
	handler := handlers.New(handlers.Dependencies{
		DB: db, Files: files,
		Auth:         authService,
		QuizHub:      quizHub,
		Events:       events,
		Modules:      services.NewModuleService(db),
		Participants: services.NewParticipantService(db),
		Sessions:     services.NewEventSessionService(db),
		Templates:    services.NewTemplateService(db),
		Teams:        teams,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := db.Ready(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := files.Ready(); err != nil {
			http.Error(w, "media unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	authService.Register(mux)
	registerRoutes(mux, handler)

	// Serve the Vue SPA from the ./dist directory (built by `npm run build`).
	// Any path not matched by an API handler above falls through here.
	// Unknown paths get index.html so Vue Router handles client-side routing.
	distDir := "./dist"
	var spaHandler http.HandlerFunc
	if _, err := os.Stat(distDir); err == nil {
		spaHandler = newSPAHandler(distDir)
	} else {
		// No dist/ folder — local dev mode, return a simple JSON health check.
		spaHandler = func(w http.ResponseWriter, r *http.Request) {
			utils.DebugLog("Received request: %s %s", r.Method, r.URL.Path)
			enableCors(w, r)
			if r.Method == "OPTIONS" {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			response := Response{Message: "Hello from Go Backend!"}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				utils.DebugLog("Error encoding response: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}
	}

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	workersDone := make(chan struct{})
	go func() {
		defer close(workersDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(workerCtx, 30*time.Second)
			if err := db.CleanupAuth(ctx, time.Now().UTC()); err != nil && workerCtx.Err() == nil {
				slog.Error("auth cleanup failed", "error", err)
			}
			if err := files.Reconcile(ctx, time.Now()); err != nil && workerCtx.Err() == nil {
				slog.Error("media cleanup failed", "error", err)
			}
			cancel()
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	defer func() { stopWorkers(); <-workersDone }()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           appHandler(mux, spaHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}
	profiler, err := diagnostics.StartFromEnv(func() any {
		stats := db.Pool.Stat()
		return map[string]any{"acquireCount": stats.AcquireCount(), "acquireDurationNs": stats.AcquireDuration().Nanoseconds(), "emptyAcquireCount": stats.EmptyAcquireCount(), "emptyAcquireWaitTimeNs": stats.EmptyAcquireWaitTime().Nanoseconds(), "acquiredConns": stats.AcquiredConns(), "totalConns": stats.TotalConns(), "maxConns": stats.MaxConns()}
	})
	if err != nil {
		return err
	}
	defer profiler.Close()
	if profiler != nil {
		slog.Info("private profiling listener enabled", "address", profiler.Addr)
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	slog.Info("server listening", "address", server.Addr)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var serveErr error
	select {
	case err := <-serverErrors:
		serveErr = err
	case <-signalCtx.Done():
	}
	stop()
	slog.Info("shutting down")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if serveErr == nil {
		serveErr = server.Shutdown(shutdownCtx)
	}
	quizHub.Shutdown()
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return nil
}

func appHandler(api, spa http.Handler) http.Handler {
	prefixes := []string{"/health", "/auth", "/events", "/sessions", "/screensavers", "/tracks", "/templates", "/media", "/ws"}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range prefixes {
			if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
				api.ServeHTTP(w, r)
				return
			}
		}
		spa.ServeHTTP(w, r)
	})
}

func enableCors(w http.ResponseWriter, r *http.Request) {
	// The application uses same-origin requests, including in local development.
}
