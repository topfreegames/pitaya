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
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/cluster"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func (m *MyComp) Remote1() ([]byte, error) {
	return nil, nil
}
func (m *MyComp) Remote2() (*TestType, error) {
	return nil, nil
}

func TestNewRemoteService(t *testing.T) {
	packetEncoder := codec.NewPomeloPacketEncoder()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
	mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
	mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
	router := router.New()
	svc := NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router)

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
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	defer func() { remotes = make(map[string]*component.Remote, 0) }()
	assert.Len(t, svc.services, 1)
	val, ok := svc.services["MyComp"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	assert.Len(t, remotes, 2)
	val2, ok := remotes["MyComp.Remote1"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = remotes["MyComp.Remote2"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
}

func TestRemoteServiceRegisterFailsIfRegisterTwice(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	err = svc.Register(&MyComp{}, []component.Option{})
	assert.Contains(t, err.Error(), "remote: service already defined")
}

func TestRemoteServiceRegisterFailsIfNoRemoteMethods(t *testing.T) {
	svc := NewRemoteService(nil, nil, nil, nil, nil, nil)
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

	svc := NewRemoteService(nil, mockRPCServer, nil, nil, nil, nil)
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
	svc := NewRemoteService(mockRPCClient, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)

	reply := uuid.New().String()
	resp := &protos.Response{Data: []byte(uuid.New().String()), Error: uuid.New().String()}
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
		{"no_target_route_error", nil, nil, constants.ErrServiceDiscoveryNotInitialized},
		{"error", sv, nil, errors.New("ble")},
		{"success", sv, &protos.Response{Data: []byte("ok")}, nil},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			svc := NewRemoteService(mockRPCClient, nil, nil, nil, nil, nil)
			assert.NotNil(t, svc)

			msg := &message.Message{}
			if sv != nil {
				mockRPCClient.EXPECT().Call(protos.RPCType_Sys, rt, ss, msg, sv).Return(table.res, table.err)
			}
			res, err := svc.remoteCall(sv, protos.RPCType_Sys, rt, ss, msg)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.res, res)
		})
	}
}

func TestRemoteServiceProcessRemoteMessages(t *testing.T) {
	// TODO
}

func TestRemoteServiceHandleRPCUser(t *testing.T) {
	// TODO
}

func TestRemoteServiceHandleRPCSys(t *testing.T) {
	// TODO
}

func TestRemoteServiceRemoteProcess(t *testing.T) {
	// TODO
}

func TestRemoteServiceRPC(t *testing.T) {
	// TODO
}
