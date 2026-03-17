package models

type DynamicCardType string

const (
	DynamicCardTypeInfo   DynamicCardType = "info"
	DynamicCardTypeAlert  DynamicCardType = "alert"
	DynamicCardTypeAction DynamicCardType = "action"
)

type DynamicCard struct {
	ID        string          `json:"id" bson:"_id"`
	EventID   string          `json:"eventId" bson:"eventId"`
	Title     string          `json:"title" bson:"title"`
	Content   string          `json:"content" bson:"content"`
	Icon      string          `json:"icon" bson:"icon"`           // Emoji or icon name
	Color     string          `json:"color" bson:"color"`         // Hex color
	Type      DynamicCardType `json:"type" bson:"type"`
	TargetURL string          `json:"targetUrl,omitempty" bson:"targetUrl,omitempty"`
	IsActive  bool            `json:"isActive" bson:"isActive"`
}
