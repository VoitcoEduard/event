package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils" // New import
	"go.mongodb.org/mongo-driver/bson"
)

// Generate a 6-digit random number string
func generateID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event.ID = generateID()
	event.IsActive = false // Default to inactive
	event.CreatedAt = time.Now()

	welcomeLabel := "What is your name?"
	switch event.Language {
	case "ru":
		welcomeLabel = "Как вас зовут?"
	case "ro":
		welcomeLabel = "Cum te numești?"
	}

	event.Modules = []models.Module{
		{
			ID:       generateID(),
			Type:     "form",
			Name:     "Welcome",
			IsActive: true,
			Data: map[string]any{
				"mandatoryOnEntry":    true,
				"hideAfterCompletion": true,
				"timeline": []map[string]any{
					{
						"id":   "step_welcome_name",
						"type": "field",
						"data": map[string]any{
							"label":        welcomeLabel,
							"fieldType":    "text",
							"compactLabel": "name",
						},
					},
				},
			},
		},
	}

	collection := db.GetCollection("events")
	_, err := collection.InsertOne(context.Background(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	utils.DebugLog("CreateEvent: Created event %s with %d modules", event.ID, len(event.Modules))
	json.NewEncoder(w).Encode(event)
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	collection := db.GetCollection("events")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	events := []models.Event{}
	if err := cursor.All(context.Background(), &events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func GetEvent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /events/{id}
	path := r.URL.Path
	prefix := "/events/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL (naive approach since using net/http)
	// Assuming /events/{id}
	id := r.URL.Path[len("/events/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{
		"name":        event.Name,
		"description": event.Description,
		"language":    event.Language,
	}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event updated"})
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	_, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event deleted"})
}

func ConnectToEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": input.Pin}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if !event.IsActive {
		http.Error(w, "Event is not active", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /events/{id}/status
	// Simple parsing assuming correct format
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/status"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": bson.M{"isActive": input.IsActive}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event status updated"})
}

func AddModule(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /events/{id}/modules
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/modules"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	// Use a temporary struct to parse the incoming JSON more flexibly
	var input struct {
		Type string         `json:"type"`
		Name string         `json:"name"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create the module with proper type conversion
	module := models.Module{
		ID:       generateID(),
		Type:     models.ModuleType(input.Type),
		Name:     input.Name,
		IsActive: false,
		Data:     make(map[string]any),
	}

	// Copy data if provided
	if input.Data != nil {
		module.Data = input.Data
	}

	collection := db.GetCollection("events")

	// Pre-check/Fix: Ensure modules array is initialized if it's currently null
	// This fixes existing events where modules might be null
	_, _ = collection.UpdateOne(context.Background(), bson.M{"_id": id, "modules": nil}, bson.M{"$set": bson.M{"modules": []models.Module{}}})

	filter := bson.M{"_id": id}
	update := bson.M{"$push": bson.M{"modules": module}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(module)
}

func UpdateModuleStatus(w http.ResponseWriter, r *http.Request) {
	// Extract IDs from URL: /events/{id}/modules/{moduleId}/status
	// This parsing is getting complicated with standard net/http, but sticking to it for consistency.
	// Format: /events/123456/modules/654321/status
	path := r.URL.Path
	parts := make([]string, 0)
	// simple split
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	// Expected key parts: ["events", "123456", "modules", "654321", "status"]
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	moduleID := parts[3]

	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": eventID, "modules.id": moduleID}
	update := bson.M{"$set": bson.M{"modules.$.isActive": input.IsActive}}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		http.Error(w, "Event or Module not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Module status updated"})
}

func SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if template.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	template.ID = generateID() // Simple ID, or use UUID in real app
	template.CreatedAt = time.Now()

	collection := db.GetCollection("templates")
	_, err := collection.InsertOne(context.Background(), template)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func GetTemplates(w http.ResponseWriter, r *http.Request) {
	collection := db.GetCollection("templates")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var templates []models.Template
	if err := cursor.All(context.Background(), &templates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// PublishModule finalizes configuration and makes it ready (conceptually "creates table")
func PublishModule(w http.ResponseWriter, r *http.Request) {
	// URL: /events/{id}/modules/{moduleId}/publish
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	// Expected: ["events", "123456", "modules", "654321", "publish"]
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	moduleID := parts[3]

	var input struct {
		Data map[string]any `json:"data"` // Final configuration data
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": eventID, "modules.id": moduleID}

	update := bson.M{
		"$set": bson.M{
			"modules.$.data": input.Data,
		},
	}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		http.Error(w, "Event or Module not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Module published/configured"})
}

func SubmitModuleResponse(w http.ResponseWriter, r *http.Request) {
	// URL: /events/{id}/modules/{moduleId}/response
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	// Expected: ["events", "123456", "modules", "654321", "response"]
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	moduleID := parts[3]

	var response models.Response
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response.EventID = eventID
	response.ModuleID = moduleID
	response.CreatedAt = time.Now()

	collection := db.GetCollection("responses")
	_, err := collection.InsertOne(context.Background(), response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	utils.DebugLog("SubmitModuleResponse: Submitted response for EventID: %s, ModuleID: %s, UserID: %s", eventID, moduleID, response.UserID)
	json.NewEncoder(w).Encode(response)
}

func DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /templates/{id}
	path := r.URL.Path
	prefix := "/templates/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	collection := db.GetCollection("templates")
	_, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template deleted"})
}

func UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /templates/{id}
	path := r.URL.Path
	prefix := "/templates/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	var template models.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("templates")
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"name":        template.Name,
			"description": template.Description,
			"data":        template.Data,
			"moduleType":  template.ModuleType,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template updated"})
}

func DeleteModule(w http.ResponseWriter, r *http.Request) {
	// URL: /events/{id}/modules/{moduleId}
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Expected: ["events", "123456", "modules", "654321"]
	// Note: if the URL ends with a slash, there effectively won't be an empty string appended if we use split logic carefully,
	// but purely splitting by '/' usually involves an empty string at start if it starts with /.
	// The current manual split logic skims leading slash if implemented as above loop?
	// Re-checking the loop:
	// "" -> '/' -> current="" -> parts append ""(skip empty check?) -> No, loop says `if current != ""`.
	// So for "/events/1/modules/2":
	// '/' -> push "" (if not empty) -> skipped
	// 'e'... -> current="events"
	// '/' -> push "events"
	// '1'...
	// So parts should be ["events", "1", "modules", "2"]

	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	eventID := parts[1]
	moduleID := parts[3]

	collection := db.GetCollection("events")

	// Use $pull to remove the module with the specific ID
	filter := bson.M{"_id": eventID}
	update := bson.M{"$pull": bson.M{"modules": bson.M{"id": moduleID}}}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.ModifiedCount == 0 {
		http.Error(w, "Event not found or module not found", http.StatusNotFound)
		return
	}

	// Also delete any responses associated with this module?
	// It's good practice to cleanup, but maybe optional. For now, let's leave responses.
	// Users might want to keep data even if module is deleted from active list?
	// Usually "delete module" implies "remove availability".

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Module deleted"})
}

// UpdateModule handles PUT /events/{eventId}/modules/{moduleId}
// Updates the module's data (e.g., background music settings)
func UpdateModule(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	eventID := parts[1]
	moduleID := parts[3]

	// Parse the input - expecting { data: { ... } }
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": eventID, "modules.id": moduleID}

	// Build the update dynamically based on what's in the data
	if input.Data != nil {
		updateSet := make(map[string]any)
		for key, value := range input.Data {
			updateSet["modules.$.data."+key] = value
		}

		update := bson.M{"$set": updateSet}
		_, err := collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Module updated"})
}

// GetModule handles GET /events/{eventId}/modules/{moduleId}
// Returns the module data (e.g., for BigScreen to load music settings)
func GetModule(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	eventID := parts[1]
	moduleID := parts[3]

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Find the module
	for _, mod := range event.Modules {
		if mod.ID == moduleID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mod)
			return
		}
	}

	http.Error(w, "Module not found", http.StatusNotFound)
}

// GetModuleResponses handles the GET request to retrieve all responses for a specific module within an event.
// Expected URL format: /events/{eventId}/modules/{moduleId}/responses
func GetModuleResponses(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Expected: ["events", "123456", "modules", "654321", "responses"]
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		utils.DebugLog("GetModuleResponses: Invalid URL: %s", path)
		return
	}
	eventID := parts[1]
	moduleID := parts[3]

	// Determine module type first
	eventCollection := db.GetCollection("events")
	var event models.Event
	err := eventCollection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		utils.DebugLog("GetModuleResponses: Event not found: %v", err)
		return
	}

	var moduleType models.ModuleType
	for _, m := range event.Modules {
		if m.ID == moduleID {
			moduleType = m.Type
			break
		}
	}

	utils.DebugLog("GetModuleResponses: Fetching responses for EventID: %s, ModuleID: %s, Type: %s", eventID, moduleID, moduleType)

	if moduleType == "quiz" || moduleType == "wrong_pictures" {
		collection := db.GetCollection("quiz_results")
		filter := bson.M{"eventId": eventID, "moduleId": moduleID}

		cursor, err := collection.Find(context.Background(), filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			utils.DebugLog("GetModuleResponses: Error finding quiz results: %v", err)
			return
		}
		defer cursor.Close(context.Background())

		var results []models.QuizResult
		if err := cursor.All(context.Background(), &results); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			utils.DebugLog("GetModuleResponses: Error decoding quiz results: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"responses": results})
		utils.DebugLog("GetModuleResponses: Successfully returned %d quiz results", len(results))

	} else {
		// Default to form responses
		collection := db.GetCollection("responses")
		filter := bson.M{"eventId": eventID, "moduleId": moduleID}

		cursor, err := collection.Find(context.Background(), filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			utils.DebugLog("GetModuleResponses: Error finding responses: %v", err)
			return
		}
		defer cursor.Close(context.Background())

		var responses []models.Response
		if err := cursor.All(context.Background(), &responses); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			utils.DebugLog("GetModuleResponses: Error decoding responses: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]models.Response{"responses": responses})
		utils.DebugLog("GetModuleResponses: Successfully returned %d responses", len(responses))
	}
}

// UpdateModuleResponse handles updating a generic response document
// URL: /events/{id}/modules/{moduleId}/responses/{responseId}
func UpdateModuleResponse(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Expected: ["events", "123456", "modules", "654321", "responses", "respId"]
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	responseID := parts[5]

	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("responses")

	// We might need to handle ObjectID vs String ID depending on how it was inserted.
	// `SubmitModuleResponse` uses `bson:"_id,omitempty"` with String ID struct field,
	// but often Mongo driver generates ObjectID if empty.
	// Looking at `SubmitModuleResponse`, `response.ID` is coming from payload? No, standard struct.
	// If ID is not set, Mongo creates ObjectID.
	// Let's assume ID is passed as string in URL and we try direct string match first,
	// typically safer to stick to how it was read.

	filter := bson.M{"_id": responseID}
	update := bson.M{"$set": bson.M{"data": input.Data}}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		// Try ObjectID if string failed? Or assume user passed correct ID.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		utils.DebugLog("[UnifiedViewUpdate] Error updating response %s: %v", responseID, err)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Response not found", http.StatusNotFound)
		utils.DebugLog("[UnifiedViewUpdate] Response %s not found for update", responseID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Response updated"})
	utils.DebugLog("[UnifiedViewUpdate] Successfully updated response %s. Payload: %+v", responseID, input.Data)
}

// UpdateQuizScore handles updating a player's score in a quiz module
// URL: /events/{id}/modules/{moduleId}/quiz-scores/{userId}
func UpdateQuizScore(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Expected: ["events", "123456", "modules", "654321", "quiz-scores", "userId"]
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	moduleID := parts[3]
	userID := parts[5]

	var input struct {
		Score int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("quiz_results")
	// Find the result document for this module
	filter := bson.M{"eventId": eventID, "moduleId": moduleID, "players.id": userID}

	// We need to update the specific player's score in the players array
	update := bson.M{"$set": bson.M{"players.$.score": input.Score}}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		utils.DebugLog("[UnifiedViewUpdate] Error updating quiz score for user %s: %v", userID, err)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Quiz Result or Player not found", http.StatusNotFound)
		utils.DebugLog("[UnifiedViewUpdate] Quiz Result not found for user %s in module %s", userID, moduleID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Score updated"})
	utils.DebugLog("[UnifiedViewUpdate] Updated score for user %s in module %s. New Score: %d", userID, moduleID, input.Score)
}

// GetScreensaverConfig returns the screensaver configuration for an event
// URL: /events/{id}/screensaver
func GetScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	// Remove /screensaver suffix if present
	id = strings.TrimSuffix(id, "/screensaver")

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Return screensaver config or empty config if not set
	if event.ScreensaverConfig != nil {
		json.NewEncoder(w).Encode(event.ScreensaverConfig)
	} else {
		json.NewEncoder(w).Encode(models.ScreensaverConfig{})
	}
}

// UpdateScreensaverConfig updates the screensaver configuration for an event
// URL: /events/{id}/screensaver
func UpdateScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	// Remove /screensaver suffix if present
	id = strings.TrimSuffix(id, "/screensaver")

	var input models.ScreensaverConfig
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"screensaverConfig": input}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

// GetSlides handles GET /events/{eventId}/slides
func GetSlides(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	slides := event.Slides
	if len(slides) == 0 {
		joinUrl := "https://example.com/?pin=" + eventID // Note: adjust to your actual base URL if needed contextually, frontend handles the real one usually but for slide editor representation

		// Initialize default slides
		slides = []models.Slide{
			{
				ID:              generateID(),
				Type:            "welcome",
				Name:            "Welcome",
				BackgroundColor: "#1A1A1D",
				Elements: []models.CanvasElement{
					{
						ID:      generateID(),
						Type:    "qr",
						X:       15,
						Y:       25,
						Width:   25,
						Height:  45,
						Content: joinUrl,
					},
					{
						ID:        generateID(),
						Type:      "text",
						X:         45,
						Y:         25,
						Width:     45,
						Height:    20,
						Content:   "Join the Game!",
						FontSize:  70,
						Color:     "#FFFFFF",
						TextAlign: "left",
					},
					{
						ID:        generateID(),
						Type:      "text",
						X:         45,
						Y:         50,
						Width:     45,
						Height:    10,
						Content:   joinUrl,
						FontSize:  45,
						Color:     "#3b82f6",
						TextAlign: "left",
					},
					{
						ID:        generateID(),
						Type:      "text",
						X:         45,
						Y:         65,
						Width:     45,
						Height:    10,
						Content:   "PIN: " + eventID,
						FontSize:  60,
						Color:     "#FFD700",
						TextAlign: "left",
					},
				},
			},
			{ID: generateID(), Type: "screensaver", Name: "Screensaver", BackgroundColor: "#000000"},
		}

		// Save them back immediately
		update := bson.M{"$set": bson.M{"slides": slides}}
		collection.UpdateOne(context.Background(), bson.M{"_id": eventID}, update)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slides)
}

// SaveSlides handles PUT /events/{eventId}/slides
func SaveSlides(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	var input []models.Slide
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i := range input {
		if input[i].ID == "" {
			input[i].ID = generateID()
		}
	}

	collection := db.GetCollection("events")
	update := bson.M{"$set": bson.M{"slides": input}}

	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": eventID}, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}
