package services

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/cluster_protobuf/protos"
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

	// Stats exports the room status
	Stats struct {
		outboundBytes int
		inboundBytes  int
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
	gob.Register(&protos.UserMessage{})
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		println("UserCount: Time=>", time.Now().String(), "Count=>", r.group.Count())
		println("OutboundBytes", r.Stats.outboundBytes)
		println("InboundBytes", r.Stats.outboundBytes)
	})
}

func reply(code int32, msg string) *protos.Response {
	return &protos.Response{
		Code: code,
		Msg:  msg,
	}
}

// Entry is the entrypoint
func (r *Room) Entry(s *session.Session) *protos.Response {
	fakeUID := uuid.New().String() // just use s.ID as uid !!!
	err := s.Bind(fakeUID)         // binding session uid
	if err != nil {
		return reply(500, err.Error())
	}
	return reply(200, "ok")
}

// Join room
func (r *Room) Join(s *session.Session) *protos.Response {
	s.Push("onMembers", &protos.AllMembers{Members: r.group.Members()})
	r.group.Broadcast("onNewUser", &protos.NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	r.group.Add(s)
	s.OnClose(func() {
		r.group.Leave(s)
	})
	return &protos.Response{Msg: "success"}
}

// Message sync last message to all members
func (r *Room) Message(s *session.Session, msg *protos.UserMessage) {
	err := r.group.Broadcast("onMessage", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

// SendRPC sends rpc
func (r *Room) SendRPC(s *session.Session, msg []byte) *protos.Response {
	ret := protos.Response{}
	err := pitaya.RPC("connector.connectorremote.remotefunc", &ret, msg)
	if err != nil {
		return reply(500, err.Error())
	}
	return reply(200, ret.Msg)
}
