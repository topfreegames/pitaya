package services

import (
	"context"

	"github.com/topfreegames/pitaya/component"
)

// Room represents a component that contains a bundle of room related handler
type Room struct {
	component.Base
}

// NewRoom returns a new room
func NewRoom() *Room {
	return &Room{}
}

// Ping returns a pong
func (r *Room) Ping(ctx context.Context) ([]byte, error) {
	return []byte("pong"), nil
}
