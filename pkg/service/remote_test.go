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
	"context"
	"errors"
	agentmocks "github.com/topfreegames/pitaya/v3/pkg/agent/mocks"
	"github.com/topfreegames/pitaya/v3/pkg/cluster"
	"github.com/topfreegames/pitaya/v3/pkg/cluster/mocks"
	component2 "github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/conn/codec"
	message2 "github.com/topfreegames/pitaya/v3/pkg/conn/message"
	messagemocks "github.com/topfreegames/pitaya/v3/pkg/conn/message/mocks"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
	e "github.com/topfreegames/pitaya/v3/pkg/errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/pkg/pipeline"
	"github.com/topfreegames/pitaya/v3/pkg/protos"
	"github.com/topfreegames/pitaya/v3/pkg/protos/test"
	"github.com/topfreegames/pitaya/v3/pkg/route"
	"github.com/topfreegames/pitaya/v3/pkg/router"
	serializemocks "github.com/topfreegames/pitaya/v3/pkg/serialize/mocks"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	sessionmocks "github.com/topfreegames/pitaya/v3/pkg/session/mocks"
)

func (m *MyComp) Remote1(ctx context.Context, ss *test.SomeStruct) (*test.SomeStruct, error) {
	return &test.SomeStruct{B: "ack"}, nil
}

func (m *MyComp) Remote2(ctx context.Context) (*test.SomeStruct, error) {
	return nil, nil
}

func (m *MyComp) RemoteRes(ctx context.Context, b *test.SomeStruct) (*test.SomeStruct, error) {
	return b, nil
}

func (m *MyComp) RemoteErr(ctx context.Context) (*test.SomeStruct, error) {
	return nil, e.NewError(errors.New("remote err"), e.ErrUnknownCode)
}

type unregisteredStruct struct{}

func TestNewRemoteService(t *testing.T) {
	packetEncoder := codec.NewPomeloPacketEncoder()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockSD := mocks.NewMockServiceDiscovery(ctrl)
	mockRPCClient := mocks.NewMockRPCClient(ctrl)
	mockRPCServer := mocks.NewMockRPCServer(ctrl)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	router := router.New()
	sv := &cluster.Server{}
	sessionPool := session.NewSessionPool()
	handlerHooks := pipeline.NewHandlerHooks()
	handlerPool := NewHandlerPool()
	svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, mockMessageEncoder, sv, sessionPool, handlerHooks, handlerPool)

	assert.NotNil(t, svc)
	assert.Empty(t, svc.services)
	assert.Equal(t, mockRPCClient, svc.rpcClient)
	assert.Equal(t, mockRPCServer, svc.rpcServer)
	assert.Equal(t, packetEncoder, svc.encoder)
	assert.Equal(t, mockSD, svc.serviceDiscovery)
	assert.Equal(t, mockSerializer, svc.serializer)
	assert.Equal(t, router, svc.router)
	assert.Equal(t, sv, svc.server)
	assert.Equal(t, sessionPool, svc.sessionPool)
	assert.Equal(t, handlerHooks, svc.handlerHooks)
	assert.Equal(t, handlerPool, svc.handlerPool)
}

