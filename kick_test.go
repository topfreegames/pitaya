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
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func TestSendKickToUsersLocalSession(t *testing.T) {
	tables := []struct {
		name         string
		uid1         string
		uid2         string
		frontendType string
		err          error
	}{
		{"success", uuid.New().String(), uuid.New().String(), "connector", nil},
		{"fail", uuid.New().String(), uuid.New().String(), "connector", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)

			s1 := session.New(mockNetworkEntity, true)
			err := s1.Bind(context.Background(), table.uid1)
			assert.NoError(t, err)

			s2 := session.New(mockNetworkEntity, true)
			err = s2.Bind(context.Background(), table.uid2)
			assert.NoError(t, err)

			mockNetworkEntity.EXPECT().Kick(context.Background()).Times(2).Return(table.err)
			if table.err == nil {
				mockNetworkEntity.EXPECT().Close().Times(2)
			}
			err = SendKickToUsers([]string{table.uid1, table.uid2}, table.frontendType)

			assert.NoError(t, err)
		})
	}
}

func TestSendKickToUsersRemoteSession(t *testing.T) {
	tables := []struct {
		name         string
		uids         []string
		frontendType string
		err          error
	}{
		{"success", []string{uuid.New().String(), uuid.New().String()}, "connector", nil},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			app.rpcClient = mockRPCClient

			for _, uid := range table.uids {
				expectedKick := &protos.KickMsg{UserId: uid}
				mockRPCClient.EXPECT().SendKick(uid, gomock.Any(), expectedKick)
			}

			err := SendKickToUsers(table.uids, table.frontendType)
			assert.NoError(t, err)
		})
	}
}
