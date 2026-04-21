package room

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

func TestRoomActor_CreateAndDestroy(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	assert.NotNil(t, actor)
	assert.Equal(t, "test-room", actor.roomID)
	assert.Equal(t, "normal", actor.state.RoomType)
	assert.Equal(t, int32(4), actor.state.MaxPlayers)
	assert.Equal(t, proto.RoomStatus_WAITING, actor.state.Status)

	actor.Shutdown()
}

func TestRoomActor_AddPlayer(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	reply := make(chan error)
	event := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply,
	}

	err := actor.SendEvent(event)
	assert.NoError(t, err)

	result := <-reply
	assert.NoError(t, result)

	state := actor.GetState()
	assert.Equal(t, int32(1), state.CurrentPlayers)
	assert.Contains(t, state.Players, "player1")
}

func TestRoomActor_AddPlayerRoomFull(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 2, 1)
	defer actor.Shutdown()

	// Add first player
	reply1 := make(chan error)
	event1 := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply1,
	}
	actor.SendEvent(event1)
	<-reply1

	// Add second player
	reply2 := make(chan error)
	event2 := &AddPlayerEvent{
		PlayerID: "player2",
		Name:     "Player 2",
		Level:    1,
		Reply:    reply2,
	}
	actor.SendEvent(event2)
	<-reply2

	// Try to add third player (should fail)
	reply3 := make(chan error)
	event3 := &AddPlayerEvent{
		PlayerID: "player3",
		Name:     "Player 3",
		Level:    1,
		Reply:    reply3,
	}
	actor.SendEvent(event3)
	result := <-reply3

	assert.Error(t, result)
	assert.Contains(t, result.Error(), "room is full")
}

func TestRoomActor_RemovePlayer(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	// Add player first
	reply1 := make(chan error)
	event1 := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply1,
	}
	actor.SendEvent(event1)
	<-reply1

	// Remove player
	reply2 := make(chan error)
	event2 := &RemovePlayerEvent{
		PlayerID: "player1",
		Reply:    reply2,
	}
	actor.SendEvent(event2)
	result := <-reply2

	assert.NoError(t, result)

	state := actor.GetState()
	assert.Equal(t, int32(0), state.CurrentPlayers)
	assert.NotContains(t, state.Players, "player1")
}

func TestRoomActor_StartGame(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	// Add player first
	reply1 := make(chan error)
	event1 := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply1,
	}
	actor.SendEvent(event1)
	<-reply1

	// Start game
	reply2 := make(chan error)
	event2 := &StartGameEvent{
		Reply: reply2,
	}
	actor.SendEvent(event2)
	result := <-reply2

	assert.NoError(t, result)

	state := actor.GetState()
	assert.Equal(t, proto.RoomStatus_PLAYING, state.Status)
	assert.False(t, state.StartedAt.IsZero())
}

func TestRoomActor_StartGameNoPlayers(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	// Try to start game without players
	reply := make(chan error)
	event := &StartGameEvent{
		Reply: reply,
	}
	actor.SendEvent(event)
	result := <-reply

	assert.Error(t, result)
	assert.Contains(t, result.Error(), "not enough players")
}

func TestRoomActor_UpdatePlayerPosition(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	// Add player first
	reply1 := make(chan error)
	event1 := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply1,
	}
	actor.SendEvent(event1)
	<-reply1

	// Update position
	reply2 := make(chan error)
	event2 := &UpdatePlayerPositionEvent{
		PlayerID: "player1",
		Position: &proto.Position{X: 10.0, Y: 20.0, Z: 0.0},
		Reply:    reply2,
	}
	actor.SendEvent(event2)
	result := <-reply2

	assert.NoError(t, result)

	state := actor.GetState()
	player := state.Players["player1"]
	assert.NotNil(t, player)
	assert.Equal(t, float32(10.0), player.Position.X)
	assert.Equal(t, float32(20.0), player.Position.Y)
}

func TestRoomActor_GetRoomInfo(t *testing.T) {
	actor := NewRoomActor("test-room", "normal", 4, 1)
	defer actor.Shutdown()

	// Add player
	reply1 := make(chan error)
	event1 := &AddPlayerEvent{
		PlayerID: "player1",
		Name:     "Player 1",
		Level:    1,
		Reply:    reply1,
	}
	actor.SendEvent(event1)
	<-reply1

	// Get room info
	reply := make(chan *proto.RoomInfo)
	event := &GetRoomInfoEvent{
		Reply: reply,
	}
	actor.SendEvent(event)
	roomInfo := <-reply

	assert.NotNil(t, roomInfo)
	assert.Equal(t, "test-room", roomInfo.RoomId)
	assert.Equal(t, "normal", roomInfo.RoomType)
	assert.Equal(t, int32(4), roomInfo.MaxPlayers)
	assert.Equal(t, int32(1), roomInfo.CurrentPlayers)
	assert.Equal(t, proto.RoomStatus_WAITING, roomInfo.Status)
	assert.Len(t, roomInfo.Players, 1)
	assert.Equal(t, "player1", roomInfo.Players[0].PlayerId)
}

func TestGameLoop_TickRate(t *testing.T) {
	gl := NewGameLoop(100 * time.Millisecond)
	defer gl.Shutdown()

	assert.NotNil(t, gl)
	assert.Equal(t, 100*time.Millisecond, gl.tickRate)
	assert.Equal(t, uint64(0), gl.currentTick)
}

