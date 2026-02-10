package hub

// Client -> Server
type ClientMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Server -> Client
type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
