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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/metrics"
	metricsmocks "github.com/topfreegames/pitaya/metrics/mocks"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
)

func TestNewNatsRPCClient(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}

	cfg := getConfig()
	sv := getServer()
	n, err := NewNatsRPCClient(cfg, sv, mockMetricsReporters, nil)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, sv, n.server)
	assert.Equal(t, cfg, n.config)
	assert.Equal(t, mockMetricsReporters, n.metricsReporters)
	assert.False(t, n.running)
}

func TestNatsRPCClientConfigure(t *testing.T) {
	t.Parallel()
	tables := []struct {
		natsConnect string
		reqTimeout  string
		err         error
	}{
		{"nats://localhost:2333", "10s", nil},
		{"nats://localhost:2333", "0", constants.ErrNatsNoRequestTimeout},
		{"", "10s", constants.ErrNoNatsConnectionString},
	}

	for _, table := range tables {
		t.Run(fmt.Sprintf("%s-%s", table.natsConnect, table.reqTimeout), func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("pitaya.cluster.rpc.client.nats.connect", table.natsConnect)
			cfg.Set("pitaya.cluster.rpc.client.nats.requesttimeout", table.reqTimeout)
			conf := getConfig(cfg)
			_, err := NewNatsRPCClient(conf, getServer(), nil, nil)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestNatsRPCClientGetSubscribeChannel(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCClient(cfg, sv, nil, nil)
	assert.Equal(t, fmt.Sprintf("pitaya/servers/%s/%s", n.server.Type, n.server.ID), n.getSubscribeChannel())
}

func TestNatsRPCClientStop(t *testing.T) {
	t.Parallel()
	cfg := getConfig()
	sv := getServer()
	n, _ := NewNatsRPCClient(cfg, sv, nil, nil)
	// change it to true to ensure it goes to false
	n.running = true
	n.stop()
	assert.False(t, n.running)
}

func TestNatsRPCClientInitShouldFailIfConnFails(t *testing.T) {
	t.Parallel()
	sv := getServer()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", "nats://localhost:1")
	config := getConfig(cfg)
	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	err := rpcClient.Init()
	assert.Error(t, err)
}

func TestNatsRPCClientInit(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()

	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	err := rpcClient.Init()
	assert.NoError(t, err)
	assert.True(t, rpcClient.running)

	// should setup conn
	assert.NotNil(t, rpcClient.conn)
	assert.True(t, rpcClient.conn.IsConnected())
}

func TestNatsRPCClientBroadcastSessionBind(t *testing.T) {
	uid := "testuid123"
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()

	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	rpcClient.Init()

	subChan := make(chan *nats.Msg)
	subs, err := rpcClient.conn.ChanSubscribe(GetBindBroadcastTopic(sv.Type), subChan)
	assert.NoError(t, err)
	// TODO this is ugly, can lead to flaky tests and we could probably do it better
	time.Sleep(50 * time.Millisecond)

	err = rpcClient.BroadcastSessionBind(uid)
	assert.NoError(t, err)

	m := helpers.ShouldEventuallyReceive(t, subChan).(*nats.Msg)

	bMsg := &protos.BindMsg{}
	err = proto.Unmarshal(m.Data, bMsg)
	assert.NoError(t, err)

	assert.Equal(t, uid, bMsg.Uid)
	assert.Equal(t, sv.ID, bMsg.Fid)

	subs.Unsubscribe()
}

func TestNatsRPCClientSendKick(t *testing.T) {
	uid := "testuid"
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()

	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	err := rpcClient.Init()
	assert.NoError(t, err)

	kickChan := make(chan *nats.Msg)
	subs, err := rpcClient.conn.ChanSubscribe(GetUserKickTopic(uid, sv.Type), kickChan)
	assert.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	kick := &protos.KickMsg{
		UserId: uid,
	}

	err = rpcClient.SendKick(uid, sv.Type, kick)
	assert.NoError(t, err)

	m := helpers.ShouldEventuallyReceive(t, kickChan).(*nats.Msg)

	actual := &protos.KickMsg{}
	err = proto.Unmarshal(m.Data, actual)
	assert.NoError(t, err)

	assert.Equal(t, kick.UserId, actual.UserId)
	err = subs.Unsubscribe()
	assert.NoError(t, err)
}

func TestNatsRPCClientSendPush(t *testing.T) {
	uid := "testuid123"
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()

	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	rpcClient.Init()

	subChan := make(chan *nats.Msg)
	subs, err := rpcClient.conn.ChanSubscribe(GetUserMessagesTopic(uid, sv.Type), subChan)
	assert.NoError(t, err)
	// TODO this is ugly, can lead to flaky tests and we could probably do it better
	time.Sleep(50 * time.Millisecond)

	push := &protos.Push{
		Route: "hellow",
		Uid:   uid,
		Data:  []byte{0x01},
	}

	err = rpcClient.SendPush(uid, sv, push)
	assert.NoError(t, err)

	m := helpers.ShouldEventuallyReceive(t, subChan).(*nats.Msg)

	actual := &protos.Push{}
	err = proto.Unmarshal(m.Data, actual)
	assert.NoError(t, err)

	assert.Equal(t, push.Route, actual.Route)
	assert.Equal(t, push.Uid, actual.Uid)
	assert.Equal(t, push.Data, actual.Data)

	subs.Unsubscribe()

}

func TestNatsRPCClientSendShouldFailIfNotRunning(t *testing.T) {
	config := getConfig()
	sv := getServer()
	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	err := rpcClient.Send("topic", []byte("data"))
	assert.Equal(t, constants.ErrRPCClientNotInitialized, err)
}

func TestNatsRPCClientSend(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	config := getConfig(cfg)
	sv := getServer()

	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	rpcClient.Init()

	tables := []struct {
		name  string
		topic string
		data  []byte
	}{
		{"test1", getChannel(sv.Type, sv.ID), []byte("test1")},
		{"test2", getChannel(sv.Type, sv.ID), []byte("test2")},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			subChan := make(chan *nats.Msg)
			subs, err := rpcClient.conn.ChanSubscribe(table.topic, subChan)
			assert.NoError(t, err)
			// TODO this is ugly, can lead to flaky tests and we could probably do it better
			time.Sleep(50 * time.Millisecond)

			err = rpcClient.Send(table.topic, table.data)
			assert.NoError(t, err)

			r := helpers.ShouldEventuallyReceive(t, subChan).(*nats.Msg)
			assert.Equal(t, table.data, r.Data)
			subs.Unsubscribe()
		})
	}
}

func TestNatsRPCClientBuildRequest(t *testing.T) {
	config := getConfig()
	sv := getServer()
	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)

	rt := route.NewRoute("sv", "svc", "method")
	ss := session.New(nil, true, "uid")
	data := []byte("data")
	id := uint(123)
	tables := []struct {
		name           string
		frontendServer bool
		rpcType        protos.RPCType
		route          *route.Route
		session        *session.Session
		msg            *message.Message
		expected       protos.Request
	}{
		{
			"test-frontend-request", true, protos.RPCType_Sys, rt, ss,
			&message.Message{Type: message.Request, ID: id, Data: data},
			protos.Request{
				Type: protos.RPCType_Sys,
				Msg: &protos.Msg{
					Route: rt.String(),
					Data:  data,
					Type:  protos.MsgType_MsgRequest,
					Id:    uint64(id),
				},
				FrontendID: sv.ID,
				Session: &protos.Session{
					Id:   ss.ID(),
					Uid:  ss.UID(),
					Data: ss.GetDataEncoded(),
				},
			},
		},
		{
			"test-rpc-sys-request", false, protos.RPCType_Sys, rt, ss,
			&message.Message{Type: message.Request, ID: id, Data: data},
			protos.Request{
				Type: protos.RPCType_Sys,
				Msg: &protos.Msg{
					Route: rt.String(),
					Data:  data,
					Type:  protos.MsgType_MsgRequest,
					Id:    uint64(id),
				},
				FrontendID: "",
				Session: &protos.Session{
					Id:   ss.ID(),
					Uid:  ss.UID(),
					Data: ss.GetDataEncoded(),
				},
			},
		},
		{
			"test-rpc-user-request", false, protos.RPCType_User, rt, ss,
			&message.Message{Type: message.Request, ID: id, Data: data},
			protos.Request{
				Type: protos.RPCType_User,
				Msg: &protos.Msg{
					Route: rt.String(),
					Data:  data,
					Type:  protos.MsgType_MsgRequest,
				},
				FrontendID: "",
			},
		},
		{
			"test-notify", false, protos.RPCType_Sys, rt, ss,
			&message.Message{Type: message.Notify, ID: id, Data: data},
			protos.Request{
				Type: protos.RPCType_Sys,
				Msg: &protos.Msg{
					Route: rt.String(),
					Data:  data,
					Type:  protos.MsgType_MsgNotify,
					Id:    0,
				},
				FrontendID: "",
				Session: &protos.Session{
					Id:   ss.ID(),
					Uid:  ss.UID(),
					Data: ss.GetDataEncoded(),
				},
			},
		},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			rpcClient.server.Frontend = table.frontendServer
			req, err := buildRequest(context.Background(), table.rpcType, table.route, table.session, table.msg, rpcClient.server)
			assert.NoError(t, err)
			assert.NotNil(t, req.Metadata)
			req.Metadata = nil
			assert.Equal(t, table.expected, req)
		})
	}
}

