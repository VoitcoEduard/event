package models

import "time"

type TeamResult struct {
	EventID    string     `json:"eventId"`
	ModuleID   string     `json:"moduleId"`
	ModuleName string     `json:"moduleName"`
	FinishedAt time.Time  `json:"finishedAt"`
	Teams      []TeamInfo `json:"teams"`
}

type TeamInfo struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Color   string       `json:"color"`
	Members []TeamMember `json:"members"`
}

type TeamMember struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}
