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
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	pitErrors "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/interfaces"
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
	bindingStorage   interfaces.BindingStorage
	clientMap        sync.Map
	dialTimeout      time.Duration
	infoRetriever    InfoRetriever
	lazy             bool
	metricsReporters []metrics.Reporter
	reqTimeout       time.Duration
	server           *Server
}

// NewGRPCClient returns a new instance of GRPCClient
func NewGRPCClient(
	config *config.Config,
	server *Server,
	metricsReporters []metrics.Reporter,
	bindingStorage interfaces.BindingStorage,
	infoRetriever InfoRetriever,
) (*GRPCClient, error) {
	gs := &GRPCClient{
		bindingStorage:   bindingStorage,
		infoRetriever:    infoRetriever,
		metricsReporters: metricsReporters,
		server:           server,
	}

	gs.configure(config)
	return gs, nil
}

type grpcClient struct {
	address   string
	cli       protos.PitayaClient
	conn      *grpc.ClientConn
	connected bool
	lock      sync.Mutex
}

// Init inits grpc rpc client
func (gs *GRPCClient) Init() error {
	return nil
}

func (gs *GRPCClient) configure(cfg *config.Config) {
	gs.dialTimeout = cfg.GetDuration("pitaya.cluster.rpc.client.grpc.dialtimeout")
	gs.lazy = cfg.GetBool("pitaya.cluster.rpc.client.grpc.lazyconnection")
	gs.reqTimeout = cfg.GetDuration("pitaya.cluster.rpc.client.grpc.requesttimeout")
}

// Call makes a RPC Call
func (gs *GRPCClient) Call(
	ctx context.Context,
	rpcType protos.RPCType,
	route *route.Route,
	session *session.Session,
	msg *message.Message,
	server *Server,
) (*protos.Response, error) {
	c, ok := gs.clientMap.Load(server.ID)
	if !ok {
		return nil, constants.ErrNoConnectionToServer
	}

	parent, err := tracing.ExtractSpan(ctx)
	if err != nil {
		logger.Log.Warnf("[grpc client] failed to retrieve parent span: %s", err.Error())
	}
	tags := opentracing.Tags{
		"span.kind":       "client",
		"local.id":        gs.server.ID,
		"peer.serverType": server.Type,
		"peer.id":         server.ID,
	}
	ctx = tracing.StartSpan(ctx, "GRPC RPC Call", tags, parent)
	defer tracing.FinishSpan(ctx, err)

	req, err := buildRequest(ctx, rpcType, route, session, msg, gs.server)
	if err != nil {
		return nil, err
	}

	ctxT, done := context.WithTimeout(ctx, gs.reqTimeout)
	defer done()

	if gs.metricsReporters != nil {
		startTime := time.Now()
		ctxT = pcontext.AddToPropagateCtx(ctxT, constants.StartTimeKey, startTime.UnixNano())
		ctxT = pcontext.AddToPropagateCtx(ctxT, constants.RouteKey, route.String())
		defer metrics.ReportTimingFromCtx(ctxT, gs.metricsReporters, "rpc", err)
	}

	res, err := c.(*grpcClient).call(ctxT, &req)
	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		if res.Error.Code == "" {
			res.Error.Code = pitErrors.ErrUnknownCode
		}
		err = &pitErrors.Error{
			Code:     res.Error.Code,
			Message:  res.Error.Msg,
			Metadata: res.Error.Metadata,
		}
		return nil, err
	}
	return res, nil
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
			err := c.(*grpcClient).sessionBindRemote(ctxT, msg)
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
		err := c.(*grpcClient).sendKick(ctxT, kick)
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
		err := c.(*grpcClient).pushToUser(ctxT, push)
		return err
	}
	return constants.ErrNoConnectionToServer
}

