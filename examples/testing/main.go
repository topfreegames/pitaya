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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/cluster"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/examples/testing/protos"
	"github.com/topfreegames/pitaya/v2/groups"
	logruswrapper "github.com/topfreegames/pitaya/v2/logger/logrus"
	"github.com/topfreegames/pitaya/v2/modules"
	"github.com/topfreegames/pitaya/v2/protos/test"
	"github.com/topfreegames/pitaya/v2/serialize/json"
	"github.com/topfreegames/pitaya/v2/serialize/protobuf"
	"github.com/topfreegames/pitaya/v2/session"
)

// TestSvc service for e2e tests
type TestSvc struct {
	component.Base
	app         pitaya.Pitaya
	sessionPool session.SessionPool
}

// TestRemoteSvc remote service for e2e tests
type TestRemoteSvc struct {
	component.Base
}

// TestRPCRequest for e2e tests
type TestRPCRequest struct {
	Route string `json:"route"`
	Data  string `json:"data"`
}

// TestSendToUsers for e2e tests
type TestSendToUsers struct {
	UIDs []string `json:"uids"`
	Msg  string   `json:"msg"`
}

// RPCTestRawPtrReturnsPtr remote for e2e tests
func (tr *TestRemoteSvc) RPCTestRawPtrReturnsPtr(ctx context.Context, data *test.TestRequest) (*test.TestResponse, error) {
	return &test.TestResponse{
		Code: 200,
		Msg:  fmt.Sprintf("got %s", data.GetMsg()),
	}, nil
}

// RPCTestPtrReturnsPtr remote for e2e tests
func (tr *TestRemoteSvc) RPCTestPtrReturnsPtr(ctx context.Context, req *test.TestRequest) (*test.TestResponse, error) {
	return &test.TestResponse{
		Code: 200,
		Msg:  fmt.Sprintf("got %s", req.Msg),
	}, nil
}

// RPCTestReturnsError remote for e2e tests
func (tr *TestRemoteSvc) RPCTestReturnsError(ctx context.Context, data *test.TestRequest) (*test.TestResponse, error) {
	return nil, pitaya.Error(errors.New("test error"), "PIT-433", map[string]string{"some": "meta"})
}

// RPCTestNoArgs remote for e2e tests
func (tr *TestRemoteSvc) RPCTestNoArgs(ctx context.Context) (*test.TestResponse, error) {
	return &test.TestResponse{
		Code: 200,
		Msg:  "got nothing",
	}, nil
}

// Init inits testsvc
func (t *TestSvc) Init() {
	err := t.app.GroupCreate(context.Background(), "g1")
	if err != nil {
		panic(err)
	}
}

// TestRequestKickUser handler for e2e tests
func (t *TestSvc) TestRequestKickUser(ctx context.Context, userID []byte) (*test.TestResponse, error) {
	s := t.sessionPool.GetSessionByUID(string(userID))
	if s == nil {
		return nil, pitaya.Error(constants.ErrSessionNotFound, "PIT-404")
	}
	err := s.Kick(ctx)
	if err != nil {
		return nil, err
	}
	return &test.TestResponse{
		Code: 200,
		Msg:  "ok",
	}, nil
}

// TestRequestKickMe handler for e2e tests
func (t *TestSvc) TestRequestKickMe(ctx context.Context) (*test.TestResponse, error) {
	s := t.app.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(constants.ErrSessionNotFound, "PIT-404")
	}
	err := s.Kick(ctx)
	if err != nil {
		return nil, err
	}
	return &test.TestResponse{
		Code: 200,
		Msg:  "ok",
	}, nil
}

// TestRequestOnlySessionReturnsPtr handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsPtr(ctx context.Context) (*test.TestResponse, error) {
	return &test.TestResponse{
		Code: 200,
		Msg:  "hello",
	}, nil
}

// TestRequestOnlySessionReturnsPtrNil handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsPtrNil(ctx context.Context) (*test.TestResponse, error) {
	return nil, nil
}

// TestRequestReturnsPtr handler for e2e tests
func (t *TestSvc) TestRequestReturnsPtr(ctx context.Context, in *test.TestRequest) (*test.TestResponse, error) {
	return &test.TestResponse{
		Code: 200,
		Msg:  in.Msg,
	}, nil
}

// TestRequestOnlySessionReturnsRawNil handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsRawNil(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// TestRequestReturnsRaw handler for e2e tests
func (t *TestSvc) TestRequestReturnsRaw(ctx context.Context, in *test.TestRequest) ([]byte, error) {
	return []byte(in.Msg), nil
}

// TestRequestReceiveReturnsRaw handler for e2e tests
func (t *TestSvc) TestRequestReceiveReturnsRaw(ctx context.Context, in []byte) ([]byte, error) {
	return in, nil
}

// TestRequestReturnsError handler for e2e tests
func (t *TestSvc) TestRequestReturnsError(ctx context.Context, in []byte) ([]byte, error) {
	return nil, pitaya.Error(errors.New("somerror"), "PIT-555")
}

// TestBind handler for e2e tests
func (t *TestSvc) TestBind(ctx context.Context) ([]byte, error) {
	uid := uuid.New().String()
	s := t.app.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, uid)
	if err != nil {
		return nil, pitaya.Error(err, "PIT-444")
	}
	err = t.app.GroupAddMember(ctx, "g1", s.UID())
	if err != nil {
		return nil, pitaya.Error(err, "PIT-441")
	}
	return []byte("ack"), nil
}

