package hub

type ClientMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
