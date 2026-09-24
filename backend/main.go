package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/utils"
)

type Response struct {
	Message string `json:"message"`
}

func main() {
	utils.DebugLog("Starting server initialization...")
	if len(os.Args) > 1 {
		if err := runAuthCommand(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}
	config, err := auth.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to MongoDB
	db.ConnectDB()
	db.InitGridFS()
	store := auth.Store{DB: db.Database()}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Fatalf("authentication indexes: %v", err)
	}
	authService, err := auth.NewService(ctx, config, store)
	if err != nil {
		log.Fatal(err)
	}
	handlers.SetAuthService(authService)

	// Initialize QuizHub
	quizHub := hub.NewQuizHub()
	quizHub.SetAuthService(authService)
	go quizHub.Run()

	// Inject QuizHub into handlers
	handlers.SetQuizHub(quizHub)

	mux := http.NewServeMux()
	authService.Register(mux)
	registerRoutes(mux, quizHub)

	// Serve the Vue SPA from the ./dist directory (built by `npm run build`).
	// Any path not matched by an API handler above falls through here.
	// Unknown paths get index.html so Vue Router handles client-side routing.
	distDir := "./dist"
	var spaHandler http.HandlerFunc
	if _, err := os.Stat(distDir); err == nil {
		fs := http.FileServer(http.Dir(distDir))
		spaHandler = func(w http.ResponseWriter, r *http.Request) {
			utils.DebugLog("Static request: %s %s", r.Method, r.URL.Path)
			enableCors(w, r)
			// If the requested file exists in dist/, serve it directly.
			// Otherwise fall back to index.html for Vue Router history mode.
			if _, err := os.Stat(distDir + r.URL.Path); os.IsNotExist(err) {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		}
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

	port := ":8080"
	utils.DebugLog("Server listening on %s", port)
	if err := http.ListenAndServe(port, appHandler(mux, spaHandler)); err != nil {
		utils.DebugLog("Server failed: %v", err) // Use utils.DebugLog here
	}
}

func appHandler(api, spa http.Handler) http.Handler {
	prefixes := []string{"/auth", "/events", "/sessions", "/screensavers", "/tracks", "/templates", "/media", "/ws"}
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
