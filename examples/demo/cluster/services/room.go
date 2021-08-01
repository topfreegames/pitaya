package services

import (
	"context"
	"fmt"
	"time"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/examples/demo/protos"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/timer"
)

// Room represents a component that contains a bundle of room related handler
// like Join/Message
type Room struct {
	component.Base
	timer *timer.Timer
	Stats *protos.Stats
}

// NewRoom returns a new room
func NewRoom() *Room {
	return &Room{
		Stats: &protos.Stats{},
	}
}

// Init runs on service initialization
func (r *Room) Init() {
	gsi := groups.NewMemoryGroupService(config.NewConfig())
	pitaya.InitGroups(gsi)
	pitaya.GroupCreate(context.Background(), "room")
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), "room")
		println("UserCount: Time=>", time.Now().String(), "Count=>", count, "Error=>", err)
		println("OutboundBytes", r.Stats.OutboundBytes)
		println("InboundBytes", r.Stats.OutboundBytes)
	})
}

// Entry is the entrypoint
func (r *Room) Entry(ctx context.Context, msg []byte) (*protos.JoinResponse, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx) // The default logger contains a requestId, the route being executed and the sessionId
	s := pitaya.GetSessionFromCtx(ctx)

	err := s.Bind(ctx, "banana")
	if err != nil {
		logger.Error("Failed to bind session")
		logger.Error(err)
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	return &protos.JoinResponse{Result: "ok"}, nil
}

// Join room
func (r *Room) Join(ctx context.Context) (*protos.JoinResponse, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	s := pitaya.GetSessionFromCtx(ctx)
	err := pitaya.GroupAddMember(ctx, "room", s.UID())
	if err != nil {
		logger.Error("Failed to join room")
		logger.Error(err)
		return nil, err
	}
	members, err := pitaya.GroupMembers(ctx, "room")
	if err != nil {
		logger.Error("Failed to get members")
		logger.Error(err)
		return nil, err
	}
	s.Push("onMembers", &protos.AllMembers{Members: members})
	err = pitaya.GroupBroadcast(ctx, "connector", "room", "onNewUser", &protos.NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	if err != nil {
		logger.Error("Failed to broadcast onNewUser")
		logger.Error(err)
		return nil, err
	}
	return &protos.JoinResponse{Result: "success"}, nil
}

// Message sync last message to all members
func (r *Room) Message(ctx context.Context, msg *protos.UserMessage) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	err := pitaya.GroupBroadcast(ctx, "connector", "room", "onMessage", msg)
	if err != nil {
		logger.Error("Error broadcasting message")
		logger.Error(err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(ctx context.Context, msg *protos.SendRPCMsg) (*protos.RPCRes, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	ret := &protos.RPCRes{}
	err := pitaya.RPCTo(ctx, msg.ServerId, msg.Route, ret, &protos.RPCMsg{Msg: msg.Msg})
	if err != nil {
		logger.Errorf("Failed to execute RPCTo %s - %s", msg.ServerId, msg.Route)
		logger.Error(err)
		return nil, pitaya.Error(err, "RPC-000")
	}
	return ret, nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(ctx context.Context, msg *protos.UserMessage, b bool, s string) (*protos.UserMessage, error) {
	return msg, nil
}
