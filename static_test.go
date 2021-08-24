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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v2/cluster"
	"github.com/topfreegames/pitaya/v2/metrics"
	"github.com/topfreegames/pitaya/v2/mocks"
	sessionmocks "github.com/topfreegames/pitaya/v2/session/mocks"
	"github.com/topfreegames/pitaya/v2/worker"
	workermocks "github.com/topfreegames/pitaya/v2/worker/mocks"
)

func TestStaticConfigure(t *testing.T) {
	// Configure(isFrontend bool, serverType string, serverMode ServerMode, serverMetadata map[string]string, cfgs ...*viper.Viper)
	// TODO(static): implement test
}

func TestStaticGetDieChan(t *testing.T) {
	// GetDieChan() chan bool
	// TODO(static): implement test
}

func TestStaticSetDebug(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := true

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().SetDebug(expected)

	DefaultApp = app
	SetDebug(expected)
}

func TestStaticSetHeartbeatTime(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := 2 * time.Second

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().SetHeartbeatTime(expected)

	DefaultApp = app
	SetHeartbeatTime(expected)
}

func TestStaticGetServerID(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := uuid.New().String()

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetServerID().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetServerID())
}

func TestStaticGetMetricsReporters(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := []metrics.Reporter{}

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetMetricsReporters().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetMetricsReporters())
}

func TestStaticGetServer(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := &cluster.Server{}

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetServer().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetServer())
}

func TestStaticGetServerByID(t *testing.T) {
	tables := []struct {
		name   string
		id     string
		err    error
		server *cluster.Server
	}{
		{"Success", uuid.New().String(), nil, &cluster.Server{}},
		{"Error", uuid.New().String(), errors.New("error"), nil},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GetServerByID(row.id).Return(row.server, row.err)

			DefaultApp = app
			server, err := GetServerByID(row.id)
			require.Equal(t, row.err, err)
			require.Equal(t, row.server, server)
		})
	}
}

func TestStaticGetServersByType(t *testing.T) {
	tables := []struct {
		name   string
		typ    string
		err    error
		server map[string]*cluster.Server
	}{
		{"Success", "type", nil, map[string]*cluster.Server{"type": {}}},
		{"Error", "type", errors.New("error"), nil},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GetServersByType(row.typ).Return(row.server, row.err)

			DefaultApp = app
			server, err := GetServersByType(row.typ)
			require.Equal(t, row.err, err)
			require.Equal(t, row.server, server)
		})
	}
}

func TestStaticGetServers(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := []*cluster.Server{{}}

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetServers().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetServers())
}

func TestStaticGetSessionFromCtx(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := sessionmocks.NewMockSession(ctrl)
	ctx := context.Background()

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetSessionFromCtx(ctx).Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetSessionFromCtx(ctx))
}

func TestStaticStart(t *testing.T) {
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().Start()

	DefaultApp = app
	Start()
}

func TestStaticSetDictionary(t *testing.T) {
	// SetDictionary(dict map[string]uint16) error
	// TODO(static): implement test
}

func TestStaticAddRoute(t *testing.T) {
	tables := []struct {
		name       string
		serverType string
		err        error
	}{
		{"Success", "type", nil},
		{"Error", "type", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().AddRoute(row.serverType, nil).Return(row.err) // Note that functions can't be tested for equality

			DefaultApp = app
			err := AddRoute(row.serverType, nil)
			require.Equal(t, row.err, err)
		})
	}
}

func TestStaticShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().Shutdown()

	DefaultApp = app
	Shutdown()
}

func TestStaticStartWorker(t *testing.T) {
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().StartWorker()

	DefaultApp = app
	StartWorker()
}

func TestStaticRegisterRPCJob(t *testing.T) {
	create := func(ctrl *gomock.Controller) worker.RPCJob {
		return workermocks.NewMockRPCJob(ctrl)
	}

	tables := []struct {
		name   string
		create func(ctrl *gomock.Controller) worker.RPCJob
		err    error
	}{
		{"Success", create, nil},
		{"Error", create, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			job := create(ctrl)
			app.EXPECT().RegisterRPCJob(job).Return(row.err)

			DefaultApp = app
			err := RegisterRPCJob(job)
			require.Equal(t, row.err, err)
		})
	}
}

