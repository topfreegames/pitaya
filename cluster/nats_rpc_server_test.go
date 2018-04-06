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
	"testing"

	"github.com/gogo/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/protos"
)

func getServer() *Server {
	return &Server{
		ID:       "id1",
		Type:     "type1",
		Frontend: true,
	}
}

func TestNewNatsRPCServer(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, err := NewNatsRPCServer(cfg, sv)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, sv, n.server)
	assert.Equal(t, cfg, n.config)
}

func TestNatsRPCServerConfigure(t *testing.T) {
	t.Parallel()
	tables := []struct {
		natsConnect        string
		messagesBufferSize int
		pushBufferSize     int
		err                error
	}{
		{"nats://localhost:2333", 10, 10, nil},
		{"nats://localhost:2333", 10, 0, constants.ErrNatsPushBufferSizeZero},
		{"nats://localhost:2333", 0, 10, constants.ErrNatsMessagesBufferSizeZero},
		{"", 10, 10, constants.ErrNoNatsConnectionString},
	}

	for _, table := range tables {
		t.Run(fmt.Sprintf("%s-%d-%d", table.natsConnect, table.messagesBufferSize, table.pushBufferSize), func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("pitaya.cluster.rpc.server.nats.connect", table.natsConnect)
			cfg.Set("pitaya.buffer.cluster.rpc.server.messages", table.messagesBufferSize)
			cfg.Set("pitaya.buffer.cluster.rpc.server.push", table.pushBufferSize)
			conf := getConfig(cfg)
			_, err := NewNatsRPCServer(conf, getServer())
			assert.Equal(t, table.err, err)
		})
	}
}

func TestNatsRPCServerGetUserMessagesTopic(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pitaya/user/bla/push", GetUserMessagesTopic("bla"))
	assert.Equal(t, "pitaya/user/123bla/push", GetUserMessagesTopic("123bla"))
	assert.Equal(t, "pitaya/user/1/push", GetUserMessagesTopic("1"))
}

func TestNatsRPCServerGetChannel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pitaya/servers/type1/sv1", getChannel("type1", "sv1"))
	assert.Equal(t, "pitaya/servers/2type1/2sv1", getChannel("2type1", "2sv1"))
}

func TestNatsRPCServerGetUnhandledRequestsChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv)
	assert.NotNil(t, n.GetUnhandledRequestsChannel())
	assert.IsType(t, make(chan *protos.Request), n.GetUnhandledRequestsChannel())
}

func TestNatsRPCServerGetUserPushChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv)
	assert.NotNil(t, n.GetUserPushChannel())
	assert.IsType(t, make(chan *protos.Push), n.GetUserPushChannel())
}

func TestNatsRPCServerSetupNatsConn(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	_, _ = NewNatsRPCServer(cfg, sv)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestNatsRPCServerSetupNatsConnShouldError(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	_, _ = NewNatsRPCServer(cfg, sv)
	conn, err := setupNatsConn("nats://localhost:1234")
	assert.Error(t, err)
	assert.Nil(t, conn)
}

func TestNatsRPCServerSubscribeToUserMessages(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	rpcServer.conn = conn
	tables := []struct {
		uid string
		msg []byte
	}{
		{"user1", []byte("msg1")},
		{"user2", []byte("")},
		{"u", []byte("000")},
	}

	for _, table := range tables {
		t.Run(table.uid, func(t *testing.T) {
			subs, err := rpcServer.SubscribeToUserMessages(table.uid)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			conn.Publish(GetUserMessagesTopic(table.uid), table.msg)
			helpers.ShouldEventuallyReceive(t, rpcServer.userPushCh)
		})
	}
}

func TestNatsRPCServerSubscribe(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	rpcServer.conn = conn
	tables := []struct {
		topic string
		msg   []byte
	}{
		{"user1/messages", []byte("msg1")},
		{"user2/messages", []byte("")},
		{"u/messages", []byte("000")},
	}

	for _, table := range tables {
		t.Run(table.topic, func(t *testing.T) {
			subs, err := rpcServer.subscribe(table.topic)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			conn.Publish(table.topic, table.msg)
			r := helpers.ShouldEventuallyReceive(t, rpcServer.subChan).(*nats.Msg)
			assert.Equal(t, table.msg, r.Data)
		})
	}
}

func TestNatsRPCServerHandleMessages(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	rpcServer.conn = conn
	tables := []struct {
		topic string
		req   *protos.Request
	}{
		{"user1/messages", &protos.Request{Type: protos.RPCType_Sys, FrontendID: "bla", Msg: &protos.Msg{ID: 1, Reply: "ae"}}},
		{"user2/messages", &protos.Request{Type: protos.RPCType_User, FrontendID: "bla2", Msg: &protos.Msg{ID: 1}}},
	}

	for _, table := range tables {
		t.Run(table.topic, func(t *testing.T) {
			subs, err := rpcServer.subscribe(table.topic)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			b, err := proto.Marshal(table.req)
			assert.NoError(t, err)
			conn.Publish(table.topic, b)
			go rpcServer.handleMessages()
			r := helpers.ShouldEventuallyReceive(t, rpcServer.unhandledReqCh).(*protos.Request)
			assert.Equal(t, table.req.FrontendID, r.FrontendID)
			assert.Equal(t, table.req.Msg.ID, r.Msg.ID)
		})
	}
}

func TestNatsRPCServerInitShouldFailIfConnFails(t *testing.T) {
	t.Parallel()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", "nats://localhost:1")
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv)
	err := rpcServer.Init()
	assert.Error(t, err)
}

func TestNatsRPCServerInit(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv)
	err := rpcServer.Init()
	assert.NoError(t, err)
	// should setup conn
	assert.NotNil(t, rpcServer.conn)
	assert.True(t, rpcServer.conn.IsConnected())
	// should subscribe
	assert.True(t, rpcServer.sub.IsValid())
	//should handle messages
	tables := []struct {
		name  string
		topic string
		req   *protos.Request
	}{
		{"test1", getChannel(sv.Type, sv.ID), &protos.Request{Type: protos.RPCType_Sys, FrontendID: "bla", Msg: &protos.Msg{ID: 1, Reply: "ae"}}},
		{"test2", getChannel(sv.Type, sv.ID), &protos.Request{Type: protos.RPCType_User, FrontendID: "bla2", Msg: &protos.Msg{ID: 1}}},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			subs, err := rpcServer.subscribe(table.topic)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			b, err := proto.Marshal(table.req)
			assert.NoError(t, err)
			rpcServer.conn.Publish(table.topic, b)
			r := helpers.ShouldEventuallyReceive(t, rpcServer.unhandledReqCh).(*protos.Request)
			<-rpcServer.unhandledReqCh
			assert.Equal(t, table.req.FrontendID, r.FrontendID)
			assert.Equal(t, table.req.Msg.ID, r.Msg.ID)
		})
	}
}