// TestBindID handler for e2e tests
func (t *TestSvc) TestBindID(ctx context.Context, byteUID []byte) ([]byte, error) {
	s := t.app.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, string(byteUID))
	if err != nil {
		return nil, pitaya.Error(err, "PIT-444")
	}
	err = t.app.GroupAddMember(ctx, "g1", s.UID())
	if err != nil {
		return nil, pitaya.Error(err, "PIT-441")
	}
	return []byte("ack"), nil
}

// TestSendGroupMsg handler for e2e tests
func (t *TestSvc) TestSendGroupMsg(ctx context.Context, msg []byte) {
	t.app.GroupBroadcast(ctx, "connector", "g1", "route.test", msg)
}

// TestSendGroupMsgPtr handler for e2e tests
func (t *TestSvc) TestSendGroupMsgPtr(ctx context.Context, msg *test.TestRequest) {
	t.app.GroupBroadcast(ctx, "connector", "g1", "route.testptr", msg)
}

// TestSendToUsers handler for e2e tests
func (t *TestSvc) TestSendToUsers(ctx context.Context, msg *TestSendToUsers) {
	t.app.SendPushToUsers("route.sendtousers", []byte(msg.Msg), msg.UIDs, "connector")
}

// TestSendRPC tests sending a RPC
func (t *TestSvc) TestSendRPC(ctx context.Context, msg *TestRPCRequest) (*protos.TestResponse, error) {
	rep := &protos.TestResponse{}
	err := t.app.RPC(ctx, msg.Route, rep, &protos.TestRequest{Msg: msg.Data})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// TestSendRPCNoArgs tests sending a RPC
func (t *TestSvc) TestSendRPCNoArgs(ctx context.Context, msg *TestRPCRequest) (*protos.TestResponse, error) {
	rep := &protos.TestResponse{}
	err := t.app.RPC(ctx, msg.Route, rep, nil)
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 32222, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	serializer := flag.String("serializer", "json", "json or protobuf")
	sdPrefix := flag.String("sdprefix", "pitaya/", "prefix to discover other servers")
	debug := flag.Bool("debug", false, "turn on debug logging")
	grpc := flag.Bool("grpc", false, "turn on grpc")
	grpcPort := flag.Int("grpcport", 3434, "the grpc server port")

	flag.Parse()

	cfg := viper.New()
	cfg.Set("pitaya.cluster.sd.etcd.prefix", *sdPrefix)
	cfg.Set("pitaya.cluster.rpc.server.grpc.port", *grpcPort)

	l := logrus.New()
	l.Formatter = &logrus.TextFormatter{}
	l.SetLevel(logrus.InfoLevel)
	if *debug {
		l.SetLevel(logrus.DebugLevel)
	}

	pitaya.SetLogger(logruswrapper.NewWithFieldLogger(l))

	app, bs, sessionPool := createApp(*serializer, *port, *grpc, *isFrontend, *svType, pitaya.Cluster, map[string]string{
		constants.GRPCHostKey: "127.0.0.1",
		constants.GRPCPortKey: fmt.Sprintf("%d", *grpcPort),
	}, cfg)

	if *grpc {
		app.RegisterModule(bs, "bindingsStorage")
	}

	app.Register(
		&TestSvc{
			app:         app,
			sessionPool: sessionPool,
		},
		component.WithName("testsvc"),
		component.WithNameFunc(strings.ToLower),
	)

	app.RegisterRemote(
		&TestRemoteSvc{},
		component.WithName("testremotesvc"),
		component.WithNameFunc(strings.ToLower),
	)

	app.Start()
}

func createApp(serializer string, port int, grpc bool, isFrontend bool, svType string, serverMode pitaya.ServerMode, metadata map[string]string, cfg ...*viper.Viper) (pitaya.Pitaya, *modules.ETCDBindingStorage, session.SessionPool) {
	conf := config.NewConfig(cfg...)
	builder := pitaya.NewBuilderWithConfigs(isFrontend, svType, serverMode, metadata, conf)

	if isFrontend {
		tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
		builder.AddAcceptor(tcp)
	}

	builder.Groups = groups.NewMemoryGroupService(*config.NewDefaultMemoryGroupConfig())

	if serializer == "json" {
		builder.Serializer = json.NewSerializer()
	} else if serializer == "protobuf" {
		builder.Serializer = protobuf.NewSerializer()
	} else {
		panic("serializer should be either json or protobuf")
	}

	var bs *modules.ETCDBindingStorage
	if grpc {
		gs, err := cluster.NewGRPCServer(*config.NewGRPCServerConfig(conf), builder.Server, builder.MetricsReporters)
		if err != nil {
			panic(err)
		}

		bs = modules.NewETCDBindingStorage(builder.Server, builder.SessionPool, *config.NewETCDBindingConfig(conf))

		gc, err := cluster.NewGRPCClient(
			*config.NewGRPCClientConfig(conf),
			builder.Server,
			builder.MetricsReporters,
			bs,
			cluster.NewInfoRetriever(*config.NewInfoRetrieverConfig(conf)),
		)
		if err != nil {
			panic(err)
		}
		builder.RPCServer = gs
		builder.RPCClient = gc
	}

	return builder.Build(), bs, builder.SessionPool
}
