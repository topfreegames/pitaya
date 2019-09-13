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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/metrics"
	metricsmocks "github.com/topfreegames/pitaya/metrics/mocks"
	"github.com/topfreegames/pitaya/protos"
	protosmocks "github.com/topfreegames/pitaya/protos/mocks"
	"github.com/topfreegames/pitaya/session"
)

func TestNewNatsRPCServer(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}

	cfg := getConfig()
	sv := getServer()
	n, err := NewNatsRPCServer(cfg, sv, mockMetricsReporters, nil)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, sv, n.server)
	assert.Equal(t, cfg, n.config)
	assert.Equal(t, mockMetricsReporters, n.metricsReporters)
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
			cfg.Set("pitaya.buffer.cluster.rpc.server.nats.messages", table.messagesBufferSize)
			cfg.Set("pitaya.buffer.cluster.rpc.server.nats.push", table.pushBufferSize)
			conf := getConfig(cfg)
			_, err := NewNatsRPCServer(conf, getServer(), nil, nil)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestNatsRPCServerGetUserMessagesTopic(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pitaya/connector/user/bla/push", GetUserMessagesTopic("bla", "connector"))
	assert.Equal(t, "pitaya/game/user/123bla/push", GetUserMessagesTopic("123bla", "game"))
	assert.Equal(t, "pitaya/connector/user/1/push", GetUserMessagesTopic("1", "connector"))
}

func TestNatsRPCServerGetUserKickTopic(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pitaya/connector/user/0/kick", GetUserKickTopic("0", "connector"))
	assert.Equal(t, "pitaya/game/user/1/kick", GetUserKickTopic("1", "game"))
	assert.Equal(t, "pitaya/connector/user/10/kick", GetUserKickTopic("10", "connector"))
	assert.Equal(t, "pitaya/game/user/11/kick", GetUserKickTopic("11", "game"))
}

func TestNatsRPCServerGetUnhandledRequestsChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	assert.NotNil(t, n.GetUnhandledRequestsChannel())
	assert.IsType(t, make(chan *protos.Request), n.GetUnhandledRequestsChannel())
}

func TestNatsRPCServerGetBindingsChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	assert.Equal(t, n.bindingsChan, n.GetBindingsChannel())
}

func TestNatsRPCServerOnSessionBind(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	rpcServer.conn = conn
	sess := session.New(nil, true, "uid123")
	assert.Nil(t, sess.Subscriptions)
	err = rpcServer.onSessionBind(context.Background(), sess)
	assert.NoError(t, err)
	assert.NotNil(t, sess.Subscriptions)
	assert.Len(t, sess.Subscriptions, 2)
	assert.NotNil(t, rpcServer.userKickCh)
}

func TestNatsRPCServerSubscribeToBindingsChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	rpcServer.conn = conn
	err = rpcServer.subscribeToBindingsChannel()
	assert.NoError(t, err)
	dt := []byte("somedata")
	conn.Publish(GetBindBroadcastTopic(sv.Type), dt)
	msg := helpers.ShouldEventuallyReceive(t, rpcServer.GetBindingsChannel()).(*nats.Msg)
	assert.Equal(t, msg.Data, dt)
}

func TestNatsRPCServerSubscribeUserKickChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	rpcServer.conn = conn
	sub, err := rpcServer.subscribeToUserKickChannel("someuid", sv.Type)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	kick := &protos.KickMsg{UserId: "randomid"}
	dt, err := proto.Marshal(kick)
	assert.NoError(t, err)
	err = conn.Publish(GetUserKickTopic("someuid", sv.Type), dt)
	assert.NoError(t, err)
	msg := helpers.ShouldEventuallyReceive(t, rpcServer.getUserKickChannel()).(*protos.KickMsg)
	assert.Equal(t, msg.UserId, kick.UserId)
}

func TestNatsRPCServerGetUserPushChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	assert.NotNil(t, n.getUserPushChannel())
	assert.IsType(t, make(chan *protos.Push), n.getUserPushChannel())
}

func TestNatsRPCServerGetUserKickChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	assert.NotNil(t, n.getUserKickChannel())
	assert.IsType(t, make(chan *protos.KickMsg), n.getUserKickChannel())
}

func TestNatsRPCServerSubscribeToUserMessages(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	rpcServer.conn = conn
	tables := []struct {
		uid    string
		svType string
		msg    []byte
	}{
		{"user1", "conn", []byte("msg1")},
		{"user2", "game", []byte("")},
		{"u", "conn", []byte("000")},
	}

	for _, table := range tables {
		t.Run(table.uid, func(t *testing.T) {
			subs, err := rpcServer.subscribeToUserMessages(table.uid, table.svType)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			conn.Publish(GetUserMessagesTopic(table.uid, table.svType), table.msg)
			helpers.ShouldEventuallyReceive(t, rpcServer.userPushCh)
		})
	}
}

