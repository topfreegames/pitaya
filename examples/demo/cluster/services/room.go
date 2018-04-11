package services

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
)

type (
	// Room represents a component that contains a bundle of room related handler
	// like Join/Message
	Room struct {
		component.Base
		group *pitaya.Group
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

// Outbound gets the outbound status
func (Stats *Stats) Outbound(s *session.Session, in []byte) ([]byte, error) {
	Stats.outboundBytes += len(in)
	return in, nil
}

// Inbound gets the inbound status
func (Stats *Stats) Inbound(s *session.Session, in []byte) ([]byte, error) {
	Stats.inboundBytes += len(in)
	return in, nil
}

// NewRoom returns a new room
func NewRoom() *Room {
	return &Room{
		group: pitaya.NewGroup("room"),
		Stats: &Stats{},
	}
}

// Init runs on service initialization
func (r *Room) Init() {
	// It is necessary to register all structs that will be used in RPC calls
	// This must be done both in the caller and callee servers
	gob.Register(&UserMessage{})
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		println("UserCount: Time=>", time.Now().String(), "Count=>", r.group.Count())
		println("OutboundBytes", r.Stats.outboundBytes)
		println("InboundBytes", r.Stats.outboundBytes)
	})
}

// Entry is the entrypoint
func (r *Room) Entry(s *session.Session, msg []byte) (*JoinResponse, error) {
	fakeUID := uuid.New().String() // just use s.ID as uid !!!
	err := s.Bind(fakeUID)         // binding session uid
	if err != nil {
		return nil, err
	}
	return &JoinResponse{Result: "ok"}, nil
}

// GetSessionData gets the session data
func (r *Room) GetSessionData(s *session.Session) (*SessionData, error) {
	return &SessionData{
		Data: s.GetData(),
	}, nil
}

// SetSessionData sets the session data
func (r *Room) SetSessionData(s *session.Session, data *SessionData) ([]byte, error) {
	err := s.SetData(data.Data)
	if err != nil {
		return nil, err
	}
	err = s.PushToFront()
	if err != nil {
		return nil, err
	}
	return []byte("success"), nil
}

// Join room
func (r *Room) Join(s *session.Session) (*JoinResponse, error) {
	err := r.group.Add(s)
	if err != nil {
		return nil, err
	}
	s.Push("onMembers", &AllMembers{Members: r.group.Members()})
	r.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	err = s.OnClose(func() {
		r.group.Leave(s)
	})
	if err != nil {
		return nil, err
	}
	return &JoinResponse{Result: "success"}, nil
}

// Message sync last message to all members
func (r *Room) Message(s *session.Session, msg *UserMessage) {
	err := r.group.Broadcast("onMessage", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(s *session.Session, msg *SendRPCMsg) *RPCResponse {
	ret := &RPCResponse{}
	err := pitaya.RPCTo(msg.ServerID, msg.Route, ret, msg.Msg)
	if err != nil {
		ret.Msg = err.Error()
		return ret
	}
	fmt.Printf("rpc ret: %s\n", ret)
	return ret
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(msg *UserMessage, b bool, s string) (*UserMessage, error) {
	return msg, nil
}
