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
	clustermocks "github.com/topfreegames/pitaya/v2/cluster/mocks"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/protos"
	sessionmocks "github.com/topfreegames/pitaya/v2/session/mocks"
)

func TestSendKickToUsersLocalSession(t *testing.T) {
	table := struct {
		uid1         string
		uid2         string
		frontendType string
		err          error
	}{
		uuid.New().String(), uuid.New().String(), "connector", nil,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s1 := sessionmocks.NewMockSession(ctrl)
	s2 := sessionmocks.NewMockSession(ctrl)
	s1.EXPECT().Kick(context.Background()).Times(1).Return(table.err)
	s2.EXPECT().Kick(context.Background()).Times(1).Return(table.err)

	mockSessionPool := sessionmocks.NewMockSessionPool(ctrl)
	mockSessionPool.EXPECT().GetSessionByUID(table.uid1).Return(s1).Times(1)
	mockSessionPool.EXPECT().GetSessionByUID(table.uid2).Return(s2).Times(1)

	config := config.NewDefaultBuilderConfig()
	builder := NewDefaultBuilder(true, "testtype", Cluster, map[string]string{}, *config)
	builder.SessionPool = mockSessionPool
	app := builder.Build()

	failedUids, err := app.SendKickToUsers([]string{table.uid1, table.uid2}, table.frontendType)
	assert.Nil(t, failedUids)
	assert.NoError(t, err)
}

func TestSendKickToUsersFail(t *testing.T) {
	table := struct {
		uid1         string
		uid2         string
		frontendType string
		err          error
	}{
		uuid.New().String(), uuid.New().String(), "connector", constants.ErrKickingUsers,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s1 := sessionmocks.NewMockSession(ctrl)
	s1.EXPECT().Kick(context.Background()).Times(1).Return(nil)

	mockSessionPool := sessionmocks.NewMockSessionPool(ctrl)
	mockSessionPool.EXPECT().GetSessionByUID(table.uid1).Return(s1).Times(1)
	mockSessionPool.EXPECT().GetSessionByUID(table.uid2).Return(nil).Times(1)

	mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
	mockRPCClient.EXPECT().SendKick(table.uid2, table.frontendType, &protos.KickMsg{UserId: table.uid2}).Return(table.err).Times(1)

	config := config.NewDefaultBuilderConfig()
	builder := NewDefaultBuilder(true, "testtype", Cluster, map[string]string{}, *config)
	builder.SessionPool = mockSessionPool
	builder.RPCClient = mockRPCClient
	app := builder.Build()

	failedUids, err := app.SendKickToUsers([]string{table.uid1, table.uid2}, table.frontendType)
	assert.Len(t, failedUids, 1)
	assert.Equal(t, failedUids[0], table.uid2)
	assert.Equal(t, err, table.err)
}

func TestSendKickToUsersRemoteSession(t *testing.T) {
	tables := []struct {
		name         string
		uids         []string
		frontendType string
		err          error
	}{
		{"success", []string{uuid.New().String(), uuid.New().String()}, "connector", nil},
		{"fail", []string{uuid.New().String(), uuid.New().String()}, "connector", constants.ErrKickingUsers},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)

			config := config.NewDefaultBuilderConfig()
			app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, *config).(*App)
			app.rpcClient = mockRPCClient

			for _, uid := range table.uids {
				expectedKick := &protos.KickMsg{UserId: uid}
				mockRPCClient.EXPECT().SendKick(uid, gomock.Any(), expectedKick).Return(table.err)
			}

			failedUids, err := app.SendKickToUsers(table.uids, table.frontendType)
			assert.Equal(t, err, table.err)
			if table.err != nil {
				assert.NotNil(t, failedUids)
				assert.Equal(t, failedUids, table.uids)
			} else {
				assert.Nil(t, failedUids)
			}
		})
	}
}
