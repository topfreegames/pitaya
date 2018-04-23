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

package service

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	connmock "github.com/topfreegames/pitaya/mocks"
	messagemocks "github.com/topfreegames/pitaya/internal/message/mocks"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
	"github.com/topfreegames/pitaya/util"
)

func (m *MyComp) Remote1(ss *SomeStruct) ([]byte, error) {
	return []byte("ok"), nil
}

func (m *MyComp) Remote2() (*SomeStruct, error) {
	return nil, nil
}

func (m *MyComp) RemoteRes(b []byte) ([]byte, error) {
	return b, nil
}

func (m *MyComp) RemoteErr() ([]byte, error) {
	return nil, e.NewError(errors.New("remote err"), e.ErrUnknownCode)
}

type unregisteredStruct struct{}

func TestNewRemoteService(t *testing.T) {
	packetEncoder := codec.NewPomeloPacketEncoder()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
	mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
	mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
	mockMessageEncoder := messagemocks.NewMockMessageEncoder(ctrl)
	router := router.New()
	svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, mockMessageEncoder)

	assert.NotNil(t, svc)
	assert.Empty(t, svc.services)
	assert.Equal(t, mockRPCClient, svc.rpcClient)
	assert.Equal(t, mockRPCServer, svc.rpcServer)
	assert.Equal(t, packetEncoder, svc.encoder)
	assert.Equal(t, mockSD, svc.serviceDiscovery)
	assert.Equal(t, mockSerializer, svc.serializer)
	assert.Equal(t, router, svc.router)
}

