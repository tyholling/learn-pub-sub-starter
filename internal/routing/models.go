package routing

import "time"

type PlayingState struct {
	IsPaused bool `json:"is_paused"`
}

type GameLog struct {
	CurrentTime time.Time
	Message     string
	Username    string
}