func TestGameLoop_StartAndStop(t *testing.T) {
	gl := NewGameLoop(100 * time.Millisecond)
	gl.Start()

	// Wait a few ticks
	time.Sleep(350 * time.Millisecond)

	// Add timeout to prevent deadlock
	done := make(chan bool)
	go func() {
		gl.Shutdown()
		done <- true
	}()

	select {
	case <-done:
		// Should have processed at least 3 ticks
		assert.GreaterOrEqual(t, gl.currentTick, uint64(3))
		assert.LessOrEqual(t, gl.currentTick, uint64(5))
	case <-time.After(5 * time.Second):
		t.Fatal("Game loop shutdown timeout")
	}
}

func TestRoomManager_CreateRoom(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()
	resp, err := rm.CreateRoom(ctx, "normal", 4, 1)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.RoomId)
	assert.Equal(t, int32(4), resp.MaxPlayers)

	actor, exists := rm.GetRoom(resp.RoomId)
	assert.True(t, exists)
	assert.NotNil(t, actor)
}

func TestRoomManager_AddPlayerToRoom(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()
	resp, err := rm.CreateRoom(ctx, "normal", 4, 1)
	assert.NoError(t, err)

	err = rm.AddPlayerToRoom(ctx, resp.RoomId, "player1", "Player 1", 1)
	assert.NoError(t, err)

	actor, exists := rm.GetRoom(resp.RoomId)
	assert.True(t, exists)

	state := actor.GetState()
	assert.Contains(t, state.Players, "player1")
}

func TestRoomManager_RemovePlayerFromRoom(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()
	resp, err := rm.CreateRoom(ctx, "normal", 4, 1)
	assert.NoError(t, err)

	rm.AddPlayerToRoom(ctx, resp.RoomId, "player1", "Player 1", 1)

	err = rm.RemovePlayerFromRoom(ctx, resp.RoomId, "player1")
	assert.NoError(t, err)

	actor, exists := rm.GetRoom(resp.RoomId)
	assert.True(t, exists)

	state := actor.GetState()
	assert.NotContains(t, state.Players, "player1")
}

func TestRoomManager_GetRoomInfo(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()
	resp, err := rm.CreateRoom(ctx, "normal", 4, 1)
	assert.NoError(t, err)

	rm.AddPlayerToRoom(ctx, resp.RoomId, "player1", "Player 1", 1)

	roomInfo, err := rm.GetRoomInfo(ctx, resp.RoomId)
	assert.NoError(t, err)
	assert.NotNil(t, roomInfo)
	assert.Equal(t, resp.RoomId, roomInfo.RoomId)
	assert.Equal(t, int32(1), roomInfo.CurrentPlayers)
}

func TestRoomManager_ListRooms(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()

	// Create multiple rooms
	resp1, _ := rm.CreateRoom(ctx, "normal", 4, 1)
	resp2, _ := rm.CreateRoom(ctx, "hard", 2, 2)
	resp3, _ := rm.CreateRoom(ctx, "normal", 4, 1)

	rooms := rm.ListRooms()
	assert.Len(t, rooms, 3)

	roomIDs := []string{resp1.RoomId, resp2.RoomId, resp3.RoomId}
	for _, room := range rooms {
		assert.Contains(t, roomIDs, room.RoomId)
	}
}

func TestRoomManager_GetRoomCount(t *testing.T) {
	rm := NewRoomManager(nil)
	defer rm.Shutdown()

	ctx := context.Background()

	// Create some rooms
	rm.CreateRoom(ctx, "normal", 4, 1)
	rm.CreateRoom(ctx, "hard", 2, 2)
	rm.CreateRoom(ctx, "normal", 4, 1)

	count := rm.GetRoomCount()
	assert.Equal(t, 3, count)
}

func TestRoomState_InitialState(t *testing.T) {
	state := &RoomState{
		RoomID:         "test-room",
		RoomType:       "normal",
		MaxPlayers:     4,
		CurrentPlayers: 0,
		Status:         proto.RoomStatus_WAITING,
		Players:        make(map[string]*PlayerState),
		CurrentWave:    0,
		Score:          0,
		GameTime:       0,
		Difficulty:     1,
		Tick:           0,
		LastUpdate:     time.Now(),
		CreatedAt:      time.Now(),
	}

	assert.Equal(t, "test-room", state.RoomID)
	assert.Equal(t, "normal", state.RoomType)
	assert.Equal(t, int32(4), state.MaxPlayers)
	assert.Equal(t, int32(0), state.CurrentPlayers)
	assert.Equal(t, proto.RoomStatus_WAITING, state.Status)
	assert.Equal(t, int32(0), state.CurrentWave)
	assert.Equal(t, int32(0), state.Score)
	assert.Equal(t, uint64(0), state.Tick)
}

func TestPlayerState_InitialState(t *testing.T) {
	state := &PlayerState{
		PlayerID:   "player1",
		Name:       "Player 1",
		Level:      1,
		Health:     100,
		MaxHealth:  100,
		Position:   &proto.Position{X: 0, Y: 0, Z: 0},
		Score:      0,
		LastAction: time.Now(),
		JoinedAt:   time.Now(),
	}

	assert.Equal(t, "player1", state.PlayerID)
	assert.Equal(t, "Player 1", state.Name)
	assert.Equal(t, int32(1), state.Level)
	assert.Equal(t, int32(100), state.Health)
	assert.Equal(t, int32(100), state.MaxHealth)
	assert.Equal(t, float32(0), state.Position.X)
	assert.Equal(t, float32(0), state.Position.Y)
	assert.Equal(t, float32(0), state.Position.Z)
	assert.Equal(t, int32(0), state.Score)
}