func TestRemoteServiceRegister(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component2.Option{})
	assert.NoError(t, err)
	defer func() { svc.remotes = make(map[string]*component2.Remote, 0) }()
	assert.Len(t, svc.services, 1)
	val, ok := svc.services["MyComp"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	val2, ok := svc.remotes["MyComp.Remote1"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = svc.remotes["MyComp.Remote2"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = svc.remotes["MyComp.RemoteErr"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	val2, ok = svc.remotes["MyComp.RemoteRes"]
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestRemoteServiceAddRemoteBindingListener(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBindingListener := mocks.NewMockRemoteBindingListener(ctrl)

	svc.AddRemoteBindingListener(mockBindingListener)
	assert.Equal(t, mockBindingListener, svc.remoteBindingListeners[0])
}

func TestRemoteServiceSessionBindRemote(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBindingListener := mocks.NewMockRemoteBindingListener(ctrl)

	svc.AddRemoteBindingListener(mockBindingListener)
	assert.Equal(t, mockBindingListener, svc.remoteBindingListeners[0])

	msg := &protos.BindMsg{
		Uid: "uid",
		Fid: "fid",
	}

	mockBindingListener.EXPECT().OnUserBind(msg.Uid, msg.Fid)

	_, err := svc.SessionBindRemote(context.Background(), msg)

	assert.NoError(t, err)
}

func TestRemoteServicePushToUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	existingUID := "uid1"
	nonexistingUID := "uid2"

	mockSession := sessionmocks.NewMockSession(ctrl)

	mockSessionPool := sessionmocks.NewMockSessionPool(ctrl)
	mockSessionPool.EXPECT().GetSessionByUID(existingUID).Return(mockSession).Times(1)
	mockSessionPool.EXPECT().GetSessionByUID(nonexistingUID).Return(nil).Times(1)

	tables := []struct {
		name string
		uid  string
		sess session.Session
		p    *protos.Push
		err  error
	}{
		{"success", "uid1", mockSession, &protos.Push{
			Route: "sv.svc.mth",
			Uid:   existingUID,
			Data:  []byte{0x01},
		}, nil},
		{"no_sess_found", "uid2", nil, &protos.Push{
			Route: "sv.svc.mth",
			Uid:   nonexistingUID,
			Data:  []byte{0x01},
		}, constants.ErrSessionNotFound},
	}

	mockSession.EXPECT().Push(tables[0].p.Route, tables[0].p.Data).Times(1)
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, mockSessionPool, nil, nil)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			_, err := svc.PushToUser(context.Background(), table.p)
			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoteServiceKickUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSessionPool := sessionmocks.NewMockSessionPool(ctrl)
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, mockSessionPool, nil, nil)

	existingUID := "uid1"
	nonexistingUID := "uid2"

	mockSession := sessionmocks.NewMockSession(ctrl)
	mockSession.EXPECT().Kick(context.Background()).Times(1)

	mockSessionPool.EXPECT().GetSessionByUID(existingUID).Return(mockSession).Times(1)
	mockSessionPool.EXPECT().GetSessionByUID(nonexistingUID).Return(nil).Times(1)

	defer ctrl.Finish()

	tables := []struct {
		name string
		uid  string
		sess session.Session
		p    *protos.KickMsg
		err  error
	}{
		{"success", existingUID, mockSession, &protos.KickMsg{
			UserId: existingUID,
		}, nil},
		{"sessionNotFound", nonexistingUID, nil, &protos.KickMsg{
			UserId: nonexistingUID,
		}, constants.ErrSessionNotFound},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			_, err := svc.KickUser(context.Background(), table.p)
			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestRemoteServiceRegisterFailsIfRegisterTwice(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component2.Option{})
	assert.NoError(t, err)
	err = svc.Register(&MyComp{}, []component2.Option{})
	assert.Contains(t, err.Error(), "remote: service already defined")
}

func TestRemoteServiceRegisterFailsIfNoRemoteMethods(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	err := svc.Register(&NoHandlerRemoteComp{}, []component2.Option{})
	assert.Equal(t, errors.New("type NoHandlerRemoteComp has no exported methods of remote type"), err)
}

func TestRemoteServiceRemoteCall(t *testing.T) {
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
			mockSession := sessionmocks.NewMockSession(ctrl)
			mockRPCClient := mocks.NewMockRPCClient(ctrl)
			sessionPool := sessionmocks.NewMockSessionPool(ctrl)
			router := router.New()
			svc := NewRemoteService(mockRPCClient, nil, nil, nil, nil, router, nil, nil, sessionPool, pipeline.NewHandlerHooks(), nil)
			assert.NotNil(t, svc)

			msg := &message2.Message{}
			ctx := context.Background()
			if table.server != nil {
				mockRPCClient.EXPECT().Call(ctx, protos.RPCType_Sys, rt, mockSession, msg, sv).Return(table.res, table.err)
			}
			res, err := svc.remoteCall(ctx, table.server, protos.RPCType_Sys, rt, mockSession, msg)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.res, res)
		})
	}
}