func TestStaticDocumentation(t *testing.T) {
	// Documentation(getPtrNames bool) (map[string]interface{}, error)
	// TODO(static): implement test
}

func TestStaticIsRunning(t *testing.T) {
	// IsRunning() bool
	// TODO(static): implement test
}

func TestStaticRPC(t *testing.T) {
	// RPC(ctx context.Context, routeStr string, reply proto.Message, arg proto.Message) error
	// TODO(static): implement test
}

func TestStaticRPCTo(t *testing.T) {
	// RPCTo(ctx context.Context, serverID, routeStr string, reply proto.Message, arg proto.Message) error
	// TODO(static): implement test
}

func TestStaticReliableRPC(t *testing.T) {
	// ReliableRPC(routeStr string, metadata map[string]interface{}, reply, arg proto.Message) (jid string, err error)
	// TODO(static): implement test
}

func TestStaticReliableRPCWithOptions(t *testing.T) {
	// ReliableRPCWithOptions(routeStr string, metadata map[string]interface{}, reply, arg proto.Message, opts *config.EnqueueOpts) (jid
	// TODO: (static)stringerr error)
}

func TestStaticSendPushToUsers(t *testing.T) {
	// SendPushToUsers(route string, v interface{}, uids []string, frontendType string) ([]string, error)
	// TODO(static): implement test
}

func TestStaticSendKickToUsers(t *testing.T) {
	// SendKickToUsers(uids []string, frontendType string) ([]string, error)
	// TODO(static): implement test
}

func TestStaticGroupCreate(t *testing.T) {
	// GroupCreate(ctx context.Context, groupName string) error
	// TODO(static): implement test
}

func TestStaticGroupCreateWithTTL(t *testing.T) {
	// GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error
	// TODO(static): implement test
}

func TestStaticGroupMembers(t *testing.T) {
	// GroupMembers(ctx context.Context, groupName string) ([]string, error)
	// TODO(static): implement test
}

func TestStaticGroupBroadcast(t *testing.T) {
	// GroupBroadcast(ctx context.Context, frontendType, groupName, route string, v interface{}) error
	// TODO(static): implement test
}

func TestStaticGroupContainsMember(t *testing.T) {
	// GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error)
	// TODO(static): implement test
}

func TestStaticGroupAddMember(t *testing.T) {
	// GroupAddMember(ctx context.Context, groupName, uid string) error
	// TODO(static): implement test
}

func TestStaticGroupRemoveMember(t *testing.T) {
	// GroupRemoveMember(ctx context.Context, groupName, uid string) error
	// TODO(static): implement test
}

func TestStaticGroupRemoveAll(t *testing.T) {
	// GroupRemoveAll(ctx context.Context, groupName string) error
	// TODO(static): implement test
}

func TestStaticGroupCountMembers(t *testing.T) {
	// GroupCountMembers(ctx context.Context, groupName string) (int, error)
	// TODO(static): implement test
}

func TestStaticGroupRenewTTL(t *testing.T) {
	// GroupRenewTTL(ctx context.Context, groupName string) error
	// TODO(static): implement test
}

func TestStaticGroupDelete(t *testing.T) {
	// GroupDelete(ctx context.Context, groupName string) error
	// TODO(static): implement test
}

func TestStaticRegister(t *testing.T) {
	// Register(c component.Component, options ...component.Option)
	// TODO(static): implement test
}

func TestStaticRegisterRemote(t *testing.T) {
	// RegisterRemote(c component.Component, options ...component.Option)
	// TODO(static): implement test
}

func TestStaticRegisterModule(t *testing.T) {
	// RegisterModule(module interfaces.Module, name string) error
	// TODO(static): implement test
}

func TestStaticRegisterModuleAfter(t *testing.T) {
	// RegisterModuleAfter(module interfaces.Module, name string) error
	// TODO(static): implement test
}

func TestStaticRegisterModuleBefore(t *testing.T) {
	// RegisterModuleBefore(module interfaces.Module, name string) error
	// TODO(static): implement test
}

func TestStaticGetModule(t *testing.T) {
	// GetModule(name string) (interfaces.Module, error)
	// TODO(static): implement test
}
