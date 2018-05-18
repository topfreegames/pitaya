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
	"math"

	"github.com/gogo/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
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
	bindingsChan       chan *nats.Msg // bindingsChan receives notify from other servers on every user bind to session
	unhandledReqCh     chan *protos.Request
	userPushCh         chan *protos.Push
	sub                *nats.Subscription
	dropped            int
	metricsReporter    metrics.Reporter
}

// NewNatsRPCServer ctor
func NewNatsRPCServer(config *config.Config, server *Server, metricsReporter metrics.Reporter) (*NatsRPCServer, error) {
	ns := &NatsRPCServer{
		config:          config,
		server:          server,
		stopChan:        make(chan bool),
		unhandledReqCh:  make(chan *protos.Request),
		dropped:         0,
		metricsReporter: metricsReporter,
	}
	if err := ns.configure(); err != nil {
		return nil, err
	}

	return ns, nil
}

func (ns *NatsRPCServer) configure() error {
	ns.connString = ns.config.GetString("pitaya.cluster.rpc.server.nats.connect")
	if ns.connString == "" {
		return constants.ErrNoNatsConnectionString
	}
	ns.messagesBufferSize = ns.config.GetInt("pitaya.buffer.cluster.rpc.server.messages")
	if ns.messagesBufferSize == 0 {
		return constants.ErrNatsMessagesBufferSizeZero
	}
	ns.pushBufferSize = ns.config.GetInt("pitaya.buffer.cluster.rpc.server.push")
	if ns.pushBufferSize == 0 {
		return constants.ErrNatsPushBufferSizeZero
	}
	ns.subChan = make(chan *nats.Msg, ns.messagesBufferSize)
	ns.bindingsChan = make(chan *nats.Msg, ns.messagesBufferSize)
	// the reason this channel is buffered is that we can achieve more performance by not
	// blocking producers on a massive push
	ns.userPushCh = make(chan *protos.Push, ns.pushBufferSize)
	return nil
}

// GetBindingsChannel gets the channel that will receive all bindings
func (ns *NatsRPCServer) GetBindingsChannel() chan *nats.Msg {
	return ns.bindingsChan
}

// GetUserMessagesTopic get the topic for user
func GetUserMessagesTopic(uid string, svType string) string {
	return fmt.Sprintf("pitaya/%s/user/%s/push", svType, uid)
}

// GetBindBroadcastTopic gets the topic on which bind events will be broadcasted
func GetBindBroadcastTopic(svType string) string {
	return fmt.Sprintf("pitaya/%s/bindings", svType)
}

// SubscribeToBindingsChannel subscribes to the channel that will receive binding notifications from other servers
func (ns *NatsRPCServer) SubscribeToBindingsChannel() error {
	_, err := ns.conn.ChanSubscribe(GetBindBroadcastTopic(ns.server.Type), ns.bindingsChan)
	return err
}

// SubscribeToUserMessages subscribes to user msg channel
func (ns *NatsRPCServer) SubscribeToUserMessages(uid string, svType string) (*nats.Subscription, error) {
	subs, err := ns.conn.Subscribe(GetUserMessagesTopic(uid, svType), func(msg *nats.Msg) {
		push := &protos.Push{}
		err := proto.Unmarshal(msg.Data, push)
		if err != nil {
			logger.Log.Error("error unmarshalling push:", err.Error())
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
		ns.conn.Close()
		close(ns.unhandledReqCh)
		close(ns.subChan)
		close(ns.bindingsChan)
	})()
	maxPending := float64(0)
	for {
		select {
		case msg := <-ns.subChan:
			ns.reportMetrics()
			dropped, err := ns.sub.Dropped()
			if err != nil {
				logger.Log.Errorf("error getting number of dropped messages: %s", err.Error())
			}
			if dropped > ns.dropped {
				logger.Log.Warnf("[rpc server] some messages were dropped! numDropped: %d", dropped)
				ns.dropped = dropped
			}
			subsChanLen := float64(len(ns.subChan))
			maxPending = math.Max(float64(maxPending), subsChanLen)
			logger.Log.Debugf("subs channel size: %d, max: %d, dropped: %d", subsChanLen, maxPending, dropped)
			req := &protos.Request{}
			err = proto.Unmarshal(msg.Data, req)
			if err != nil {
				// should answer rpc with an error
				logger.Log.Error("error unmarshalling rpc message:", err.Error())
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
	// TODO should we have concurrency here? it feels like we should
	go ns.handleMessages()
	conn, err := setupNatsConn(ns.connString)
	if err != nil {
		return err
	}
	ns.conn = conn
	if ns.sub, err = ns.subscribe(getChannel(ns.server.Type, ns.server.ID)); err != nil {
		return err
	}
	return ns.SubscribeToBindingsChannel()
}

// AfterInit runs after initialization
func (ns *NatsRPCServer) AfterInit() {}

// BeforeShutdown runs before shutdown
func (ns *NatsRPCServer) BeforeShutdown() {}

// Shutdown stops nats rpc server
func (ns *NatsRPCServer) Shutdown() error {
	close(ns.stopChan)
	return nil
}

func (ns *NatsRPCServer) subscribe(topic string) (*nats.Subscription, error) {
	return ns.conn.ChanSubscribe(topic, ns.subChan)
}

func (ns *NatsRPCServer) stop() {
}

func (ns *NatsRPCServer) reportMetrics() {
	chSizeName := "buffered_channel_size"
	chQueueName := "channel_queue_size"
	if ns.metricsReporter != nil {
		if err := ns.metricsReporter.ReportCount(ns.dropped, "rpc_server_dropped_messages"); err != nil {
			logger.Log.Warnf("failed to report dropped message: %s", err.Error())
		}

		tag := "name:rpcServerSub"
		if err := ns.metricsReporter.ReportCount(len(ns.subChan), chSizeName, tag); err != nil {
			logger.Log.Warnf("failed to report subChan size: %s", err.Error())
		}
		queueSize := len(ns.subChan) - ns.messagesBufferSize
		if queueSize >= 0 {
			logger.Log.Warnf("subChan is at maximum capacity, waiting msgs: %d", queueSize)
			if err := ns.metricsReporter.ReportCount(queueSize, chQueueName, tag); err != nil {
				logger.Log.Warnf("failed to report subChan queue size: %s", err.Error())
			}
		}

		tag = "name:rpcServerBindings"
		if err := ns.metricsReporter.ReportCount(len(ns.bindingsChan), chSizeName, tag); err != nil {
			logger.Log.Warnf("failed to report bindingsChan size: %s", err.Error())
		}
		queueSize = len(ns.bindingsChan) - ns.messagesBufferSize
		if queueSize >= 0 {
			logger.Log.Warnf("bindingsChan is at maximum capacity, waiting msgs: %d", queueSize)
			if err := ns.metricsReporter.ReportCount(queueSize, chQueueName, tag); err != nil {
				logger.Log.Warnf("failed to report bindingsChan queue size: %s", err.Error())
			}
		}

		tag = "name:rpcServerUserPush"
		if err := ns.metricsReporter.ReportCount(len(ns.userPushCh), chSizeName, tag); err != nil {
			logger.Log.Warnf("failed to report userPushCh size: %s", err.Error())
		}
		queueSize = len(ns.userPushCh) - ns.pushBufferSize
		if queueSize >= 0 {
			logger.Log.Warnf("userPushCh is at maximum capacity, waiting msgs: %d", queueSize)
			if err := ns.metricsReporter.ReportCount(queueSize, chQueueName, tag); err != nil {
				logger.Log.Warnf("failed to report userPushCh queue size: %s", err.Error())
			}
		}
	}
}
