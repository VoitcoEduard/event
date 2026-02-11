package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
)

type Response struct {
	Message string `json:"message"`
}

func main() {
	DebugLog("Starting server initialization...")

	// Connect to MongoDB
	db.ConnectDB()

	// Initialize QuizHub
	quizHub := hub.NewQuizHub()
	go quizHub.Run()

	// Inject QuizHub into handlers
	handlers.SetQuizHub(quizHub)

	// WebSocket Endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(quizHub, w, r)
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
			handlers.CreateEvent(w, r)
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
			handlers.SaveTemplate(w, r)
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
			handlers.UpdateTemplate(w, r)
			return
		}
		if r.Method == "DELETE" {
			handlers.DeleteTemplate(w, r)
			return
		} // Handle other methods if needed
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
				handlers.UpdateModuleStatus(w, r)
				return
			}
		}

		if strings.Contains(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/publish") {
			if r.Method == "PUT" {
				handlers.PublishModule(w, r)
				return
			}
		}

		// Quiz routes
		if strings.Contains(r.URL.Path, "/modules/") && strings.Contains(r.URL.Path, "/quiz/") {
			if strings.HasSuffix(r.URL.Path, "/questions") && r.Method == "PUT" {
				handlers.UpdateQuizQuestions(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/state") && r.Method == "GET" {
				handlers.GetQuizState(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/start") && r.Method == "POST" {
				handlers.StartQuiz(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/next") && r.Method == "POST" {
				handlers.NextQuestion(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/results") && r.Method == "GET" {
				handlers.GetQuizResults(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/reset") && r.Method == "POST" {
				handlers.ResetQuiz(w, r)
				return
			}
		}

		if strings.HasSuffix(r.URL.Path, "/modules") {
			if r.Method == "POST" {
				handlers.AddModule(w, r)
				return
			}
		}

		// Check for /status suffix
		if len(r.URL.Path) > 7 && r.URL.Path[len(r.URL.Path)-7:] == "/status" {
			if r.Method == "PUT" {
				handlers.UpdateEventStatus(w, r)
				return
			}
		}

		switch r.Method {
		case "GET":
			handlers.GetEvent(w, r)
		case "PUT":
			handlers.UpdateEvent(w, r)
		case "DELETE":
			handlers.DeleteEvent(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		DebugLog("Received request: %s %s", r.Method, r.URL.Path)
		enableCors(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := Response{Message: "Hello from Go Backend!"}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			DebugLog("Error encoding response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	port := ":8080"
	DebugLog("Server listening on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

func enableCors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	}
}
