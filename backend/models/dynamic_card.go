package models

type DynamicCardType string

const (
	DynamicCardTypeInfo   DynamicCardType = "info"
	DynamicCardTypeAlert  DynamicCardType = "alert"
	DynamicCardTypeAction DynamicCardType = "action"
)

type DynamicCard struct {
	ID        string          `json:"id"`
	EventID   string          `json:"eventId"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Icon      string          `json:"icon"`  // Shared AppIcon name; unknown values use the card-type fallback
	Color     string          `json:"color"` // Hex color
	Type      DynamicCardType `json:"type"`
	TargetURL string          `json:"targetUrl,omitempty"`
	IsActive  bool            `json:"isActive"`
}
