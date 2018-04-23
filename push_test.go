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

package pitaya

import (
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/cluster"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	"github.com/topfreegames/pitaya/protos"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

type someStruct struct {
	A int
}

func TestSendToUsersFailsIfErrSerializing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	app.serializer = mockSerializer

	route := "some.route.bla"
	data := &someStruct{A: 10}
	uid := uuid.New().String()
	expectedErr := errors.New("serialize error")
	mockSerializer.EXPECT().Marshal(data).Return(nil, expectedErr)

	err := SendToUsers(route, data, []string{uid})
	assert.Equal(t, expectedErr, err)
}

func TestSendToUsersLocalSession(t *testing.T) {
	tables := []struct {
		name string
		err  error
	}{
		{"successful_request", nil},
		{"failed_request", errors.New("failed push")},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)

			route := "some.route.bla"
			data := []byte("hello")
			uid1 := uuid.New().String()
			uid2 := uuid.New().String()

			s1 := session.New(mockNetworkEntity, true)
			err := s1.Bind(uid1)
			assert.NoError(t, err)
			s2 := session.New(mockNetworkEntity, true)
			err = s2.Bind(uid2)
			assert.NoError(t, err)

			mockNetworkEntity.EXPECT().Push(route, data).Return(table.err).Times(2)
			err = SendToUsers(route, data, []string{uid1, uid2})
			assert.NoError(t, err)
		})
	}
}

func TestSendToUsersRemoteSession(t *testing.T) {
	tables := []struct {
		name string
		err  error
	}{
		{"successful_request", nil},
		{"failed_request", errors.New("failed send")},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			app.rpcClient = mockRPCClient

			route := "some.route.bla"
			data := []byte("hello")
			uid1 := uuid.New().String()
			uid2 := uuid.New().String()

			topic1 := cluster.GetUserMessagesTopic(uid1)
			topic2 := cluster.GetUserMessagesTopic(uid2)
			expectedMsg1, _ := proto.Marshal(&protos.Push{
				Route: route,
				Uid:   uid1,
				Data:  data,
			})
			expectedMsg2, _ := proto.Marshal(&protos.Push{
				Route: route,
				Uid:   uid2,
				Data:  data,
			})
			mockRPCClient.EXPECT().Send(topic1, expectedMsg1).Return(table.err)
			mockRPCClient.EXPECT().Send(topic2, expectedMsg2).Return(table.err)
			err := SendToUsers(route, data, []string{uid1, uid2})
			assert.NoError(t, err)
		})
	}
}
