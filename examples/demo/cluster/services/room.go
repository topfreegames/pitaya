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

	// RPCMessage represents a rpc message
	RPCMessage struct {
		ServerID string `json:"serverID"`
		Data     string `json:"data"`
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

// Init runs on srevice initialization
func (r *Room) Init() {
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
func (r *Room) Entry(s *session.Session, msg []byte) error {
	fakeUID := uuid.New().String() //just use s.ID as uid !!!
	err := s.Bind(fakeUID)         // binding session uid
	if err != nil {
		return err
	}
	return s.Response(&JoinResponse{Result: "ok"})
}

// GetSessionData gets the session data
func (r *Room) GetSessionData(s *session.Session, msg []byte) error {
	return s.Response(s.GetData())
}

// SetSessionData sets the session data
func (r *Room) SetSessionData(s *session.Session, data *SessionData) error {
	err := s.SetData(data.Data)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	err = s.PushToFront()
	return s.Response("success")
}

// Join room
func (r *Room) Join(s *session.Session, msg []byte) error {
	s.Push("onMembers", &AllMembers{Members: r.group.Members()})
	r.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	r.group.Add(s)
	s.OnClose(func() {
		r.group.Leave(s)
	})
	return s.Response(&JoinResponse{Result: "success"})
}

// Message sync last message to all members
func (r *Room) Message(s *session.Session, msg *UserMessage) error {
	return r.group.Broadcast("onMessage", msg)
}

// SendRPC sends rpc
func (r *Room) SendRPC(s *session.Session, msg *RPCMessage) error {
	mmsg := &UserMessage{
		Name:    "funciona",
		Content: "vai funcionar",
	}
	b := true
	str := "AE PQP"
	res, err := pitaya.RPC("room.room.messageremote", &UserMessage{}, mmsg, b, str)
	if err != nil {
		fmt.Printf("rpc error: %s\n", err)
		return err
	}
	fmt.Printf("rpc res %s\n", res)
	return nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(msg *UserMessage, b bool, s string) (*UserMessage, error) {
	fmt.Println("CHEGOU", b, s)
	return msg, nil
}
