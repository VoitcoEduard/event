package models

import (
	"strings"
	"time"
)

const OtherOptionValue = "__other__"

type Response struct {
	ID           string            `json:"id"`
	EventID      string            `json:"eventId"`
	ModuleID     string            `json:"moduleId"`
	UserID       string            `json:"userId"`
	Data         map[string]any    `json:"data"`
	OtherAnswers map[string]string `json:"otherAnswers,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
}

// ResolveOtherAnswers replaces "__other__" sentinel values in Data with the
// corresponding trimmed custom text from OtherAnswers.
func (r *Response) ResolveOtherAnswers() {
	if r == nil || len(r.Data) == 0 || len(r.OtherAnswers) == 0 {
		return
	}
	for fieldID, rawVal := range r.Data {
		customText := strings.TrimSpace(r.OtherAnswers[fieldID])
		if customText == "" {
			continue
		}
		switch v := rawVal.(type) {
		case string:
			if v == OtherOptionValue {
				r.Data[fieldID] = customText
			}
		case []any:
			replaced := make([]any, len(v))
			for i, item := range v {
				if s, ok := item.(string); ok && s == OtherOptionValue {
					replaced[i] = customText
				} else {
					replaced[i] = item
				}
			}
			r.Data[fieldID] = replaced
		case []string:
			replaced := make([]string, len(v))
			for i, item := range v {
				if item == OtherOptionValue {
					replaced[i] = customText
				} else {
					replaced[i] = item
				}
			}
			r.Data[fieldID] = replaced
		}
	}
}
