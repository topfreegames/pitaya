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
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/serialize/protobuf"
	"github.com/topfreegames/pitaya/session"
)

// TestSvc service for e2e tests
type TestSvc struct {
	component.Base
	group *pitaya.Group
}

// TestRemoteSvc remote service for e2e tests
type TestRemoteSvc struct {
	component.Base
}

// TestResponse for e2e tests
type TestResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TestRequest for e2e tests
type TestRequest struct {
	Msg string `json:"msg"`
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
func (tr *TestRemoteSvc) RPCTestRawPtrReturnsPtr(ctx context.Context, data []byte) (*TestResponse, error) {
	return &TestResponse{
		Code: 200,
		Msg:  fmt.Sprintf("got %s", string(data)),
	}, nil
}

// RPCTestPtrReturnsPtr remote for e2e tests
func (tr *TestRemoteSvc) RPCTestPtrReturnsPtr(ctx context.Context, req *TestRequest) (*TestResponse, error) {
	return &TestResponse{
		Code: 200,
		Msg:  fmt.Sprintf("got %s", req.Msg),
	}, nil
}

// RPCTestReturnsError remote for e2e tests
func (tr *TestRemoteSvc) RPCTestReturnsError(ctx context.Context, data []byte) (*TestResponse, error) {
	return nil, pitaya.Error(errors.New("test error"), "PIT-433", map[string]string{"some": "meta"})

}

// RPCTestNoArgs remote for e2e tests
func (tr *TestRemoteSvc) RPCTestNoArgs(ctx context.Context) (*TestResponse, error) {
	return &TestResponse{
		Code: 200,
		Msg:  "got nothing",
	}, nil
}

// Init inits testsvc
func (t *TestSvc) Init() {
	t.group = pitaya.NewGroup("g1")
}

// TestRequestKickUser handler for e2e tests
func (t *TestSvc) TestRequestKickUser(ctx context.Context, userID []byte) (*TestResponse, error) {
	s := session.GetSessionByUID(string(userID))
	if s == nil {
		return nil, pitaya.Error(constants.ErrSessionNotFound, "PIT-404")
	}
	err := s.Kick(ctx)
	if err != nil {
		return nil, err
	}
	return &TestResponse{
		Code: 200,
		Msg:  "ok",
	}, nil
}

// TestRequestKickMe handler for e2e tests
func (t *TestSvc) TestRequestKickMe(ctx context.Context) (*TestResponse, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(constants.ErrSessionNotFound, "PIT-404")
	}
	err := s.Kick(ctx)
	if err != nil {
		return nil, err
	}
	return &TestResponse{
		Code: 200,
		Msg:  "ok",
	}, nil
}

// TestRequestOnlySessionReturnsPtr handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsPtr(ctx context.Context) (*TestResponse, error) {
	return &TestResponse{
		Code: 200,
		Msg:  "hello",
	}, nil
}

// TestRequestOnlySessionReturnsPtrNil handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsPtrNil(ctx context.Context) (*TestResponse, error) {
	return nil, nil
}

// TestRequestReturnsPtr handler for e2e tests
func (t *TestSvc) TestRequestReturnsPtr(ctx context.Context, in *TestRequest) (*TestResponse, error) {
	return &TestResponse{
		Code: 200,
		Msg:  in.Msg,
	}, nil
}

// TestRequestOnlySessionReturnsRawNil handler for e2e tests
func (t *TestSvc) TestRequestOnlySessionReturnsRawNil(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// TestRequestReturnsRaw handler for e2e tests
func (t *TestSvc) TestRequestReturnsRaw(ctx context.Context, in *TestRequest) ([]byte, error) {
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
	s := pitaya.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, uid)
	if err != nil {
		return nil, pitaya.Error(err, "PIT-444")
	}
	err = t.group.Add(s)
	if err != nil {
		return nil, pitaya.Error(err, "PIT-441")
	}
	return []byte("ack"), nil
}

// TestBindID handler for e2e tests
func (t *TestSvc) TestBindID(ctx context.Context, byteUID []byte) ([]byte, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, string(byteUID))
	if err != nil {
		return nil, pitaya.Error(err, "PIT-444")
	}
	err = t.group.Add(s)
	if err != nil {
		return nil, pitaya.Error(err, "PIT-441")
	}
	return []byte("ack"), nil
}

// TestSendGroupMsg handler for e2e tests
func (t *TestSvc) TestSendGroupMsg(ctx context.Context, msg []byte) {
	t.group.Broadcast("route.test", msg)
}

// TestSendGroupMsgPtr handler for e2e tests
func (t *TestSvc) TestSendGroupMsgPtr(ctx context.Context, msg *TestRequest) {
	t.group.Broadcast("route.testptr", msg)
}

// TestSendToUsers handler for e2e tests
func (t *TestSvc) TestSendToUsers(ctx context.Context, msg *TestSendToUsers) {
	pitaya.SendPushToUsers("route.sendtousers", []byte(msg.Msg), msg.UIDs, "connector")
}

// TestSendRPCPointer tests sending a RPC
func (t *TestSvc) TestSendRPCPointer(ctx context.Context, msg *TestRPCRequest) (*TestResponse, error) {
	rep := &TestResponse{}
	err := pitaya.RPC(ctx, msg.Route, rep, &TestRequest{Msg: msg.Data})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// TestSendRPC tests sending a RPC
func (t *TestSvc) TestSendRPC(ctx context.Context, msg *TestRPCRequest) (*TestResponse, error) {
	rep := &TestResponse{}
	err := pitaya.RPC(ctx, msg.Route, rep, []byte(msg.Data))
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// TestSendRPCNoArgs tests sending a RPC
func (t *TestSvc) TestSendRPCNoArgs(ctx context.Context, msg *TestRPCRequest) (*TestResponse, error) {
	rep := &TestResponse{}
	err := pitaya.RPC(ctx, msg.Route, rep, []byte(nil))
	if err != nil {
		return nil, err
	}
	return rep, nil
}

func main() {
	gob.Register(&TestRequest{})

	port := flag.Int("port", 32222, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	serializer := flag.String("serializer", "json", "json or protobuf")
	sdPrefix := flag.String("sdprefix", "pitaya/", "prefix to discover other servers")
	debug := flag.Bool("debug", false, "turn on debug logging")

	flag.Parse()

	l := logrus.New()
	l.Formatter = &logrus.TextFormatter{}
	l.SetLevel(logrus.InfoLevel)
	if *debug {
		l.SetLevel(logrus.DebugLevel)
	}

	pitaya.SetLogger(l)

	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", *port))

	pitaya.Register(
		&TestSvc{},
		component.WithName("testsvc"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.RegisterRemote(
		&TestRemoteSvc{},
		component.WithName("testremotesvc"),
		component.WithNameFunc(strings.ToLower),
	)

	if *serializer == "json" {
		pitaya.SetSerializer(json.NewSerializer())
	} else if *serializer == "protobuf" {
		pitaya.SetSerializer(protobuf.NewSerializer())
	} else {
		panic("serializer should be either json or protobuf")
	}

	if *isFrontend {
		pitaya.AddAcceptor(tcp)
	}

	cfg := viper.New()
	cfg.Set("pitaya.cluster.sd.etcd.prefix", *sdPrefix)
	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{}, cfg)

	pitaya.Start()
}
