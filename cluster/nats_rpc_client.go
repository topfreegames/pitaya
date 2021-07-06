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
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/tracing"
)

// NatsRPCClient struct
type NatsRPCClient struct {
	config                 *config.Config
	conn                   *nats.Conn
	connString             string
	connectionTimeout      time.Duration
	maxReconnectionRetries int
	reqTimeout             time.Duration
	running                bool
	server                 *Server
	metricsReporters       []metrics.Reporter
	appDieChan             chan bool
}

// NewNatsRPCClient ctor
func NewNatsRPCClient(
	config *config.Config,
	server *Server,
	metricsReporters []metrics.Reporter,
	appDieChan chan bool,
) (*NatsRPCClient, error) {
	ns := &NatsRPCClient{
		config:            config,
		server:            server,
		running:           false,
		metricsReporters:  metricsReporters,
		appDieChan:        appDieChan,
		connectionTimeout: nats.DefaultTimeout,
	}
	if err := ns.configure(); err != nil {
		return nil, err
	}
	return ns, nil
}

func (ns *NatsRPCClient) configure() error {
	ns.connString = ns.config.GetString("pitaya.cluster.rpc.client.nats.connect")
	if ns.connString == "" {
		return constants.ErrNoNatsConnectionString
	}
	ns.connectionTimeout = ns.config.GetDuration("pitaya.cluster.rpc.client.nats.connectiontimeout")
	ns.maxReconnectionRetries = ns.config.GetInt("pitaya.cluster.rpc.client.nats.maxreconnectionretries")
	ns.reqTimeout = ns.config.GetDuration("pitaya.cluster.rpc.client.nats.requesttimeout")
	if ns.reqTimeout == 0 {
		return constants.ErrNatsNoRequestTimeout
	}
	return nil
}

// BroadcastSessionBind sends the binding information to other servers that may be interested in this info
func (ns *NatsRPCClient) BroadcastSessionBind(uid string) error {
	msg := &protos.BindMsg{
		Uid: uid,
		Fid: ns.server.ID,
	}
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return ns.Send(GetBindBroadcastTopic(ns.server.Type), msgData)
}

// Send publishes a message in a given topic
func (ns *NatsRPCClient) Send(topic string, data []byte) error {
	if !ns.running {
		return constants.ErrRPCClientNotInitialized
	}
	return ns.conn.Publish(topic, data)
}

// SendPush sends a message to a user
func (ns *NatsRPCClient) SendPush(userID string, frontendSv *Server, push *protos.Push) error {
	topic := GetUserMessagesTopic(userID, frontendSv.Type)
	msg, err := proto.Marshal(push)
	if err != nil {
		return err
	}
	return ns.Send(topic, msg)
}

// SendKick kicks an user
func (ns *NatsRPCClient) SendKick(userID string, serverType string, kick *protos.KickMsg) error {
	topic := GetUserKickTopic(userID, serverType)
	msg, err := proto.Marshal(kick)
	if err != nil {
		return err
	}
	return ns.Send(topic, msg)
}

// Call calls a method remotelly
func (ns *NatsRPCClient) Call(
	ctx context.Context,
	rpcType protos.RPCType,
	route *route.Route,
	session *session.Session,
	msg *message.Message,
	server *Server,
) (*protos.Response, error) {
	parent, err := tracing.ExtractSpan(ctx)
	if err != nil {
		logger.Log.Warnf("failed to retrieve parent span: %s", err.Error())
	}
	tags := opentracing.Tags{
		"span.kind":       "client",
		"local.id":        ns.server.ID,
		"peer.serverType": server.Type,
		"peer.id":         server.ID,
	}
	ctx = tracing.StartSpan(ctx, "NATS RPC Call", tags, parent)
	defer tracing.FinishSpan(ctx, err)

	if !ns.running {
		err = constants.ErrRPCClientNotInitialized
		return nil, err
	}
	req, err := buildRequest(ctx, rpcType, route, session, msg, ns.server)
	if err != nil {
		return nil, err
	}
	marshalledData, err := proto.Marshal(&req)
	if err != nil {
		return nil, err
	}

	var m *nats.Msg

	if ns.metricsReporters != nil {
		startTime := time.Now()
		ctx = pcontext.AddToPropagateCtx(ctx, constants.StartTimeKey, startTime.UnixNano())
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, route.String())
		defer func() {
			typ := "rpc"
			metrics.ReportTimingFromCtx(ctx, ns.metricsReporters, typ, err)
		}()
	}
	m, err = ns.conn.Request(getChannel(server.Type, server.ID), marshalledData, ns.reqTimeout)
	if err != nil {
		return nil, err
	}

	res := &protos.Response{}
	err = proto.Unmarshal(m.Data, res)
	if err != nil {
		return nil, err
	}

	if res.Error != nil {
		if res.Error.Code == "" {
			res.Error.Code = errors.ErrUnknownCode
		}
		err = &errors.Error{
			Code:     res.Error.Code,
			Message:  res.Error.Msg,
			Metadata: res.Error.Metadata,
		}
		return nil, err
	}
	return res, nil
}

// Init inits nats rpc client
func (ns *NatsRPCClient) Init() error {
	ns.running = true
	logger.Log.Debugf("connecting to nats (client) with timeout of %s", ns.connectionTimeout)
	conn, err := setupNatsConn(
		ns.connString,
		ns.appDieChan,
		nats.MaxReconnects(ns.maxReconnectionRetries),
		nats.Timeout(ns.connectionTimeout),
	)
	if err != nil {
		return err
	}
	ns.conn = conn
	return nil
}

// AfterInit runs after initialization
func (ns *NatsRPCClient) AfterInit() {}

// BeforeShutdown runs before shutdown
func (ns *NatsRPCClient) BeforeShutdown() {}

// Shutdown stops nats rpc server
func (ns *NatsRPCClient) Shutdown() error {
	return nil
}

func (ns *NatsRPCClient) stop() {
	ns.running = false
}

func (ns *NatsRPCClient) getSubscribeChannel() string {
	return fmt.Sprintf("pitaya/servers/%s/%s", ns.server.Type, ns.server.ID)
}
