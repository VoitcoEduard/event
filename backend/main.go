package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

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

	// Connect to MongoDB
	db.ConnectDB()
	db.InitGridFS()

	// Initialize QuizHub
	quizHub := hub.NewQuizHub()
	go quizHub.Run()

	// Inject QuizHub into handlers
	handlers.SetQuizHub(quizHub)

	// WebSocket Endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(quizHub, w, r)
	})

	// Auth endpoint (public)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "POST" {
			handlers.Login(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Register Handlers
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		switch r.Method {
		case "GET":
			handlers.GetEvents(w, r)
		case "POST":
			handlers.VerifyAuth(handlers.CreateEvent)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/events/connect", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "POST" {
			handlers.ConnectToEvent(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/templates", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		switch r.Method {
		case "GET":
			handlers.GetTemplates(w, r)
		case "POST":
			handlers.VerifyAuth(handlers.SaveTemplate)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/templates/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "PUT" {
			handlers.VerifyAuth(handlers.UpdateTemplate)(w, r)
			return
		}
		if r.Method == "DELETE" {
			handlers.VerifyAuth(handlers.DeleteTemplate)(w, r)
			return
		}
	})

	http.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		// Check for modules related paths
		if strings.Contains(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/response") {
			if r.Method == "POST" {
				handlers.SubmitModuleResponse(w, r)
				return
			}
		}

		if strings.Contains(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/status") {
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.UpdateModuleStatus)(w, r)
				return
			}
		}

		if strings.Contains(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/publish") {
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.PublishModule)(w, r)
				return
			}
		}

		// Quiz routes
		if strings.Contains(r.URL.Path, "/modules/") && strings.Contains(r.URL.Path, "/quiz/") {
			if strings.HasSuffix(r.URL.Path, "/questions") && r.Method == "PUT" {
				handlers.VerifyAuth(handlers.UpdateQuizQuestions)(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/state") && r.Method == "GET" {
				handlers.GetQuizState(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/start") && r.Method == "POST" {
				handlers.VerifyAuth(handlers.StartQuiz)(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/next") && r.Method == "POST" {
				handlers.VerifyAuth(handlers.NextQuestion)(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/results") && r.Method == "GET" {
				handlers.GetQuizResults(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/reset") && r.Method == "POST" {
				handlers.VerifyAuth(handlers.ResetQuiz)(w, r)
				return
			}
		}

		// Get Module Responses
		if strings.Contains(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/responses") && r.Method == "GET" {
			handlers.VerifyAuth(handlers.GetModuleResponses)(w, r)
			return
		}

		// Delete Module
		if strings.Contains(r.URL.Path, "/modules/") && r.Method == "DELETE" {
			handlers.VerifyAuth(handlers.DeleteModule)(w, r)
			return
		}

		// Update Module (including background music settings)
		if strings.Contains(r.URL.Path, "/modules/") && r.Method == "PUT" {
			handlers.VerifyAuth(handlers.UpdateModule)(w, r)
			return
		}

		// Get Module (for BigScreen to load music settings)
		if strings.Contains(r.URL.Path, "/modules/") && r.Method == "GET" {
			handlers.GetModule(w, r)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/modules") {
			if r.Method == "POST" {
				handlers.VerifyAuth(handlers.AddModule)(w, r)
				return
			}
		}

		// Update Module Response
		if strings.Contains(r.URL.Path, "/modules/") && strings.Contains(r.URL.Path, "/responses/") && r.Method == "PUT" {
			handlers.VerifyAuth(handlers.UpdateModuleResponse)(w, r)
			return
		}

		// Update Quiz Score
		if strings.Contains(r.URL.Path, "/modules/") && strings.Contains(r.URL.Path, "/quiz-scores/") && r.Method == "PUT" {
			handlers.VerifyAuth(handlers.UpdateQuizScore)(w, r)
			return
		}

		// Check for /status suffix (event status)
		if len(r.URL.Path) > 7 && r.URL.Path[len(r.URL.Path)-7:] == "/status" {
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.UpdateEventStatus)(w, r)
				return
			}
		}

		// Screensaver config routes
		if strings.HasSuffix(r.URL.Path, "/screensaver") {
			if r.Method == "GET" {
				handlers.GetScreensaverConfig(w, r)
				return
			}
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.UpdateScreensaverConfig)(w, r)
				return
			}
		}

		// ScreenState routes
		if strings.HasSuffix(r.URL.Path, "/screen-state") {
			if r.Method == "GET" {
				handlers.GetScreenState(w, r)
				return
			}
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.UpdateScreenState)(w, r)
				return
			}
		}

		if strings.HasSuffix(r.URL.Path, "/screen-stream") {
			if r.Method == "GET" {
				// Note: SSE stream should not require standard verify auth because EventSource in browsers
				// doesn't send Authorization headers easily. Maybe check PIN or rely on public view.
				// Since BigScreen just watches, let's make it public.
				handlers.ScreenStream(w, r)
				return
			}
		}

		if strings.HasSuffix(r.URL.Path, "/slides") {
			if r.Method == "GET" {
				handlers.GetSlides(w, r)
				return
			}
			if r.Method == "PUT" {
				handlers.VerifyAuth(handlers.SaveSlides)(w, r)
				return
			}
		}

		switch r.Method {
		case "GET":
			handlers.GetEvent(w, r)
		case "PUT":
			handlers.VerifyAuth(handlers.UpdateEvent)(w, r)
		case "DELETE":
			handlers.VerifyAuth(handlers.DeleteEvent)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Screensavers library
	http.HandleFunc("/screensavers", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		switch r.Method {
		case "GET":
			handlers.GetScreensavers(w, r)
		case "POST":
			handlers.VerifyAuth(handlers.CreateScreensaver)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/screensavers/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "DELETE" {
			handlers.VerifyAuth(handlers.DeleteScreensaver)(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Tracks
	http.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "GET" {
			handlers.GetTracks(w, r)
		} else if r.Method == "POST" {
			handlers.VerifyAuth(handlers.CreateTrack)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/tracks/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "DELETE" {
			handlers.VerifyAuth(handlers.DeleteTrack)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Media upload route
	http.HandleFunc("/media/upload", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method == "POST" {
			handlers.VerifyAuth(handlers.UploadMedia)(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Media routes with ID
	http.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		path := r.URL.Path
		// Check if it ends with /file
		if strings.HasSuffix(path, "/file") {
			if r.Method == "GET" {
				handlers.ServeMedia(w, r)
				return
			}
		} else {
			// /media/{id}
			switch r.Method {
			case "GET":
				handlers.GetMedia(w, r)
				return
			case "DELETE":
				handlers.VerifyAuth(handlers.DeleteMedia)(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Serve the Vue SPA from the ./dist directory (built by `npm run build`).
	// Any path not matched by an API handler above falls through here.
	// Unknown paths get index.html so Vue Router handles client-side routing.
	distDir := "./dist"
	if _, err := os.Stat(distDir); err == nil {
		fs := http.FileServer(http.Dir(distDir))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			utils.DebugLog("Static request: %s %s", r.Method, r.URL.Path)
			enableCors(w, r)
			// If the requested file exists in dist/, serve it directly.
			// Otherwise fall back to index.html for Vue Router history mode.
			if _, err := os.Stat(distDir + r.URL.Path); os.IsNotExist(err) {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	} else {
		// No dist/ folder — local dev mode, return a simple JSON health check.
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		})
	}

	port := ":8080"
	utils.DebugLog("Server listening on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		utils.DebugLog("Server failed: %v", err) // Use utils.DebugLog here
	}
}

func enableCors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	}
}