func TestRemoteServiceHandleRPCUser(t *testing.T) {
	handlerPool := NewHandlerPool()

	tObj := &MyComp{}
	m, ok := reflect.TypeOf(tObj).MethodByName("Remote1")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	comp := &component2.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 2}

	m, ok = reflect.TypeOf(tObj).MethodByName("RemoteErr")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtErr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	compErr := &component2.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 2}

	m, ok = reflect.TypeOf(tObj).MethodByName("Remote2")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtStr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	compStr := &component2.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 2}

	m, ok = reflect.TypeOf(tObj).MethodByName("RemoteRes")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtRes := route.NewRoute("", uuid.New().String(), uuid.New().String())
	compRes := &component2.Remote{Receiver: reflect.ValueOf(tObj), Method: m, HasArgs: m.Type.NumIn() > 2, Type: reflect.TypeOf(&test.SomeStruct{B: "aa"})}

	b, err := proto.Marshal(&test.SomeStruct{B: "aa"})
	assert.NoError(t, err)
	tables := []struct {
		name         string
		req          *protos.Request
		rt           *route.Route
		errSubstring string
	}{
		{"remote_not_found", &protos.Request{Msg: &protos.Msg{}}, route.NewRoute("bla", "bla", "bla"), "route not found"},
		{"failed_unmarshal", &protos.Request{Msg: &protos.Msg{Data: []byte("dd")}}, rt, "reflect: Call using zero Value argument"},
		{"failed_pcall", &protos.Request{Msg: &protos.Msg{}}, rtErr, "remote err"},
		{"success_nil_response", &protos.Request{Msg: &protos.Msg{}}, rtStr, ""},
		{"success_response", &protos.Request{Msg: &protos.Msg{Data: b}}, rtRes, ""},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := mocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := mocks.NewMockRPCClient(ctrl)
			mockRPCServer := mocks.NewMockRPCServer(ctrl)
			messageEncoder := message2.NewMessagesEncoder(false)
			router := router.New()
			sessionPool := session.NewSessionPool()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, &cluster.Server{}, sessionPool, pipeline.NewHandlerHooks(), handlerPool)

			svc.remotes[rt.Short()] = comp
			svc.remotes[rtErr.Short()] = compErr
			svc.remotes[rtStr.Short()] = compStr
			svc.remotes[rtRes.Short()] = compRes

			assert.NotNil(t, svc)
			res := svc.handleRPCUser(context.Background(), table.req, table.rt)
			assert.NoError(t, err)
			if table.errSubstring != "" {
				assert.Contains(t, res.Error.Msg, table.errSubstring)
			} else if table.req.Msg.Data != nil {
				assert.NotNil(t, res.Data)
			}
		})
	}
}

func TestRemoteServiceHandleRPCSys(t *testing.T) {
	tObj := &TestType{}
	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())

	tables := []struct {
		name         string
		req          *protos.Request
		rt           *route.Route
		errSubstring string
	}{
		{"new_remote_err", &protos.Request{
			Msg:     &protos.Msg{Reply: uuid.New().String()},
			Session: &protos.Session{Data: []byte("{no")},
		}, nil, "invalid character 'n' looking for beginning of object key string"},
		{"process_handler_msg_err", &protos.Request{Msg: &protos.Msg{Reply: uuid.New().String()}}, route.NewRoute("bla", "bla", "bla"), "bla.bla.bla not found"},
		{"success", &protos.Request{Msg: &protos.Msg{Data: []byte("ok")}}, rt, ""},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := mocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := mocks.NewMockRPCClient(ctrl)
			mockRPCServer := mocks.NewMockRPCServer(ctrl)
			messageEncoder := message2.NewMessagesEncoder(false)
			router := router.New()
			sessionPool := session.NewSessionPool()
			handlerPool := NewHandlerPool()
			handlerPool.handlers[rt.Short()] = &component2.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, &cluster.Server{}, sessionPool, pipeline.NewHandlerHooks(), handlerPool)
			assert.NotNil(t, svc)

			if table.errSubstring == "" {
				mockSerializer.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).Return(nil)
			}
			res := svc.handleRPCSys(nil, table.req, table.rt)

			if table.errSubstring != "" {
				assert.Contains(t, res.Error.Msg, table.errSubstring)
			} else {
				assert.Equal(t, table.req.Msg.Data, res.Data)
			}

		})
	}
}