func TestRemoteServiceRegister(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	defer func() { remotes = make(map[string]*component.Remote, 0) }()
	assert.Len(t, svc.services, 1)
	val, ok := svc.services["MyComp"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	assert.Len(t, remotes, 4)
	val2, ok := remotes["MyComp.Remote1"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = remotes["MyComp.Remote2"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = remotes["MyComp.RemoteErr"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	val2, ok = remotes["MyComp.RemoteRes"]
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestRemoteServiceRegisterFailsIfRegisterTwice(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	err = svc.Register(&MyComp{}, []component.Option{})
	assert.Contains(t, err.Error(), "remote: service already defined")
}

func TestRemoteServiceRegisterFailsIfNoRemoteMethods(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&NoHandlerRemoteComp{}, []component.Option{})
	assert.Equal(t, errors.New("type NoHandlerRemoteComp has no exported methods of suitable type"), err)
}

func TestRemoteServiceProcessUserPush(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
	mockEntity := mocks.NewMockNetworkEntity(ctrl)

	uid := uuid.New().String()
	push := &protos.Push{Route: "bla.bla", Uid: uid, Data: []byte("data")}
	userPushCh := make(chan *protos.Push)

	ss := session.New(mockEntity, true)
	err := ss.Bind(uid)
	assert.NoError(t, err)

	svc := NewRemoteService(nil, mockRPCServer, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
	mockRPCServer.EXPECT().GetUserPushChannel().Return(userPushCh)

	called := false
	mockEntity.EXPECT().Push(push.Route, push.Data).Do(func(route string, data []byte) {
		called = true
	})
	go svc.ProcessUserPush()
	userPushCh <- push
	helpers.ShouldEventuallyReturn(t, func() bool { return called == true }, true)
}

func TestRemoteServiceSendReply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
	svc := NewRemoteService(mockRPCClient, nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)

	reply := uuid.New().String()
	resp := &protos.Response{Data: []byte(uuid.New().String()), Error: &protos.Error{Msg: uuid.New().String()}}
	p, _ := proto.Marshal(resp)
	mockRPCClient.EXPECT().Send(reply, p)
	svc.sendReply(reply, resp)
}

func TestRemoteServiceRemoteCall(t *testing.T) {
	ss := session.New(nil, true)
	rt := route.NewRoute("sv", "svc", "method")
	sv := &cluster.Server{}
	tables := []struct {
		name   string
		server *cluster.Server
		res    *protos.Response
		err    error
	}{
		{"no_target_route_error", nil, nil, e.NewError(constants.ErrServiceDiscoveryNotInitialized, e.ErrInternalCode)},
		{"error", sv, nil, errors.New("ble")},
		{"success", sv, &protos.Response{Data: []byte("ok")}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, nil, nil, nil, nil, router, nil)
			assert.NotNil(t, svc)

			msg := &message.Message{}
			if table.server != nil {
				mockRPCClient.EXPECT().Call(protos.RPCType_Sys, rt, ss, msg, sv).Return(table.res, table.err)
			}
			res, err := svc.remoteCall(table.server, protos.RPCType_Sys, rt, ss, msg)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.res, res)
		})
	}
}

func TestRemoteServiceHandleRPCUser(t *testing.T) {
	tObj := &MyComp{}
	m, ok := reflect.TypeOf(tObj).MethodByName("Remote1")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	remotes[rt.Short()] = &component.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 1}
	m, ok = reflect.TypeOf(tObj).MethodByName("RemoteErr")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtErr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	remotes[rtErr.Short()] = &component.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 1}
	m, ok = reflect.TypeOf(tObj).MethodByName("Remote2")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtStr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	remotes[rtStr.Short()] = &component.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 1}
	m, ok = reflect.TypeOf(tObj).MethodByName("RemoteRes")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtRes := route.NewRoute("", uuid.New().String(), uuid.New().String())
	remotes[rtRes.Short()] = &component.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 1}

	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode([]interface{}{[]byte("ok")})
	assert.NoError(t, err)
	tables := []struct {
		name         string
		req          *protos.Request
		rt           *route.Route
		errSubstring string
	}{
		{"remote_not_found", &protos.Request{Msg: &protos.Msg{}}, route.NewRoute("bla", "bla", "bla"), "route not found"},
		{"failed_unmarshal", &protos.Request{Msg: &protos.Msg{Data: []byte("dd")}}, rt, "unexpected EOF"},
		{"failed_pcall", &protos.Request{Msg: &protos.Msg{}}, rtErr, "remote err"},
		{"success_nil_response", &protos.Request{Msg: &protos.Msg{}}, rtStr, ""},
		{"success_response", &protos.Request{Msg: &protos.Msg{Data: buf.Bytes()}}, rtRes, ""},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
			messageEncoder := message.NewEncoder(false)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder)
			assert.NotNil(t, svc)

			mockRPCClient.EXPECT().Send(table.req.Msg.Reply, gomock.Any()).Do(func(r string, data []byte) {
				res := &protos.Response{}
				err := proto.Unmarshal(data, res)
				assert.NoError(t, err)

				if table.errSubstring != "" {
					assert.Contains(t, res.Error.Msg, table.errSubstring)
				} else if table.req.Msg.Data != nil {
					assert.NotNil(t, res.Data)
				}
			})
			svc.handleRPCUser(table.req, table.rt)
		})
	}
}

func TestRemoteServiceHandleRPCSys(t *testing.T) {
	tObj := &TestType{}
	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	tables := []struct {
		name         string
		req          *protos.Request
		rt           *route.Route
		errSubstring string
	}{
		{"new_remote_err", &protos.Request{
			Msg:     &protos.Msg{Reply: uuid.New().String()},
			Session: &protos.Session{Data: []byte("no")},
		}, nil, "unexpected EOF"},
		{"process_handler_msg_err", &protos.Request{Msg: &protos.Msg{Reply: uuid.New().String()}}, route.NewRoute("bla", "bla", "bla"), "bla.bla.bla not found"},
		{"success", &protos.Request{Msg: &protos.Msg{Data: []byte("ok")}}, rt, ""},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
			messageEncoder := message.NewEncoder(false)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder)
			assert.NotNil(t, svc)

			if table.errSubstring == "" {
				mockSerializer.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).Return(nil)
			}
			mockRPCClient.EXPECT().Send(table.req.Msg.Reply, gomock.Any()).Do(func(r string, data []byte) {
				res := &protos.Response{}
				err := proto.Unmarshal(data, res)
				assert.NoError(t, err)

				if table.errSubstring != "" {
					assert.Contains(t, res.Error.Msg, table.errSubstring)
				} else {
					assert.Equal(t, table.req.Msg.Data, res.Data)
				}
			})
			svc.handleRPCSys(table.req, table.rt)
		})
	}
}