func TestNatsRPCServerSubscribe(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(cfg, sv, nil, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}

	rpcServer, _ := NewNatsRPCServer(cfg, sv, mockMetricsReporters, nil)
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	rpcServer.conn = conn
	tables := []struct {
		topic string
		req   *protos.Request
	}{
		{"user1/messages", &protos.Request{Type: protos.RPCType_Sys, FrontendID: "bla", Msg: &protos.Msg{Id: 1, Reply: "ae"}}},
		{"user2/messages", &protos.Request{Type: protos.RPCType_User, FrontendID: "bla2", Msg: &protos.Msg{Id: 1}}},
	}

	go rpcServer.handleMessages()

	for _, table := range tables {
		t.Run(table.topic, func(t *testing.T) {
			subs, err := rpcServer.subscribe(table.topic)
			assert.NoError(t, err)
			assert.Equal(t, true, subs.IsValid())
			b, err := proto.Marshal(table.req)
			assert.NoError(t, err)

			mockMetricsReporter.EXPECT().ReportGauge(metrics.DroppedMessages, gomock.Any(), float64(0))
			mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), gomock.Any()).Times(3)

			conn.Publish(table.topic, b)
			r := helpers.ShouldEventuallyReceive(t, rpcServer.unhandledReqCh).(*protos.Request)
			assert.Equal(t, table.req.FrontendID, r.FrontendID)
			assert.Equal(t, table.req.Msg.Id, r.Msg.Id)
		})
	}
}

func TestNatsRPCServerInitShouldFailIfConnFails(t *testing.T) {
	t.Parallel()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", "nats://localhost:1")
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv, nil, nil)
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
	rpcServer, _ := NewNatsRPCServer(config, sv, nil, nil)
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
		{"test1", getChannel(sv.Type, sv.ID), &protos.Request{Type: protos.RPCType_Sys, FrontendID: "bla", Msg: &protos.Msg{Id: 1, Reply: "ae"}}},
		{"test2", getChannel(sv.Type, sv.ID), &protos.Request{Type: protos.RPCType_User, FrontendID: "bla2", Msg: &protos.Msg{Id: 1, Reply: "boa"}}},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			c := make(chan *nats.Msg)
			rpcServer.conn.ChanSubscribe(table.req.Msg.Reply, c)
			rpcServer.unhandledReqCh <- table.req
			r := helpers.ShouldEventuallyReceive(t, c).(*nats.Msg)
			assert.NotNil(t, r.Data)
		})
	}
}

func TestNatsRPCServerProcessBindings(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv, nil, nil)
	err := rpcServer.Init()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	pitayaSvMock := protosmocks.NewMockPitayaServer(ctrl)
	defer ctrl.Finish()

	rpcServer.SetPitayaServer(pitayaSvMock)

	bindMsg := &protos.BindMsg{
		Uid: "testuid",
		Fid: "testfid",
	}

	bindData, err := proto.Marshal(bindMsg)
	assert.NoError(t, err)

	msg := &nats.Msg{
		Data: bindData,
	}

	unmarshalledMsg := &protos.BindMsg{}
	_ = proto.Unmarshal(bindData, unmarshalledMsg)

	pitayaSvMock.EXPECT().SessionBindRemote(context.Background(), unmarshalledMsg).Do(func(ctx context.Context, b *protos.BindMsg) {
		assert.Equal(t, bindMsg.Uid, b.Uid)
		assert.Equal(t, bindMsg.Fid, b.Fid)
	})

	rpcServer.bindingsChan <- msg
	time.Sleep(30 * time.Millisecond)
}

func TestNatsRPCServerProcessPushes(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv, nil, nil)
	err := rpcServer.Init()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	pitayaSvMock := protosmocks.NewMockPitayaServer(ctrl)
	defer ctrl.Finish()

	rpcServer.SetPitayaServer(pitayaSvMock)

	push := &protos.Push{
		Route: "someroute",
		Uid:   "someuid",
		Data:  []byte{0x01},
	}

	pitayaSvMock.EXPECT().PushToUser(context.Background(), push).Do(func(ctx context.Context, p *protos.Push) {
		assert.Equal(t, push.Route, p.Route)
		assert.Equal(t, push.Uid, p.Uid)
		assert.Equal(t, push.Data, p.Data)
	})

	rpcServer.userPushCh <- push
	time.Sleep(30 * time.Millisecond)
}

func TestNatsRPCServerProcessKick(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()
	rpcServer, _ := NewNatsRPCServer(config, sv, nil, nil)
	err := rpcServer.Init()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	pitayaSvMock := protosmocks.NewMockPitayaServer(ctrl)
	defer ctrl.Finish()

	rpcServer.SetPitayaServer(pitayaSvMock)

	kick := &protos.KickMsg{
		UserId: "someuid",
	}

	pitayaSvMock.EXPECT().KickUser(gomock.Any(), kick).Do(func(ctx context.Context, p *protos.KickMsg) {
		assert.Equal(t, kick.UserId, p.UserId)
	})

	rpcServer.userKickCh <- kick
	time.Sleep(30 * time.Millisecond)
}

func TestNatsRPCServerReportMetrics(t *testing.T) {
	cfg := getConfig()
	sv := getServer()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}

	rpcServer, _ := NewNatsRPCServer(cfg, sv, mockMetricsReporters, nil)
	rpcServer.dropped = 100
	rpcServer.messagesBufferSize = 100
	rpcServer.pushBufferSize = 100

	rpcServer.subChan <- &nats.Msg{}
	rpcServer.bindingsChan <- &nats.Msg{}
	rpcServer.userPushCh <- &protos.Push{}

	mockMetricsReporter.EXPECT().ReportGauge(metrics.DroppedMessages, gomock.Any(), float64(rpcServer.dropped))
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(99)).Times(3)
	rpcServer.reportMetrics()
}
