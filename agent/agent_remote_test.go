// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

package agent

import (
	"context"
	"errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/cluster"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	codecmocks "github.com/topfreegames/pitaya/conn/codec/mocks"
	"github.com/topfreegames/pitaya/conn/message"
	messagemocks "github.com/topfreegames/pitaya/conn/message/mocks"
	"github.com/topfreegames/pitaya/conn/packet"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
)

type someStruct struct {
	A string
}

func TestNewRemote(t *testing.T) {
	uid := uuid.New().String()
	ss := &protos.Session{Uid: uid}
	reply := uuid.New().String()
	frontendID := uuid.New().String()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
	mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)

	remote, err := NewRemote(ss, reply, mockRPCClient, mockEncoder, mockSerializer, mockSD, frontendID, mockMessageEncoder)
	assert.NoError(t, err)
	assert.NotNil(t, remote)
	assert.IsType(t, make(chan struct{}), remote.chDie)
	assert.Equal(t, reply, remote.reply)
	assert.Equal(t, mockSerializer, remote.serializer)
	assert.Equal(t, mockEncoder, remote.encoder)
	assert.Equal(t, mockRPCClient, remote.rpcClient)
	assert.Equal(t, mockSD, remote.serviceDiscovery)
	assert.Equal(t, frontendID, remote.frontendID)
	assert.NotNil(t, remote.Session)
	assert.False(t, remote.Session.IsFrontend)
}

func TestNewRemoteFailsIfFailedToSetEncodedData(t *testing.T) {
	ss := &protos.Session{Data: []byte("invalid")}

	remote, err := NewRemote(ss, "", nil, nil, nil, nil, "", nil)
	assert.Equal(t, errors.New("invalid character 'i' looking for beginning of value").Error(), err.Error())
	assert.Nil(t, remote)
}

func TestAgentRemoteClose(t *testing.T) {
	remote, err := NewRemote(nil, "", nil, nil, nil, nil, "", nil)
	assert.NoError(t, err)
	assert.NotNil(t, remote)
	err = remote.Close()
	assert.NoError(t, err)
}

func TestAgentRemoteRemoteAddr(t *testing.T) {
	remote, err := NewRemote(nil, "", nil, nil, nil, nil, "", nil)
	assert.NoError(t, err)
	assert.NotNil(t, remote)
	addr := remote.RemoteAddr()
	assert.Nil(t, addr)
}

func TestAgentRemotePush(t *testing.T) {
	route := uuid.New().String()
	tables := []struct {
		name         string
		uid          string
		rpcClient    cluster.RPCClient
		data         interface{}
		errSerialize error
		err          error
	}{
		{"nats_rpc_session_not_bound", "", &cluster.NatsRPCClient{}, nil, nil, constants.ErrNoUIDBind},
		{"success_raw_message", uuid.New().String(), nil, []byte("ok"), nil, nil},
		{"failed_struct_message_serialize", uuid.New().String(), nil, &someStruct{A: "ok"}, errors.New("failed serialize"), errors.New("failed serialize")},
		{"success_struct_message", uuid.New().String(), nil, &someStruct{A: "ok"}, nil, nil},
		{"failed_send", uuid.New().String(), nil, []byte("ok"), nil, errors.New("failed send")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if table.rpcClient == nil {
				table.rpcClient = clustermocks.NewMockRPCClient(ctrl)
			}
			fSvID := "123id"
			ss := &protos.Session{Uid: table.uid}
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			remote, err := NewRemote(ss, "", table.rpcClient, nil, mockSerializer, mockSD, fSvID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, remote)

			if table.uid != "" {
				expectedData := []byte("done")

				if reflect.TypeOf(table.data) == reflect.TypeOf(([]byte)(nil)) {
					expectedData = table.data.([]byte)
				} else {
					mockSerializer.EXPECT().Marshal(table.data).Return(expectedData, table.errSerialize)
				}

				if table.errSerialize == nil {
					expectedPush := &protos.Push{
						Route: route,
						Uid:   table.uid,
						Data:  expectedData,
					}
					table.rpcClient.(*clustermocks.MockRPCClient).EXPECT().SendPush(table.uid, gomock.Any(), expectedPush).Return(table.err)
				}
			}

			if table.err != constants.ErrNoUIDBind {
				mockSD.EXPECT().GetServer(fSvID).Return(cluster.NewServer(fSvID, "connector", true), nil)
			}
			err = remote.Push(route, table.data)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestKickRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rpcClient := clustermocks.NewMockRPCClient(ctrl)
	ss := &protos.Session{Uid: uuid.New().String()}
	mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	frontID := uuid.New().String()
	remote, err := NewRemote(ss, "", rpcClient, nil, mockSerializer, mockSD, frontID, nil)
	assert.NoError(t, err)

	mockSD.EXPECT().GetServer(frontID)
	c := context.Background()
	r, _ := route.Decode("sys.kick")
	rpcClient.EXPECT().Call(c, protos.RPCType_User, r, gomock.Nil(), gomock.Any(), gomock.Nil())
	err = remote.Kick(c)

	assert.NoError(t, err)
}

func TestAgentRemoteResponseMID(t *testing.T) {
	tables := []struct {
		name         string
		mid          uint
		data         interface{}
		msgErr       bool
		errEncode    error
		errSerialize error
		err          error
	}{
		{"success_raw_message", uint(rand.Int()), []byte("ok"), false, nil, nil, nil},
		{"success_struct_message", uint(rand.Int()), &someStruct{A: "ok"}, false, nil, nil, nil},
		{"success_struct_message_with_error", uint(rand.Int()), &someStruct{A: "ok"}, true, nil, nil, nil},
		{"failed_struct_message_serialize", uint(rand.Int()), &someStruct{A: "ok"}, false, nil, errors.New("failed serialize"), errors.New("failed serialize")},
		{"failed_encode", uint(rand.Int()), &someStruct{A: "ok"}, false, errors.New("failed encode"), nil, errors.New("failed encode")},
		{"failed_send", uint(rand.Int()), &someStruct{A: "ok"}, false, nil, nil, errors.New("failed send")},
		{"zero_mid", 0, nil, false, nil, nil, constants.ErrSessionOnNotify},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reply := uuid.New().String()
			uid := uuid.New().String()
			ss := &protos.Session{Uid: uid}
			mockEnconder := codecmocks.NewMockPacketEncoder(ctrl)
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			messageEncoder := message.NewMessagesEncoder(false)
			remote, err := NewRemote(ss, reply, mockRPCClient, mockEnconder, mockSerializer, nil, "", messageEncoder)
			assert.NoError(t, err)
			assert.NotNil(t, remote)

			if table.mid != uint(0) {
				serializeRet := []byte("ok")
				if reflect.TypeOf(table.data) == reflect.TypeOf(([]byte)(nil)) {
					serializeRet = table.data.([]byte)
				} else {
					mockSerializer.EXPECT().Marshal(table.data).Return(serializeRet, table.errSerialize)
				}

				if table.errSerialize == nil {
					rawMsg := &message.Message{
						Type: message.Response,
						Data: serializeRet,
						ID:   table.mid,
						Err:  table.msgErr,
					}
					expectedMsg, _ := messageEncoder.Encode(rawMsg)
					mockEnconder.EXPECT().Encode(gomock.Any(), expectedMsg).Return(nil, table.errEncode).Do(
						func(typ packet.Type, d []byte) {
							// cannot compare inside the expect because they are equivalent but not equal
							assert.EqualValues(t, packet.Data, typ)
						})

					if table.errEncode == nil {
						mockRPCClient.EXPECT().Send(reply, gomock.Any()).Return(table.err)
					}
				}

			}
			if table.msgErr {
				err = remote.ResponseMID(nil, table.mid, table.data, table.msgErr)
			} else {
				err = remote.ResponseMID(nil, table.mid, table.data)
			}
			assert.Equal(t, table.err, err)
		})
	}
}