func TestRemoteServiceRemoteProcess(t *testing.T) {
	sv := &cluster.Server{}
	rt := route.NewRoute("sv", "svc", "method")

	tables := []struct {
		name           string
		msgType        message.Type
		remoteCallErr  error
		responseMIDErr bool
	}{
		{"failed_remote_call", message.Request, errors.New("rpc failed"), false},
		{"failed_response_mid", message.Request, nil, true},
		{"success_request", message.Request, nil, false},
		{"success_notify", message.Notify, nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
			messageEncoder := message.NewEncoder(false)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder)
			assert.NotNil(t, svc)

			expectedMsg := &message.Message{
				ID:    uint(rand.Int()),
				Type:  message.Request,
				Route: rt.Short(),
				Data:  []byte("ok"),
			}
			mockRPCClient.EXPECT().Call(protos.RPCType_Sys, rt, gomock.Any(), expectedMsg, gomock.Any()).Return(&protos.Response{Data: []byte("ok")}, table.remoteCallErr)

			if table.remoteCallErr != nil {
				mockSerializer.EXPECT().Marshal(gomock.Any()).Return([]byte("err"), nil)
			}

			encoder := codec.NewPomeloPacketEncoder()
			mockConn := connmock.NewMockConn(ctrl)
			ag := agent.NewAgent(mockConn, nil, encoder, mockSerializer, 1*time.Second, 1, nil, messageEncoder)

			if table.responseMIDErr {
				ag.SetStatus(constants.StatusClosed)
				mockSerializer.EXPECT().Marshal(gomock.Any()).Return([]byte("err"), nil)
			}
			svc.remoteProcess(sv, ag, rt, expectedMsg)
		})
	}
}

func TestRemoteServiceRPC(t *testing.T) {
	rt := route.NewRoute("sv", "svc", "method")
	tables := []struct {
		name        string
		serverID    string
		reply       interface{}
		args        []interface{}
		foundServer bool
		err         error
	}{
		{"bad_args", "", nil, []interface{}{&unregisteredStruct{}}, false, errors.New("gob: type not registered for interface: service.unregisteredStruct")},
		{"server_id_and_no_target", "serverId", nil, []interface{}{&SomeStruct{}}, false, constants.ErrServerNotFound},
		{"failed_remote_call", "serverId", nil, []interface{}{&SomeStruct{}}, true, errors.New("rpc failed")},
		{"success", "serverId", &SomeStruct{}, []interface{}{&SomeStruct{}}, true, nil},
		{"success_nil_reply", "serverId", nil, []interface{}{&SomeStruct{}}, true, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
			messageEncoder := message.NewEncoder(false)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder)
			assert.NotNil(t, svc)

			if table.serverID != "" {
				var sdRet *cluster.Server
				if table.foundServer {
					sdRet = &cluster.Server{}
				}
				mockSD.EXPECT().GetServer(table.serverID).Return(sdRet, nil)
			}

			var expected *SomeStruct
			if table.foundServer {
				expectedData, _ := util.GobEncode(table.args...)
				expectedMsg := &message.Message{
					Type:  message.Request,
					Route: rt.Short(),
					Data:  expectedData,
				}

				expected = &SomeStruct{A: 1, B: "one"}
				buf := bytes.NewBuffer(nil)
				err := gob.NewEncoder(buf).Encode(expected)
				assert.NoError(t, err)
				mockRPCClient.EXPECT().Call(protos.RPCType_User, rt, gomock.Any(), expectedMsg, gomock.Any()).Return(&protos.Response{Data: buf.Bytes()}, table.err)
			}
			err := svc.RPC(table.serverID, rt, table.reply, table.args...)
			assert.Equal(t, table.err, err)
			if table.reply != nil {
				assert.Equal(t, table.reply, expected)
			}
		})
	}
}
