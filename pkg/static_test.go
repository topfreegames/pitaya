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
	"math"
	"testing"
	"time"

	"github.com/topfreegames/pitaya/v3/pkg/constants"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v3/pkg/cluster"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/interfaces"
	"github.com/topfreegames/pitaya/v3/pkg/metrics"
	"github.com/topfreegames/pitaya/v3/pkg/mocks"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	sessionmocks "github.com/topfreegames/pitaya/v3/pkg/session/mocks"
	"github.com/topfreegames/pitaya/v3/pkg/worker"
	workermocks "github.com/topfreegames/pitaya/v3/pkg/worker/mocks"
	"google.golang.org/protobuf/runtime/protoiface"
)

func TestStaticConfigure(t *testing.T) {
	Configure(true, "frontendType", Cluster, map[string]string{}, []*viper.Viper{}...)

	require.NotNil(t, DefaultApp)
	require.NotNil(t, session.DefaultSessionPool)
}

func TestStaticGetDieChan(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := make(chan bool)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetDieChan().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetDieChan())
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
	ss := sessionmocks.NewMockSession(ctrl)

	ctx := context.WithValue(context.Background(), constants.SessionCtxKey, ss)
	s := GetSessionFromCtx(ctx)
	require.Equal(t, ss, s)
}

func TestStaticStart(t *testing.T) {
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().Start()

	DefaultApp = app
	Start()
}

func TestStaticSetDictionary(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := map[string]uint16{"test": 1}

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().SetDictionary(expected).Return(nil)

	DefaultApp = app
	SetDictionary(expected)
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
	tables := []struct {
		name         string
		expectedBool bool
		returned     map[string]interface{}
		err          error
	}{
		{"Success", true, map[string]interface{}{}, nil},
		{"Error", true, nil, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().Documentation(row.expectedBool).Return(row.returned, row.err)

			DefaultApp = app
			ret, err := Documentation(row.expectedBool)
			require.Equal(t, row.returned, ret)
			require.Equal(t, row.err, err)
		})
	}
}

func TestStaticIsRunning(t *testing.T) {
	tables := []struct {
		name     string
		returned bool
	}{
		{"Success", true},
		{"Error", false},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().IsRunning().Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, IsRunning())
		})
	}
}

func TestStaticRPC(t *testing.T) {
	ctx := context.Background()
	routeStr := "route"
	var reply protoiface.MessageV1
	var arg protoiface.MessageV1

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().RPC(ctx, routeStr, reply, arg).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, RPC(ctx, routeStr, reply, arg))
		})
	}
}

func TestStaticRPCTo(t *testing.T) {
	ctx := context.Background()
	routeStr := "route"
	serverId := uuid.New().String()
	var reply protoiface.MessageV1
	var arg protoiface.MessageV1

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().RPCTo(ctx, serverId, routeStr, reply, arg).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, RPCTo(ctx, serverId, routeStr, reply, arg))
		})
	}
}

func TestStaticReliableRPC(t *testing.T) {
	tables := []struct {
		name     string
		route    string
		metadata map[string]interface{}
		reply    proto.Message
		arg      proto.Message
		jid      string
		err      error
	}{
		{"Success", "route", map[string]interface{}{}, nil, nil, "jid", nil},
		{"Error", "route", map[string]interface{}{}, nil, nil, "", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().ReliableRPC(row.route, row.metadata, row.reply, row.arg).Return(row.jid, row.err)

			DefaultApp = app
			jid, err := ReliableRPC(row.route, row.metadata, row.reply, row.arg)
			require.Equal(t, row.err, err)
			require.Equal(t, row.jid, jid)
		})
	}
}

func TestStaticReliableRPCWithOptions(t *testing.T) {
	tables := []struct {
		name     string
		route    string
		metadata map[string]interface{}
		reply    proto.Message
		arg      proto.Message
		opts     *config.EnqueueOpts
		jid      string
		err      error
	}{
		{"Success", "route", map[string]interface{}{}, nil, nil, nil, "jid", nil},
		{"Error", "route", map[string]interface{}{}, nil, nil, nil, "", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().ReliableRPCWithOptions(row.route, row.metadata, row.reply, row.arg, row.opts).Return(row.jid, row.err)

			DefaultApp = app
			jid, err := ReliableRPCWithOptions(row.route, row.metadata, row.reply, row.arg, row.opts)
			require.Equal(t, row.err, err)
			require.Equal(t, row.jid, jid)
		})
	}
}

func TestStaticSendPushToUsers(t *testing.T) {
	tables := []struct {
		name         string
		route        string
		v            interface{}
		uids         []string
		frontendType string
		returned     []string
		err          error
	}{
		{"Success", "route", nil, []string{"member"}, "frontendType", []string{"returned"}, nil},
		{"Error", "route", nil, nil, "frontendType", nil, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().SendPushToUsers(row.route, row.v, row.uids, row.frontendType).Return(row.returned, row.err)

			DefaultApp = app
			returned, err := SendPushToUsers(row.route, row.v, row.uids, row.frontendType)
			require.Equal(t, row.err, err)
			require.Equal(t, row.returned, returned)
		})
	}
}

func TestStaticSendKickToUsers(t *testing.T) {
	tables := []struct {
		name         string
		uids         []string
		frontendType string
		returned     []string
		err          error
	}{
		{"Success", []string{"member"}, "frontendType", []string{"returned"}, nil},
		{"Error", nil, "frontendType", nil, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().SendKickToUsers(row.uids, row.frontendType).Return(row.returned, row.err)

			DefaultApp = app
			returned, err := SendKickToUsers(row.uids, row.frontendType)
			require.Equal(t, row.err, err)
			require.Equal(t, row.returned, returned)
		})
	}
}