func TestAgentRemoteSendRequest(t *testing.T) {
	tables := []struct {
		name         string
		serverID     string
		reqRoute     string
		data         interface{}
		errSerialize error
		errGetServer error
		err          error
		resp         *protos.Response
	}{
		{"test_failed_bad_route", uuid.New().String(), uuid.New().String(), []byte("ok"), nil, nil, errors.New("invalid route"), nil},
		{"test_success_raw", uuid.New().String(), "", []byte("ok"), nil, nil, nil, &protos.Response{Data: []byte("resp")}},
		{"test_success_struct", uuid.New().String(), "", &someStruct{A: "ok"}, nil, nil, nil, &protos.Response{Data: []byte("resp")}},
		{"test_failed_serialize", uuid.New().String(), "", &someStruct{A: "ok"}, errors.New("ser"), nil, errors.New("ser"), nil},
		{"test_failed_get_server", uuid.New().String(), "", &someStruct{A: "ok"}, nil, errors.New("get sv"), errors.New("get sv"), nil},
		{"test_failed_call", uuid.New().String(), "", &someStruct{A: "ok"}, nil, nil, errors.New("call"), nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
			remote, err := NewRemote(nil, "", mockRPCClient, nil, mockSerializer, mockSD, "", mockMessageEncoder)
			assert.NoError(t, err)
			assert.NotNil(t, remote)

			if table.reqRoute == "" {
				table.reqRoute = "bla.bla"

				serializeRet := []byte("ok")
				if reflect.TypeOf(table.data) == reflect.TypeOf(([]byte)(nil)) {
					serializeRet = table.data.([]byte)
				} else {
					mockSerializer.EXPECT().Marshal(table.data).Return(serializeRet, table.errSerialize)
				}

				if table.errSerialize == nil {
					expectedServer := &cluster.Server{}
					mockSD.EXPECT().GetServer(table.serverID).Return(expectedServer, table.errGetServer)

					if table.errGetServer == nil {
						r, _ := route.Decode(table.reqRoute)
						expectedMsg := &message.Message{
							Route: table.reqRoute,
							Data:  serializeRet,
						}
						mockRPCClient.EXPECT().Call(nil, protos.RPCType_User, r, nil, expectedMsg, expectedServer).Return(table.resp, table.err)
					}
				}
			}

			resp, err := remote.SendRequest(nil, table.serverID, table.reqRoute, table.data)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.resp, resp)
		})
	}
}
