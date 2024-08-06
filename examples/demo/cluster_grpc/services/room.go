package services

import (
	"context"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"

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

// Outbound gets the outbound status
func (Stats *Stats) Outbound(ctx context.Context, in []byte) ([]byte, error) {
	Stats.outboundBytes += len(in)
	return in, nil
}

// Inbound gets the inbound status
func (Stats *Stats) Inbound(ctx context.Context, in []byte) ([]byte, error) {
	Stats.inboundBytes += len(in)
	return in, nil
}

// NewRoom returns a new room
func NewRoom(app pitaya.Pitaya) *Room {
	return &Room{
		app:   app,
		Stats: &Stats{},
	}
}

// Init runs on service initialization
func (r *Room) Init() {
	r.app.GroupCreate(context.Background(), "room")
	// It is necessary to register all structs that will be used in RPC calls
	// This must be done both in the caller and callee servers
	gob.Register(&UserMessage{})
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := r.app.GroupCountMembers(context.Background(), "room")
		println("UserCount: Time=>", time.Now().String(), "Count=>", count, "Error=>", err)
		println("OutboundBytes", r.Stats.outboundBytes)
		println("InboundBytes", r.Stats.outboundBytes)
	})
}

// Entry is the entrypoint
func (r *Room) Entry(ctx context.Context, msg []byte) (*JoinResponse, error) {
	s := r.app.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, strconv.Itoa(int(s.ID())))
	if err != nil {
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	return &JoinResponse{Result: "ok"}, nil
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
	s := r.app.GetSessionFromCtx(ctx)
	err := s.SetData(data.Data)
	if err != nil {
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
	s := r.app.GetSessionFromCtx(ctx)
	err := r.app.GroupAddMember(ctx, "room", s.UID())
	if err != nil {
		return nil, err
	}
	members, err := r.app.GroupMembers(ctx, "room")
	if err != nil {
		return nil, err
	}
	s.Push("onMembers", &AllMembers{Members: members})
	r.app.GroupBroadcast(ctx, "connector", "room", "onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	err = s.OnClose(func() {
		r.app.GroupRemoveMember(ctx, "room", s.UID())
	})
	if err != nil {
		return nil, err
	}
	return &JoinResponse{Result: "success"}, nil
}

// Message sync last message to all members
func (r *Room) Message(ctx context.Context, msg *UserMessage) {
	err := r.app.GroupBroadcast(ctx, "connector", "room", "onMessage", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(ctx context.Context, msg *cluster_protos.RPCMsg) (*cluster_protos.RPCRes, error) {
	ret := &cluster_protos.RPCRes{}
	err := r.app.RPC(ctx, "connector.connectorremote.remotefunc", ret, msg)
	if err != nil {
		return nil, pitaya.Error(err, "RPC-000")
	}
	return ret, nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(ctx context.Context, msg *UserMessage, b bool, s string) (*UserMessage, error) {
	return msg, nil
}
