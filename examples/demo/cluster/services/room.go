package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya/v3/examples/demo/cluster_protos"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/timer"
)

type (
	// Room represents a component that contains a bundle of room related handler
	// like Join/Message
	Room struct {
		component.Base
		timer *timer.Timer
		app   pitaya.Pitaya
		Stats *cluster_protos.Stats
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
func NewRoom(app pitaya.Pitaya) *Room {
	return &Room{
		app:   app,
		Stats: &cluster_protos.Stats{},
	}
}

// Init runs on service initialization
func (r *Room) Init() {
	r.app.GroupCreate(context.Background(), "room")
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := r.app.GroupCountMembers(context.Background(), "room")
		println("UserCount: Time=>", time.Now().String(), "Count=>", count, "Error=>", err)
		println("OutboundBytes", r.Stats.OutboundBytes)
		println("InboundBytes", r.Stats.OutboundBytes)
	})
}

// Entry is the entrypoint
func (r *Room) Entry(ctx context.Context, msg []byte) (*cluster_protos.JoinResponse, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx) // The default logger contains a requestId, the route being executed and the sessionId
	s := r.app.GetSessionFromCtx(ctx)

	err := s.Bind(ctx, uuid.New().String())
	if err != nil {
		logger.Error("Failed to bind session")
		logger.Error(err)
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	return &cluster_protos.JoinResponse{Result: "ok"}, nil
}

// GetSessionData gets the session data
func (r *Room) GetSessionData(ctx context.Context) (*SessionData, error) {
	s := r.app.GetSessionFromCtx(ctx)
	return &SessionData{
		Data: s.GetData(),
	}, nil
}

// SetSessionData sets the session data
func (r *Room) SetSessionData(ctx context.Context, data *SessionData) ([]byte, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	s := r.app.GetSessionFromCtx(ctx)
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

// Notify push is a notify route that triggers a push to a session
func (r *Room) NotifyPush(ctx context.Context) {
	s := r.app.GetSessionFromCtx(ctx)
	r.app.SendPushToUsers("testPush", &cluster_protos.RPCMsg{Msg: "test"}, []string{s.UID()}, "connector")
}

// Join room
func (r *Room) Join(ctx context.Context) (*cluster_protos.JoinResponse, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	s := r.app.GetSessionFromCtx(ctx)
	err := r.app.GroupAddMember(ctx, "room", s.UID())
	if err != nil {
		logger.Error("Failed to join room")
		logger.Error(err)
		return nil, err
	}
	members, err := r.app.GroupMembers(ctx, "room")
	if err != nil {
		logger.Error("Failed to get members")
		logger.Error(err)
		return nil, err
	}
	s.Push("onMembers", &cluster_protos.AllMembers{Members: members})
	err = r.app.GroupBroadcast(ctx, "connector", "room", "onNewUser", &cluster_protos.NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	if err != nil {
		logger.Error("Failed to broadcast onNewUser")
		logger.Error(err)
		return nil, err
	}
	return &cluster_protos.JoinResponse{Result: "success"}, nil
}

// Leave room
func (r *Room) Leave(ctx context.Context) ([]byte, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	s := r.app.GetSessionFromCtx(ctx)
	err := r.app.GroupRemoveMember(ctx, "room", s.UID())
	if err != nil {
		logger.Error(err)
		return []byte("failed"), err
	}
	return []byte("success"), err
}

// Message sync last message to all members
func (r *Room) Message(ctx context.Context, msg *cluster_protos.UserMessage) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	err := r.app.GroupBroadcast(ctx, "connector", "room", "onMessage", msg)
	if err != nil {
		logger.Error("Error broadcasting message")
		logger.Error(err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(ctx context.Context, msg *cluster_protos.SendRPCMsg) (*cluster_protos.RPCRes, error) {
	logger := pitaya.GetDefaultLoggerFromCtx(ctx)
	ret := &cluster_protos.RPCRes{}
	err := r.app.RPCTo(ctx, msg.ServerId, msg.Route, ret, &cluster_protos.RPCMsg{Msg: msg.Msg})
	if err != nil {
		logger.Errorf("Failed to execute RPCTo %s - %s", msg.ServerId, msg.Route)
		logger.Error(err)
		return nil, pitaya.Error(err, "RPC-000")
	}
	return ret, nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(ctx context.Context, msg *cluster_protos.UserMessage, b bool, s string) (*cluster_protos.UserMessage, error) {
	return msg, nil
}
