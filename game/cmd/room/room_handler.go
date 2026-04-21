package main

import (
	"context"
	"fmt"

	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/game/internal/proto"
	"github.com/topfreegames/pitaya/v3/game/internal/room"
)

type RoomHandler struct {
	component.Base
	app         pitaya.Pitaya
	roomManager *room.RoomManager
}

func NewRoomHandler(app pitaya.Pitaya) *RoomHandler {
	return &RoomHandler{
		app:         app,
		roomManager: room.NewRoomManager(app),
	}
}

func (r *RoomHandler) Init() {
}

func (r *RoomHandler) AfterInit() {
}

func (r *RoomHandler) BeforeShutdown() {
	r.roomManager.Shutdown()
}

func (r *RoomHandler) Shutdown() {
}

func (r *RoomHandler) CreateRoom(ctx context.Context, req *proto.CreateRoomRequest) (*proto.CreateRoomResponse, error) {
	if req.RoomType == "" {
		return nil, fmt.Errorf("room_type is required")
	}
	if req.MaxPlayers <= 0 || req.MaxPlayers > 10 {
		return nil, fmt.Errorf("max_players must be between 1 and 10")
	}

	resp, err := r.roomManager.CreateRoom(ctx, req.RoomType, req.MaxPlayers, req.Difficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	return resp, nil
}

func (r *RoomHandler) JoinRoom(ctx context.Context, req *proto.JoinRoomRequest) (*proto.JoinRoomResponse, error) {
	if req.RoomId == "" {
		return nil, fmt.Errorf("room_id is required")
	}

	roomInfo, err := r.roomManager.GetRoomInfo(ctx, req.RoomId)
	if err != nil {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get room info: %v", err),
		}, nil
	}

	if roomInfo.Status != proto.RoomStatus_WAITING {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: "Room is not in waiting status",
		}, nil
	}

	if roomInfo.CurrentPlayers >= roomInfo.MaxPlayers {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: "Room is full",
		}, nil
	}

	session := r.app.GetSessionFromCtx(ctx)
	if session == nil {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: "No session found",
		}, nil
	}

	playerID := session.UID()
	if playerID == "" {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: "No player ID found in session",
		}, nil
	}

	if err := r.roomManager.AddPlayerToRoom(ctx, req.RoomId, playerID, "Player", 1); err != nil {
		return &proto.JoinRoomResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to add player to room: %v", err),
		}, nil
	}

	updatedRoomInfo, err := r.roomManager.GetRoomInfo(ctx, req.RoomId)
	if err != nil {
		return &proto.JoinRoomResponse{
			Success: true,
			Message: "Joined room successfully",
		}, nil
	}

	return &proto.JoinRoomResponse{
		Success:  true,
		Message:  "Joined room successfully",
		RoomInfo: updatedRoomInfo,
	}, nil
}

func (r *RoomHandler) LeaveRoom(ctx context.Context, req *proto.LeaveRoomRequest) (*proto.LeaveRoomResponse, error) {
	if req.RoomId == "" {
		return nil, fmt.Errorf("room_id is required")
	}

	session := r.app.GetSessionFromCtx(ctx)
	if session == nil {
		return &proto.LeaveRoomResponse{
			Success: false,
		}, nil
	}

	playerID := session.UID()
	if playerID == "" {
		return &proto.LeaveRoomResponse{
			Success: false,
		}, nil
	}

	if err := r.roomManager.RemovePlayerFromRoom(ctx, req.RoomId, playerID); err != nil {
		return &proto.LeaveRoomResponse{
			Success: false,
		}, nil
	}

	return &proto.LeaveRoomResponse{
		Success: true,
	}, nil
}

func (r *RoomHandler) GetRoomInfo(ctx context.Context, req *proto.GetRoomInfoRequest) (*proto.GetRoomInfoResponse, error) {
	if req.RoomId == "" {
		return nil, fmt.Errorf("room_id is required")
	}

	roomInfo, err := r.roomManager.GetRoomInfo(ctx, req.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get room info: %w", err)
	}

	return &proto.GetRoomInfoResponse{
		RoomInfo: roomInfo,
	}, nil
}

func (r *RoomHandler) RoomList(ctx context.Context, req *proto.RoomListRequest) (*proto.RoomListResponse, error) {
	rooms := r.roomManager.ListRooms()

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	start := int(offset)
	if start > len(rooms) {
		start = len(rooms)
	}

	end := start + int(limit)
	if end > len(rooms) {
		end = len(rooms)
	}

	filteredRooms := rooms[start:end]

	return &proto.RoomListResponse{
		Rooms:      filteredRooms,
		TotalCount: int32(len(rooms)),
	}, nil
}

func (r *RoomHandler) Move(ctx context.Context, req *proto.MoveRequest) (*proto.MoveResponse, error) {
	session := r.app.GetSessionFromCtx(ctx)
	if session == nil {
		return nil, fmt.Errorf("no session found")
	}

	playerID := session.UID()
	if playerID == "" {
		return nil, fmt.Errorf("no player ID found in session")
	}

	roomID := r.getPlayerRoomID(playerID)
	if roomID == "" {
		return nil, fmt.Errorf("player not in any room")
	}

	if err := r.roomManager.UpdatePlayerPosition(ctx, roomID, playerID, req.TargetPosition); err != nil {
		return nil, fmt.Errorf("failed to update player position: %w", err)
	}

	return &proto.MoveResponse{
		Success:      true,
		NewPosition: req.TargetPosition,
		MovementCost: 0,
	}, nil
}

func (r *RoomHandler) Attack(ctx context.Context, req *proto.AttackRequest) (*proto.AttackResponse, error) {
	session := r.app.GetSessionFromCtx(ctx)
	if session == nil {
		return nil, fmt.Errorf("no session found")
	}

	playerID := session.UID()
	if playerID == "" {
		return nil, fmt.Errorf("no player ID found in session")
	}

	roomID := r.getPlayerRoomID(playerID)
	if roomID == "" {
		return nil, fmt.Errorf("player not in any room")
	}

	return &proto.AttackResponse{
		Success:     true,
		DamageDealt: 10,
		TargetKilled: false,
		NewScore:    0,
	}, nil
}

func (r *RoomHandler) UseSkill(ctx context.Context, req *proto.UseSkillRequest) (*proto.UseSkillResponse, error) {
	session := r.app.GetSessionFromCtx(ctx)
	if session == nil {
		return nil, fmt.Errorf("no session found")
	}

	playerID := session.UID()
	if playerID == "" {
		return nil, fmt.Errorf("no player ID found in session")
	}

	roomID := r.getPlayerRoomID(playerID)
	if roomID == "" {
		return nil, fmt.Errorf("player not in any room")
	}

	return &proto.UseSkillResponse{
		Success:          true,
		CooldownRemaining: 5,
		ManaCost:         10,
	}, nil
}

func (r *RoomHandler) getPlayerRoomID(playerID string) string {
	rooms := r.roomManager.ListRooms()
	for _, roomInfo := range rooms {
		for _, player := range roomInfo.Players {
			if player.PlayerId == playerID {
				return roomInfo.RoomId
			}
		}
	}
	return ""
}

func (r *RoomHandler) CleanupEmptyRooms() {
	fmt.Printf("Cleaning up empty rooms, current count: %d\n", r.roomManager.GetRoomCount())
}
