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
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	pitErrors "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/interfaces"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/tracing"
	"google.golang.org/grpc"
)

// GRPCClient rpc server struct
type GRPCClient struct {
	server           *Server
	config           *config.Config
	metricsReporters []metrics.Reporter
	clientMap        sync.Map
	bindingStorage   interfaces.BindingStorage
	reqTimeout       time.Duration
	dialTimeout      time.Duration
}

// NewGRPCClient returns a new instance of GRPCClient
func NewGRPCClient(config *config.Config, server *Server, metricsReporters []metrics.Reporter, bindingStorage ...interfaces.BindingStorage) (*GRPCClient, error) {
	var bs interfaces.BindingStorage
	if len(bindingStorage) > 0 {
		bs = bindingStorage[0]
	}
	gs := &GRPCClient{
		config:           config,
		server:           server,
		metricsReporters: metricsReporters,
		bindingStorage:   bs,
	}

	gs.configure()

	return gs, nil
}

// Init inits grpc rpc client
func (gs *GRPCClient) Init() error {
	return nil
}

func (gs *GRPCClient) configure() {
	gs.reqTimeout = gs.config.GetDuration("pitaya.cluster.rpc.client.grpc.requesttimeout")
	gs.dialTimeout = gs.config.GetDuration("pitaya.cluster.rpc.client.grpc.dialtimeout")
}

// Call makes a RPC Call
func (gs *GRPCClient) Call(ctx context.Context, rpcType protos.RPCType, route *route.Route, session *session.Session, msg *message.Message, server *Server) (*protos.Response, error) {
	parent, err := tracing.ExtractSpan(ctx)
	if err != nil {
		logger.Log.Warnf("failed to retrieve parent span: %s", err.Error())
	}
	tags := opentracing.Tags{
		"span.kind":       "client",
		"local.id":        gs.server.ID,
		"peer.serverType": server.Type,
		"peer.id":         server.ID,
	}
	ctx = tracing.StartSpan(ctx, "RPC Call", tags, parent)
	defer tracing.FinishSpan(ctx, err)

	req, err := buildRequest(ctx, rpcType, route, session, msg, gs.server)
	if err != nil {
		return nil, err
	}
	if c, ok := gs.clientMap.Load(server.ID); ok {
		ctxT, done := context.WithTimeout(ctx, gs.reqTimeout)
		defer done()
		res, err := c.(protos.PitayaClient).Call(ctxT, &req)
		if err != nil {
			return nil, err
		}
		if res.Error != nil {
			if res.Error.Code == "" {
				res.Error.Code = pitErrors.ErrUnknownCode
			}
			e := &pitErrors.Error{
				Code:     res.Error.Code,
				Message:  res.Error.Msg,
				Metadata: res.Error.Metadata,
			}
			return nil, e
		}
		return res, nil

	}
	return nil, constants.ErrNoConnectionToServer
}

// Send not implemented in grpc client
func (gs *GRPCClient) Send(uid string, d []byte) error {
	return constants.ErrNotImplemented
}

// BroadcastSessionBind sends the binding information to other servers that may be interested in this info
func (gs *GRPCClient) BroadcastSessionBind(uid string) error {
	if gs.bindingStorage == nil {
		return constants.ErrNoBindingStorageModule
	}
	fid, _ := gs.bindingStorage.GetUserFrontendID(uid, gs.server.Type)
	if fid != "" {
		if c, ok := gs.clientMap.Load(fid); ok {
			msg := &protos.BindMsg{
				Uid: uid,
				Fid: gs.server.ID,
			}
			ctxT, done := context.WithTimeout(context.Background(), gs.reqTimeout)
			defer done()
			_, err := c.(protos.PitayaClient).SessionBindRemote(ctxT, msg)
			return err
		}
	}
	return nil
}

// SendKick sends a kick to an user
func (gs *GRPCClient) SendKick(userID string, serverType string, kick *protos.KickMsg) error {
	var svID string
	var err error

	if gs.bindingStorage == nil {
		return constants.ErrNoBindingStorageModule
	}

	svID, err = gs.bindingStorage.GetUserFrontendID(userID, serverType)
	if err != nil {
		return err
	}

	if c, ok := gs.clientMap.Load(svID); ok {
		ctxT, done := context.WithTimeout(context.Background(), gs.reqTimeout)
		defer done()
		_, err := c.(protos.PitayaClient).KickUser(ctxT, kick)
		return err
	}
	return constants.ErrNoConnectionToServer
}

// SendPush sends a message to an user, if you dont know the serverID that the user is connected to, you need to set a BindingStorage when creating the client
// TODO: Jaeger?
func (gs *GRPCClient) SendPush(userID string, frontendSv *Server, push *protos.Push) error {
	var svID string
	var err error
	if frontendSv.ID != "" {
		svID = frontendSv.ID
	} else {
		if gs.bindingStorage == nil {
			return constants.ErrNoBindingStorageModule
		}
		svID, err = gs.bindingStorage.GetUserFrontendID(userID, frontendSv.Type)
		if err != nil {
			return err
		}
	}
	if c, ok := gs.clientMap.Load(svID); ok {
		ctxT, done := context.WithTimeout(context.Background(), gs.reqTimeout)
		defer done()
		_, err := c.(protos.PitayaClient).PushToUser(ctxT, push)
		return err
	}
	return constants.ErrNoConnectionToServer
}

// AddServer is called when a new server is discovered
func (gs *GRPCClient) AddServer(sv *Server) {
	var host, port string
	var ok bool
	if host, ok = sv.Metadata["grpc-host"]; !ok {
		logger.Log.Errorf("server %s doesn't have a grpc-host specified in metadata", sv.ID)
		return
	}
	if port, ok = sv.Metadata["grpc-port"]; !ok {
		logger.Log.Errorf("server %s doesn't have a grpc-port specified in metadata", sv.ID)
		return
	}
	address := fmt.Sprintf("%s:%s", host, port)
	ctxT, done := context.WithTimeout(context.Background(), gs.dialTimeout)
	defer done()
	conn, err := grpc.DialContext(ctxT, address, grpc.WithInsecure())
	if err != nil {
		logger.Log.Errorf("unable to connect to server %s at %s: %v", sv.ID, address, err)
		return
	}
	c := protos.NewPitayaClient(conn)
	gs.clientMap.Store(sv.ID, c)
	logger.Log.Debugf("[grpc client] added server %s at %s", sv.ID, address)
}

// RemoveServer is called when a server is removed
func (gs *GRPCClient) RemoveServer(sv *Server) {
	if _, ok := gs.clientMap.Load(sv.ID); ok {
		// TODO: do I need to disconnect client?
		gs.clientMap.Delete(sv.ID)
		logger.Log.Debugf("[grpc client] removed server %s", sv.ID)
	}
}

// AfterInit runs after initialization
func (gs *GRPCClient) AfterInit() {}

// BeforeShutdown runs before shutdown
func (gs *GRPCClient) BeforeShutdown() {}

// Shutdown stops grpc rpc server
func (gs *GRPCClient) Shutdown() error {
	return nil
}
