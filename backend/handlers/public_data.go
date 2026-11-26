package handlers

import (
	"encoding/json"

	"github.com/user/go-expo-sample/backend/models"
)

func publicModule(module models.Module) models.Module {
	if module.Type != models.ModuleTypeSimpleQuiz && module.Type != models.ModuleTypeQuiz && module.Type != models.ModuleTypeWrongPictures {
		return module
	}
	data, err := json.Marshal(module.Data)
	if err != nil {
		module.Data = nil
		return module
	}
	var filtered map[string]any
	if err := json.Unmarshal(data, &filtered); err != nil {
		module.Data = nil
		return module
	}
	delete(filtered, "judgeToken")
	if module.Type == models.ModuleTypeSimpleQuiz {
		delete(filtered, "questions")
	} else {
		delete(filtered, "questions")
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
