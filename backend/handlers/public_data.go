package handlers

import (
	"encoding/json"

	"github.com/user/go-expo-sample/backend/models"
)

// publicModule returns an independent projection so filtering never changes
// the data used by the host or stored in the database.
func publicModule(module models.Module) models.Module {
	if module.Type != models.ModuleTypeSimpleQuiz && module.Type != models.ModuleTypeQuiz {
		return module
	}
	data, err := json.Marshal(module.Data)
	if err != nil {
		module.Data = nil // Fail closed if an unexpected value cannot be encoded.
		return module
	}
	var filtered map[string]any
	if err := json.Unmarshal(data, &filtered); err != nil {
		module.Data = nil
		return module
	}
	delete(filtered, "judgeToken")
	if module.Type == models.ModuleTypeSimpleQuiz {
		// The live state endpoint exposes the current question and revealed answer.
		delete(filtered, "questions")
	} else if questions, ok := filtered["questions"].([]any); ok {
		for _, item := range questions {
			if question, ok := item.(map[string]any); ok {
				delete(question, "judgeNotes")
				delete(question, "correctAnswerText")
			}
		}
	}
	module.Data = filtered
	return module
}

func publicEvent(event models.Event) models.Event {
	modules := make([]models.Module, len(event.Modules))
	for i, module := range event.Modules {
		modules[i] = publicModule(module)
	}
	event.Modules = modules
	return event
}
