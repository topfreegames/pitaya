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

type (
	// Room represents a component that contains a bundle of room related handler
	// like Join/Message
	Room struct {
		component.Base
		timer *timer.Timer
		Stats *Stats
	}

	// UserMessage represents a message that user sent
	UserMessage struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}

	// Stats exports the room status
	Stats struct {
		outboundBytes int
		inboundBytes  int
	}

	// RPCResponse represents a rpc message
	RPCResponse struct {
		Msg string `json:"msg"`
	}

	// SendRPCMsg represents a rpc message
	SendRPCMsg struct {
		ServerID string `json:"serverId"`
		Route    string `json:"route"`
		Msg      string `json:"msg"`
	}

	// NewUser message will be received when new user join room
	NewUser struct {
		Content string `json:"content"`
	}

	// AllMembers contains all members uid
	AllMembers struct {
		Members []string `json:"members"`
	}

	// JoinResponse represents the result of joining room
	JoinResponse struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
)

// NewRoom returns a new room
func NewRoom() *Room {
	return &Room{
		Stats: &Stats{},
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
		println("OutboundBytes", r.Stats.outboundBytes)
		println("InboundBytes", r.Stats.outboundBytes)
	})
}

// Entry is the entrypoint
func (r *Room) Entry(ctx context.Context, msg []byte) (*JoinResponse, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx) // The default logger contains a requestId, the route being executed and the sessionId
	s := pitaya.GetSessionFromCtx(ctx)

	err := s.Bind(ctx, "helroow")
	if err != nil {
		logger.Error("Failed to bind session")
		logger.Error(err)
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	return &JoinResponse{Result: "ok"}, nil
}

// GetSessionData gets the session data
func (r *Room) GetSessionData(ctx context.Context) (*SessionData, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	return &SessionData{
		Data: s.GetData(),
	}, nil
}

// SetSessionData sets the session data
func (r *Room) SetSessionData(ctx context.Context, data *SessionData) ([]byte, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	s := pitaya.GetSessionFromCtx(ctx)
	err := s.SetData(data.Data)
	if err != nil {
		logger.Error("Failed to set session data")
		logger.Error(err)
		return nil, err
	}
	err = s.PushToFront(ctx)
	if err != nil {
		return nil, err
	}
	return []byte("success"), nil
}

// Join room
func (r *Room) Join(ctx context.Context) (*JoinResponse, error) {
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
	s.Push("onMembers", &AllMembers{Members: members})
	err = pitaya.GroupBroadcast(ctx, "connector", "room", "onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	if err != nil {
		logger.Error("Failed to broadcast onNewUser")
		logger.Error(err)
		return nil, err
	}
	return &JoinResponse{Result: "success"}, nil
}

// Message sync last message to all members
func (r *Room) Message(ctx context.Context, msg *UserMessage) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	err := pitaya.GroupBroadcast(ctx, "connector", "room", "onMessage", msg)
	if err != nil {
		logger.Error("Error broadcasting message")
		logger.Error(err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(ctx context.Context, msg *SendRPCMsg) (*protos.RPCRes, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	ret := &protos.RPCRes{}
	err := pitaya.RPCTo(ctx, msg.ServerID, msg.Route, ret, &protos.RPCMsg{Msg: msg.Msg})
	if err != nil {
		logger.Errorf("Failed to execute RPCTo %s - %s", msg.ServerID, msg.Route)
		logger.Error(err)
		return nil, pitaya.Error(err, "RPC-000")
	}
	return ret, nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(ctx context.Context, msg *UserMessage, b bool, s string) (*UserMessage, error) {
	return msg, nil
}
