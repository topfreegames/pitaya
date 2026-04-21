package room

import (
	"fmt"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

type AddPlayerEvent struct {
	PlayerID string
	Name     string
	Level    int32
	Reply    chan error
}

func (e *AddPlayerEvent) Process(state *RoomState) error {
	var err error

	if state.Status != proto.RoomStatus_WAITING {
		err = fmt.Errorf("room is not in waiting status")
	} else if state.CurrentPlayers >= state.MaxPlayers {
		err = fmt.Errorf("room is full")
	} else if _, exists := state.Players[e.PlayerID]; exists {
		err = fmt.Errorf("player already in room")
	} else {
		state.Players[e.PlayerID] = &PlayerState{
			PlayerID:   e.PlayerID,
			Name:       e.Name,
			Level:      e.Level,
			Health:     100,
			MaxHealth:  100,
			Position:   &proto.Position{X: 0, Y: 0, Z: 0},
			Score:      0,
			LastAction: time.Now(),
			JoinedAt:   time.Now(),
		}
		state.CurrentPlayers++
		state.LastUpdate = time.Now()
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type RemovePlayerEvent struct {
	PlayerID string
	Reply    chan error
}

func (e *RemovePlayerEvent) Process(state *RoomState) error {
	var err error

	if _, exists := state.Players[e.PlayerID]; !exists {
		err = fmt.Errorf("player not found in room")
	} else {
		delete(state.Players, e.PlayerID)
		state.CurrentPlayers--
		state.LastUpdate = time.Now()

		if state.CurrentPlayers == 0 && state.Status == proto.RoomStatus_WAITING {
			state.Status = proto.RoomStatus_FINISHED
			state.FinishedAt = time.Now()
		}
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type StartGameEvent struct {
	Reply chan error
}

func (e *StartGameEvent) Process(state *RoomState) error {
	var err error

	if state.Status != proto.RoomStatus_WAITING {
		err = fmt.Errorf("room is not in waiting status")
	} else if state.CurrentPlayers < 1 {
		err = fmt.Errorf("not enough players to start game")
	} else {
		state.Status = proto.RoomStatus_PLAYING
		state.StartedAt = time.Now()
		state.LastUpdate = time.Now()
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type UpdatePlayerPositionEvent struct {
	PlayerID string
	Position *proto.Position
	Reply    chan error
}

func (e *UpdatePlayerPositionEvent) Process(state *RoomState) error {
	var err error

	player, exists := state.Players[e.PlayerID]
	if !exists {
		err = fmt.Errorf("player not found in room")
	} else {
		player.Position = e.Position
		player.LastAction = time.Now()
		state.LastUpdate = time.Now()
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type UpdatePlayerHealthEvent struct {
	PlayerID string
	Health   int32
	Reply    chan error
}

func (e *UpdatePlayerHealthEvent) Process(state *RoomState) error {
	var err error

	player, exists := state.Players[e.PlayerID]
	if !exists {
		err = fmt.Errorf("player not found in room")
	} else {
		player.Health = e.Health
		player.LastAction = time.Now()
		state.LastUpdate = time.Now()
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type UpdatePlayerScoreEvent struct {
	PlayerID string
	Score    int32
	Reply    chan error
}

func (e *UpdatePlayerScoreEvent) Process(state *RoomState) error {
	var err error

	player, exists := state.Players[e.PlayerID]
	if !exists {
		err = fmt.Errorf("player not found in room")
	} else {
		player.Score = e.Score
		player.LastAction = time.Now()
		state.LastUpdate = time.Now()
	}

	select {
	case e.Reply <- err:
	default:
	}

	return err
}

type GetRoomInfoEvent struct {
	Reply chan *proto.RoomInfo
}

func (e *GetRoomInfoEvent) Process(state *RoomState) error {
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

	select {
	case e.Reply <- roomInfo:
	default:
	}

	return nil
}