func TestNatsRPCClientCallShouldFailIfNotRunning(t *testing.T) {
	config := getConfig()
	sv := getServer()
	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	res, err := rpcClient.Call(context.Background(), protos.RPCType_Sys, nil, nil, nil, sv)
	assert.Equal(t, constants.ErrRPCClientNotInitialized, err)
	assert.Nil(t, res)
}

func TestNatsRPCClientCall(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	sv := getServer()
	defer s.Shutdown()
	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", s.Addr()))
	cfg.Set("pitaya.cluster.rpc.client.nats.requesttimeout", "300ms")
	config := getConfig(cfg)
	rpcClient, _ := NewNatsRPCClient(config, sv, nil, nil)
	rpcClient.Init()

	rt := route.NewRoute("sv", "svc", "method")
	ss := session.New(nil, true, "uid")

	msg := &message.Message{
		Type: message.Request,
		ID:   uint(123),
		Data: []byte("data"),
	}

	tables := []struct {
		name     string
		response interface{}
		expected *protos.Response
		err      error
	}{
		{"test_error", &protos.Response{Data: []byte("nok"), Error: &protos.Error{Msg: "nok"}}, nil, e.NewError(errors.New("nok"), e.ErrUnknownCode)},
		{"test_ok", &protos.Response{Data: []byte("ok")}, &protos.Response{Data: []byte("ok")}, nil},
		{"test_bad_response", []byte("invalid"), nil, errors.New("cannot parse invalid wire-format data")},
		{"test_bad_proto", &protos.Session{Id: 1, Uid: "snap"}, nil, errors.New("cannot parse invalid wire-format data")},
		{"test_no_response", nil, nil, errors.New("nats: timeout")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
			assert.NoError(t, err)

			sv2 := getServer()
			sv2.Type = uuid.New().String()
			sv2.ID = uuid.New().String()
			subs, err := conn.Subscribe(getChannel(sv2.Type, sv2.ID), func(m *nats.Msg) {
				if table.response != nil {
					if val, ok := table.response.(*protos.Response); ok {
						b, _ := proto.Marshal(val)
						conn.Publish(m.Reply, b)
					} else if val, ok := table.response.(*protos.Session); ok {
						b, _ := proto.Marshal(val)
						conn.Publish(m.Reply, b)
					} else {
						conn.Publish(m.Reply, table.response.([]byte))
					}
				}
			})
			assert.NoError(t, err)
			// TODO this is ugly, can lead to flaky tests and we could probably do it better
			time.Sleep(50 * time.Millisecond)
			res, err := rpcClient.Call(context.Background(), protos.RPCType_Sys, rt, ss, msg, sv2)
			assert.Equal(t, table.expected, res)
                        if err != nil {
			  assert.Contains(t, err.Error(), table.err.Error())
                        }
			err = subs.Unsubscribe()
			assert.NoError(t, err)
			conn.Close()
		})
	}
}