func TestRemoteServiceRemoteProcess(t *testing.T) {
	sv := &cluster.Server{}
	rt := route.NewRoute("sv", "svc", "method")

	tables := []struct {
		name          string
		msgType       message2.Type
		remoteCallErr error
		responseMIDErr error
	}{
		{"failed_remote_call", message2.Request, errors.New("rpc failed"), nil},
		{"failed_response_mid", message2.Request, nil, errors.New("err")},
		{"success_request", message2.Request, nil, nil},
		{"success_notify", message2.Notify, nil, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			expectedMsg := &message2.Message{
				ID:    uint(rand.Int()),
				Type:  table.msgType,
				Route: rt.Short(),
				Data:  []byte("ok"),
			}
			ctx := context.Background()

			packetEncoder := codec.NewPomeloPacketEncoder()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := mocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := mocks.NewMockRPCClient(ctrl)
			mockRPCServer := mocks.NewMockRPCServer(ctrl)
			messageEncoder := message2.NewMessagesEncoder(false)
			router := router.New()
			sessionPool := session.NewSessionPool()
			mockSession := sessionmocks.NewMockSession(ctrl)

			mockAgent := agentmocks.NewMockAgent(ctrl)
			mockAgent.EXPECT().GetSession().Return(mockSession).AnyTimes()

			mockRPCClient.EXPECT().Call(ctx, protos.RPCType_Sys, rt, gomock.Any(), expectedMsg, gomock.Any()).Return(&protos.Response{Data: []byte("ok")}, table.remoteCallErr)

			if table.remoteCallErr != nil {
				mockAgent.EXPECT().AnswerWithError(ctx, expectedMsg.ID, gomock.Any())
			} else if expectedMsg.Type != message2.Notify {
				mockSession.EXPECT().ResponseMID(ctx, expectedMsg.ID, gomock.Any()).Return(table.responseMIDErr)
			}

			if table.responseMIDErr != nil {
				mockAgent.EXPECT().AnswerWithError(ctx, expectedMsg.ID, table.responseMIDErr)
			}

			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, &cluster.Server{}, sessionPool, pipeline.NewHandlerHooks(), nil)
			svc.remoteProcess(ctx, sv, mockAgent, rt, expectedMsg)
		})
	}
}

func TestRemoteServiceRPC(t *testing.T) {
	rt := route.NewRoute("sv", "svc", "method")
	tables := []struct {
		name        string
		serverID    string
		reply       proto.Message
		arg         proto.Message
		foundServer bool
		err         error
	}{
		{"server_id_and_no_target", "serverId", nil, &test.SomeStruct{}, false, constants.ErrServerNotFound},
		{"failed_remote_call", "serverId", nil, &test.SomeStruct{}, true, errors.New("rpc failed")},
		{"success", "serverId", &test.SomeStruct{}, &test.SomeStruct{}, true, nil},
		{"success_nil_reply", "serverId", nil, &test.SomeStruct{}, true, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := mocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := mocks.NewMockRPCClient(ctrl)
			mockRPCServer := mocks.NewMockRPCServer(ctrl)
			messageEncoder := message2.NewMessagesEncoder(false)
			router := router.New()
			sessionPool := session.NewSessionPool()
			svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, &cluster.Server{}, sessionPool, pipeline.NewHandlerHooks(), nil)
			assert.NotNil(t, svc)

			if table.serverID != "" {
				var sdRet *cluster.Server
				if table.foundServer {
					sdRet = &cluster.Server{}
				}
				mockSD.EXPECT().GetServer(table.serverID).Return(sdRet, nil)
			}

			var expected *test.SomeStruct
			ctx := context.Background()
			if table.foundServer {
				expectedData, _ := proto.Marshal(table.arg)
				expectedMsg := &message2.Message{
					Type:  message2.Request,
					Route: rt.Short(),
					Data:  expectedData,
				}

				expected = &test.SomeStruct{}
				b, err := proto.Marshal(expected)
				assert.NoError(t, err)
				mockRPCClient.EXPECT().Call(ctx, protos.RPCType_User, rt, gomock.Any(), expectedMsg, gomock.Any()).Return(&protos.Response{Data: b}, table.err)
			}
			err := svc.RPC(ctx, table.serverID, rt, table.reply, table.arg)
			assert.Equal(t, table.err, err)
			if table.reply != nil {
				assert.Equal(t, table.reply, expected)
			}
		})
	}
}
