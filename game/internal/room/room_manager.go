package room

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

type RoomManager struct {
	app      pitaya.Pitaya
	rooms    map[string]*RoomActor
	mu       sync.RWMutex
	shutdown chan struct{}
	done     chan struct{}
}

func NewRoomManager(app pitaya.Pitaya) *RoomManager {
	rm := &RoomManager{
		app:      app,
		rooms:    make(map[string]*RoomActor),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}

	go rm.run()
	return rm
}

func (rm *RoomManager) run() {
	defer close(rm.done)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.cleanupExpiredRooms()

		case <-rm.shutdown:
			rm.shutdownAllRooms()
			return
		}
	}
}

func (rm *RoomManager) CreateRoom(ctx context.Context, roomType string, maxPlayers int32, difficulty int32) (*proto.CreateRoomResponse, error) {
	roomID := uuid.New().String()

	actor := NewRoomActor(roomID, roomType, maxPlayers, difficulty)

	rm.mu.Lock()
	rm.rooms[roomID] = actor
	rm.mu.Unlock()

	return &proto.CreateRoomResponse{
		RoomId:    roomID,
		RoomToken: roomID,
		MaxPlayers: maxPlayers,
	}, nil
}

func (rm *RoomManager) GetRoom(roomID string) (*RoomActor, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	actor, exists := rm.rooms[roomID]
	return actor, exists
}

func (rm *RoomManager) AddPlayerToRoom(ctx context.Context, roomID, playerID, name string, level int32) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &AddPlayerEvent{
		PlayerID: playerID,
		Name:     name,
		Level:    level,
		Reply:    reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) RemovePlayerFromRoom(ctx context.Context, roomID, playerID string) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &RemovePlayerEvent{
		PlayerID: playerID,
		Reply:    reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) StartGame(ctx context.Context, roomID string) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &StartGameEvent{
		Reply: reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) GetRoomInfo(ctx context.Context, roomID string) (*proto.RoomInfo, error) {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan *proto.RoomInfo)
	event := &GetRoomInfoEvent{
		Reply: reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return nil, err
	}

	return <-reply, nil
}

func (rm *RoomManager) UpdatePlayerPosition(ctx context.Context, roomID, playerID string, position *proto.Position) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &UpdatePlayerPositionEvent{
		PlayerID: playerID,
		Position: position,
		Reply:    reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) UpdatePlayerHealth(ctx context.Context, roomID, playerID string, health int32) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &UpdatePlayerHealthEvent{
		PlayerID: playerID,
		Health:   health,
		Reply:    reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) UpdatePlayerScore(ctx context.Context, roomID, playerID string, score int32) error {
	actor, exists := rm.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	reply := make(chan error)
	event := &UpdatePlayerScoreEvent{
		PlayerID: playerID,
		Score:    score,
		Reply:    reply,
	}

	if err := actor.SendEvent(event); err != nil {
		return err
	}

	return <-reply
}

func (rm *RoomManager) cleanupExpiredRooms() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	expiredRooms := []string{}

	for roomID, actor := range rm.rooms {
		state := actor.GetState()
		if now.Sub(state.LastUpdate) > RoomTTL {
			expiredRooms = append(expiredRooms, roomID)
		}
	}

	for _, roomID := range expiredRooms {
		fmt.Printf("Cleaning up expired room: %s\n", roomID)
		actor := rm.rooms[roomID]
		actor.Shutdown()
		delete(rm.rooms, roomID)
	}
}

func (rm *RoomManager) shutdownAllRooms() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for roomID, actor := range rm.rooms {
		fmt.Printf("Shutting down room: %s\n", roomID)
		actor.Shutdown()
		delete(rm.rooms, roomID)
	}
}

func (rm *RoomManager) Shutdown() {
	close(rm.shutdown)
	<-rm.done
}

func (rm *RoomManager) GetRoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

func (rm *RoomManager) ListRooms() []*proto.RoomInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]*proto.RoomInfo, 0, len(rm.rooms))
	for _, actor := range rm.rooms {
		state := actor.GetState()
		roomInfo := &proto.RoomInfo{
			RoomId:         state.RoomID,
			RoomType:       state.RoomType,
			MaxPlayers:     state.MaxPlayers,
			CurrentPlayers: state.CurrentPlayers,
			Status:         state.Status,
			CurrentWave:    state.CurrentWave,
			Score:          state.Score,
			Players:        make([]*proto.PlayerInfo, 0, len(state.Players)),
		}

		for _, player := range state.Players {
			roomInfo.Players = append(roomInfo.Players, &proto.PlayerInfo{
				PlayerId:  player.PlayerID,
				Name:       player.Name,
				Level:      player.Level,
				Health:     player.Health,
				MaxHealth:  player.MaxHealth,
				Position:   player.Position,
				Score:      player.Score,
			})
		}

		rooms = append(rooms, roomInfo)
	}

	return rooms
}