func TestStaticGroupCreate(t *testing.T) {
	ctx := context.Background()
	groupName := "group"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupCreate(ctx, groupName).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupCreate(ctx, groupName))
		})
	}
}

func TestStaticGroupCreateWithTTL(t *testing.T) {
	ctx := context.Background()
	groupName := "group"
	ttl := 2 * time.Second

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupCreateWithTTL(ctx, groupName, ttl).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupCreateWithTTL(ctx, groupName, ttl))
		})
	}
}

func TestStaticGroupMembers(t *testing.T) {
	ctx := context.Background()
	tables := []struct {
		name      string
		groupName string
		members   []string
		err       error
	}{
		{"Success", "name", []string{"member"}, nil},
		{"Error", "name", nil, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupMembers(ctx, row.groupName).Return(row.members, row.err)

			DefaultApp = app
			members, err := GroupMembers(ctx, row.groupName)
			require.Equal(t, row.err, err)
			require.Equal(t, row.members, members)
		})
	}
}

func TestStaticGroupBroadcast(t *testing.T) {
	ctx := context.Background()
	groupName := "group"
	frontendType := "type"
	route := "route"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupBroadcast(ctx, frontendType, groupName, route, nil).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupBroadcast(ctx, frontendType, groupName, route, nil))
		})
	}
}

func TestStaticGroupContainsMember(t *testing.T) {
	ctx := context.Background()
	tables := []struct {
		name      string
		groupName string
		uid       string
		contains  bool
		err       error
	}{
		{"Success", "groupName", uuid.New().String(), true, nil},
		{"Success/False", "groupName", uuid.New().String(), false, nil},
		{"Error", "groupName", uuid.New().String(), true, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupContainsMember(ctx, row.groupName, row.uid).Return(row.contains, row.err)

			DefaultApp = app
			contains, err := GroupContainsMember(ctx, row.groupName, row.uid)
			require.Equal(t, row.err, err)
			require.Equal(t, row.contains, contains)
		})
	}
}

func TestStaticGroupAddMember(t *testing.T) {
	ctx := context.Background()
	groupName := "group"
	uid := uuid.New().String()

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupAddMember(ctx, groupName, uid).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupAddMember(ctx, groupName, uid))
		})
	}
}

func TestStaticGroupRemoveMember(t *testing.T) {
	ctx := context.Background()
	groupName := "group"
	uid := uuid.New().String()

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupRemoveMember(ctx, groupName, uid).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupRemoveMember(ctx, groupName, uid))
		})
	}
}

func TestStaticGroupRemoveAll(t *testing.T) {
	ctx := context.Background()
	groupName := "group"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupRemoveAll(ctx, groupName).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupRemoveAll(ctx, groupName))
		})
	}
}

func TestStaticGroupCountMembers(t *testing.T) {
	ctx := context.Background()
	tables := []struct {
		name      string
		groupName string
		count     int
		err       error
	}{
		{"Success", "name", 3, nil},
		{"Error", "name", 0, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupCountMembers(ctx, row.groupName).Return(row.count, row.err)

			DefaultApp = app
			count, err := GroupCountMembers(ctx, row.groupName)
			require.Equal(t, row.err, err)
			require.Equal(t, row.count, count)
		})
	}
}

func TestStaticGroupRenewTTL(t *testing.T) {
	ctx := context.Background()
	groupName := "group"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupRenewTTL(ctx, groupName).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupRenewTTL(ctx, groupName))
		})
	}
}

func TestStaticGroupDelete(t *testing.T) {
	ctx := context.Background()
	groupName := "group"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GroupDelete(ctx, groupName).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, GroupDelete(ctx, groupName))
		})
	}
}

func TestStaticRegister(t *testing.T) {
	var c component.Component
	options := []component.Option{}
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().Register(c, options)

	DefaultApp = app
	Register(c, options...)
}

func TestStaticRegisterRemote(t *testing.T) {
	var c component.Component
	options := []component.Option{}
	ctrl := gomock.NewController(t)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().RegisterRemote(c, options)

	DefaultApp = app
	RegisterRemote(c, options...)
}

func TestStaticRegisterModule(t *testing.T) {
	var module interfaces.Module
	name := "name"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().RegisterModule(module, name).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, RegisterModule(module, name))
		})
	}
}

func TestStaticRegisterModuleAfter(t *testing.T) {
	var module interfaces.Module
	name := "name"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().RegisterModuleAfter(module, name).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, RegisterModuleAfter(module, name))
		})
	}
}

func TestStaticRegisterModuleBefore(t *testing.T) {
	var module interfaces.Module
	name := "name"

	tables := []struct {
		name     string
		returned error
	}{
		{"Success", nil},
		{"Error", errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().RegisterModuleBefore(module, name).Return(row.returned)

			DefaultApp = app
			require.Equal(t, row.returned, RegisterModuleBefore(module, name))
		})
	}
}

func TestStaticGetModule(t *testing.T) {
	tables := []struct {
		name       string
		moduleName string
		module     interfaces.Module
		err        error
	}{
		{"Success", "name", nil, nil},
		{"Error", "name", nil, errors.New("error")},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			app := mocks.NewMockPitaya(ctrl)
			app.EXPECT().GetModule(row.moduleName).Return(row.module, row.err)

			DefaultApp = app
			module, err := GetModule(row.moduleName)
			require.Equal(t, row.err, err)
			require.Equal(t, row.module, module)
		})
	}
}

func TestGetNumberOfConnectedClients(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := int64(math.MaxInt64)

	app := mocks.NewMockPitaya(ctrl)
	app.EXPECT().GetNumberOfConnectedClients().Return(expected)

	DefaultApp = app
	require.Equal(t, expected, GetNumberOfConnectedClients())
}
