// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cluster

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/protos"
)

// NatsRPCServer struct
type NatsRPCServer struct {
	connString         string
	server             *Server
	conn               *nats.Conn
	pushBufferSize     int
	messagesBufferSize int
	config             *config.Config
	stopChan           chan bool
	subChan            chan *nats.Msg // subChan is the channel used by the server to receive network messages addressed to itself
	unhandledReqCh     chan *protos.Request
	userPushCh         chan *protos.Push
	sub                *nats.Subscription
}

// NewNatsRPCServer ctor
func NewNatsRPCServer(config *config.Config, server *Server) *NatsRPCServer {
	ns := &NatsRPCServer{
		config:         config,
		server:         server,
		stopChan:       make(chan bool),
		unhandledReqCh: make(chan *protos.Request),
	}
	ns.configure()
	return ns
}

func (ns *NatsRPCServer) configure() {
	ns.connString = ns.config.GetString("pitaya.cluster.rpc.server.nats.connect")
	ns.messagesBufferSize = ns.config.GetInt("pitaya.buffer.cluster.rpc.server.messages")
	ns.pushBufferSize = ns.config.GetInt("pitays.buffer.cluster.rpc.server.push")
	ns.subChan = make(chan *nats.Msg, ns.messagesBufferSize)
	// the reason this channel is buffered is that we can achieve more performance by not
	// blocking producers on a massive push
	ns.userPushCh = make(chan *protos.Push, ns.pushBufferSize)
}

// GetUserMessagesTopic get the topic for user
func GetUserMessagesTopic(uid string) string {
	return fmt.Sprintf("pitaya/user/%s/push", uid)
}

// SubscribeToUserMessages subscribes to user msg channel
func (ns *NatsRPCServer) SubscribeToUserMessages(uid string) (*nats.Subscription, error) {
	subs, err := ns.conn.Subscribe(GetUserMessagesTopic(uid), func(msg *nats.Msg) {
		push := &protos.Push{}
		err := proto.Unmarshal(msg.Data, push)
		if err != nil {
			log.Error("error unmarshalling push:", err.Error())
		}
		ns.userPushCh <- push
	})
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (ns *NatsRPCServer) handleMessages() {
	defer (func() {
		close(ns.unhandledReqCh)
		close(ns.subChan)
	})()
	for {
		select {
		case msg := <-ns.subChan:
			req := &protos.Request{}
			err := proto.Unmarshal(msg.Data, req)
			if err != nil {
				// should answer rpc with an error
				log.Error("error unmarshalling rpc message:", err.Error())
				continue
			}
			req.Msg.Reply = msg.Reply
			ns.unhandledReqCh <- req
		case <-ns.stopChan:
			return
		}
	}
}

// GetUnhandledRequestsChannel returns the channel that will receive unhandled messages
func (ns *NatsRPCServer) GetUnhandledRequestsChannel() chan *protos.Request {
	return ns.unhandledReqCh
}

// GetUserPushChannel returns the channel that will receive user pushs
func (ns *NatsRPCServer) GetUserPushChannel() chan *protos.Push {
	return ns.userPushCh
}

// Init inits nats rpc server
func (ns *NatsRPCServer) Init() error {
	go ns.handleMessages()
	conn, err := setupNatsConn(ns.connString)
	if err != nil {
		return err
	}
	ns.conn = conn
	if ns.sub, err = ns.subscribe(getChannel(ns.server.Type, ns.server.ID)); err != nil {
		return err
	}
	return nil
}

// AfterInit runs after initialization
func (ns *NatsRPCServer) AfterInit() {}

// BeforeShutdown runs before shutdown
func (ns *NatsRPCServer) BeforeShutdown() {}

// Shutdown stops nats rpc server
func (ns *NatsRPCServer) Shutdown() error {
	ns.stopChan <- true
	return nil
}

func (ns *NatsRPCServer) subscribe(topic string) (*nats.Subscription, error) {
	return ns.conn.ChanSubscribe(topic, ns.subChan)
}

func (ns *NatsRPCServer) stop() {
}

func getChannel(serverType, serverID string) string {
	return fmt.Sprintf("pitaya/servers/%s/%s", serverType, serverID)
}

func setupNatsConn(connectString string) (*nats.Conn, error) {
	nc, err := nats.Connect(connectString,
		nats.DisconnectHandler(func(_ *nats.Conn) {
			log.Warn("disconnected from nats!")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Warnf("reconnected to nats %s!", nc.ConnectedUrl)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Warnf("nats connection closed. reason: %q", nc.LastError())
		}),
	)
	if err != nil {
		return nil, err
	}
	return nc, nil
}
