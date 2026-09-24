package models

import "time"

type TeamResult struct {
	EventID    string     `json:"eventId" bson:"eventId"`
	ModuleID   string     `json:"moduleId" bson:"moduleId"`
	ModuleName string     `json:"moduleName" bson:"moduleName"`
	FinishedAt time.Time  `json:"finishedAt" bson:"finishedAt"`
	Teams      []TeamInfo `json:"teams" bson:"teams"`
}

type TeamInfo struct {
	ID      string       `json:"id" bson:"id"`
	Name    string       `json:"name" bson:"name"`
	Color   string       `json:"color" bson:"color"`
	Members []TeamMember `json:"members" bson:"members"`
}

type TeamMember struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
}