// AddServer is called when a new server is discovered
func (gs *GRPCClient) AddServer(sv *Server) {
	var host, port, portKey string
	var ok bool

	host, portKey = gs.getServerHost(sv)
	if host == "" {
		logger.Log.Errorf("[grpc client] server %s has no grpcHost specified in metadata", sv.ID)
		return
	}

	if port, ok = sv.Metadata[portKey]; !ok {
		logger.Log.Errorf("[grpc client] server %s has no %s specified in metadata", sv.ID, portKey)
		return
	}

	address := fmt.Sprintf("%s:%s", host, port)
	client := &grpcClient{address: address}
	if !gs.lazy {
		if err := client.connect(); err != nil {
			logger.Log.Errorf("[grpc client] unable to connect to server %s at %s: %v", sv.ID, address, err)
		}
	}
	gs.clientMap.Store(sv.ID, client)
	logger.Log.Debugf("[grpc client] added server %s at %s", sv.ID, address)
}

// RemoveServer is called when a server is removed
func (gs *GRPCClient) RemoveServer(sv *Server) {
	if c, ok := gs.clientMap.Load(sv.ID); ok {
		c.(*grpcClient).disconnect()
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

func (gs *GRPCClient) getServerHost(sv *Server) (host, portKey string) {
	var (
		serverRegion, hasRegion   = sv.Metadata[constants.RegionKey]
		externalHost, hasExternal = sv.Metadata[constants.GRPCExternalHostKey]
		internalHost, _           = sv.Metadata[constants.GRPCHostKey]
	)

	hasRegion = hasRegion && serverRegion != ""
	hasExternal = hasExternal && externalHost != ""

	if !hasRegion {
		if hasExternal {
			logger.Log.Warnf("[grpc client] server %s has no region specified in metadata, using external host", sv.ID)
			return externalHost, constants.GRPCExternalPortKey
		}

		logger.Log.Warnf("[grpc client] server %s has no region nor external host specified in metadata, using internal host", sv.ID)
		return internalHost, constants.GRPCPortKey
	}

	if gs.infoRetriever.Region() == serverRegion || !hasExternal {
		logger.Log.Infof("[grpc client] server %s is in same region or external host not provided, using internal host", sv.ID)
		return internalHost, constants.GRPCPortKey
	}

	logger.Log.Infof("[grpc client] server %s is in other region, using external host", sv.ID)
	return externalHost, constants.GRPCExternalPortKey
}

func (gc *grpcClient) connect() error {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	if gc.connected {
		return nil
	}

	conn, err := grpc.Dial(
		gc.address,
		grpc.WithInsecure(),
	)
	if err != nil {
		return err
	}
	c := protos.NewPitayaClient(conn)
	gc.cli = c
	gc.conn = conn
	gc.connected = true
	return nil
}

func (gc *grpcClient) disconnect() {
	gc.lock.Lock()
	if gc.connected {
		gc.conn.Close()
		gc.connected = false
	}
	gc.lock.Unlock()
}

func (gc *grpcClient) pushToUser(ctx context.Context, push *protos.Push) error {
	if !gc.connected {
		if err := gc.connect(); err != nil {
			return err
		}
	}
	_, err := gc.cli.PushToUser(ctx, push)
	return err
}

func (gc *grpcClient) call(ctx context.Context, req *protos.Request) (*protos.Response, error) {
	if !gc.connected {
		if err := gc.connect(); err != nil {
			return nil, err
		}
	}
	return gc.cli.Call(ctx, req)
}

func (gc *grpcClient) sessionBindRemote(ctx context.Context, req *protos.BindMsg) error {
	if !gc.connected {
		if err := gc.connect(); err != nil {
			return err
		}
	}
	_, err := gc.cli.SessionBindRemote(ctx, req)
	return err
}

func (gc *grpcClient) sendKick(ctx context.Context, req *protos.KickMsg) error {
	if !gc.connected {
		if err := gc.connect(); err != nil {
			return err
		}
	}
	_, err := gc.cli.KickUser(ctx, req)
	return err
}
