package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"strings"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/chat/protos"
	"github.com/topfreegames/pitaya/serialize/protobuf"
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
		stats *stats
	}

	stats struct {
		outboundBytes int
		inboundBytes  int
	}
)

func (stats *stats) outbound(s *session.Session, in []byte) ([]byte, error) {
	stats.outboundBytes += len(in)
	return in, nil
}

func (stats *stats) inbound(s *session.Session, in []byte) ([]byte, error) {
	stats.inboundBytes += len(in)
	return in, nil
}

// NewRoom returns a new room
func NewRoom() *Room {
	return &Room{
		group: pitaya.NewGroup("room"),
		stats: &stats{},
	}
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		println("UserCount: Time=>", time.Now().String(), "Count=>", r.group.Count())
		println("OutboundBytes", r.stats.outboundBytes)
		println("InboundBytes", r.stats.outboundBytes)
	})
}

// Join room
func (r *Room) Join(s *session.Session, msg []byte) *protos.JoinResponse {
	res := &protos.JoinResponse{}
	fakeUID := s.ID()                         //just use s.ID as uid !!!
	err := s.Bind(strconv.Itoa(int(fakeUID))) // binding session uid

	if err != nil {
		res.Result = err.Error()
		return res
	}

	s.Push("onMembers", &protos.AllMembers{Members: r.group.Members()})
	// notify others
	r.group.Broadcast("onNewUser", &protos.NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	// new user join group
	r.group.Add(s) // add session to group

	// on session close, remove it from group
	s.OnClose(func() {
		r.group.Leave(s)
	})

	res.Result = "success"
	return res
}

// Message sync last message to all members
func (r *Room) Message(s *session.Session, msg *protos.UserMessage) {
	err := r.group.Broadcast("onMessage", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

func main() {
	defer (func() {
		pitaya.Shutdown()
	})()

	protos, err := os.Open("./protos/chat.proto")
	if err != nil {
		panic(err)
	}

	protosMapping, err := os.Open("./protos/mapping.json")
	if err != nil {
		panic(err)
	}

	s, err := protobuf.NewSerializer(protos, protosMapping)

	if err != nil {
		panic(err)
	}

	pitaya.SetSerializer(s)

	// rewrite component and handler name
	room := NewRoom()
	pitaya.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	// traffic stats
	pitaya.AfterHandler(room.stats.outbound)
	pitaya.BeforeHandler(room.stats.inbound)

	log.SetFlags(log.LstdFlags | log.Llongfile)

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	//TODO need to fix that? pitaya.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	ws := acceptor.NewWSAcceptor(":3250", "/pitaya")
	pitaya.AddAcceptor(ws)
	pitaya.Configure(true, "chat", pitaya.Standalone)
	pitaya.Start()
}
