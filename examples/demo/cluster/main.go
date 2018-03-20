package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"strings"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/serialize/json"
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

	// UserMessage represents a message that user sent
	UserMessage struct {
		Name    string `json:"name"`
		Content string `json:"content"`
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
func (r *Room) Join(s *session.Session, msg []byte) error {
	var fakeUID string
	if s.UID() == "" {
		fakeUID := uuid.New().String() //just use s.ID as uid !!!
		s.Bind(fakeUID)                // binding session uid
	}

	s.Push("onMembers", &AllMembers{Members: r.group.Members()})
	// notify others
	r.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %s", fakeUID)})
	// new user join group
	r.group.Add(s) // add session to group

	// on session close, remove it from group
	s.OnClose(func() {
		r.group.Leave(s)
	})
	return s.Response(&JoinResponse{Result: "success"})
}

// Message sync last message to all members
func (r *Room) Message(s *session.Session, msg *UserMessage) error {
	return r.group.Broadcast("onMessage", msg)
}

// Test1 test
func (r *Room) Test1(s *session.Session, data []byte) error {
	go func(s *session.Session) {
		time.Sleep(time.Duration(5) * time.Second)
		s.Response("okkay1")
	}(s)
	return nil
}

// Test2 test
func (r *Room) Test2(s *session.Session, data []byte) error {
	return s.Response("okkay2")
}

// SendRPC sends rpc
func (r *Room) SendRPC(s *session.Session, msg *RPCMessage) error {
	mmsg := &UserMessage{
		Name:    "funciona",
		Content: "please",
	}
	b := false
	str := "AE PQP"
	gob.Register(UserMessage{})
	res, err := pitaya.RPC("room.room.messageremote", &UserMessage{}, mmsg, b, str)
	if err != nil {
		fmt.Printf("rpc error: %s", err)
		return err
	}
	fmt.Printf("rpc res %s", res)
	return nil
}

// MessageRemote just echoes the given message
func (r *Room) MessageRemote(msg *UserMessage, b bool, s string) (*UserMessage, error) {
	fmt.Println("CHEGOU", b, s)
	if b {
		return nil, fmt.Errorf("FUCK")
	}
	return msg, nil
}

// StatsRemote returns a room stats
func (r *Room) StatsRemote() (*stats, error) {
	return r.stats, nil
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "game", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	defer (func() {
		pitaya.Shutdown()
	})()

	pitaya.SetSerializer(json.NewSerializer())
	pitaya.SetServerType(*svType)

	// rewrite component and handler name
	room := NewRoom()
	pitaya.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.RegisterRemote(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	// traffic stats
	pitaya.AfterHandler(room.stats.outbound)
	pitaya.BeforeHandler(room.stats.inbound)

	log.SetFlags(log.LstdFlags | log.Llongfile)

	//TODO fix pitaya.SetWSPath("/pitaya")
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	//TODO need to fix that? pitaya.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	ws := acceptor.NewWSAcceptor(fmt.Sprintf(":%d", *port), "/pitaya")
	pitaya.AddAcceptor(ws)
	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster)
	pitaya.Start()
}
