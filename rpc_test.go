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
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v2/cluster"
	clustermocks "github.com/topfreegames/pitaya/v2/cluster/mocks"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/conn/codec"
	"github.com/topfreegames/pitaya/v2/conn/message"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/pipeline"
	"github.com/topfreegames/pitaya/v2/protos"
	"github.com/topfreegames/pitaya/v2/protos/test"
	"github.com/topfreegames/pitaya/v2/route"
	"github.com/topfreegames/pitaya/v2/router"
	serializemocks "github.com/topfreegames/pitaya/v2/serialize/mocks"
	"github.com/topfreegames/pitaya/v2/service"
	sessionmocks "github.com/topfreegames/pitaya/v2/session/mocks"
)

func TestDoSendRPCNotInitialized(t *testing.T) {
	config := config.NewDefaultPitayaConfig()
	app := NewDefaultApp(true, "testtype", Standalone, map[string]string{}, *config).(*App)
	err := app.doSendRPC(nil, "", "", nil, nil)
	assert.Equal(t, constants.ErrRPCServerNotInitialized, err)
}

type RemoteMock struct {
	component.Base
	called bool
	res    *test.SomeStruct
	err    error
}

func (r *RemoteMock) Bla(ctx context.Context, args *test.SomeStruct) (*test.SomeStruct, error) {
	r.called = true
	return r.res, r.err
}

func TestDoSendRPC(t *testing.T) {
	config := config.NewDefaultPitayaConfig()
	config.Cluster.RPC.Server.LoopbackEnabled = true
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, *config).(*App)
	app.server.ID = "myserver"
	app.rpcServer = &cluster.NatsRPCServer{}
	tables := []struct {
		name       string
		routeStr   string
		reply      proto.Message
		arg        proto.Message
		isLoopback bool
		err        error
	}{
		{"bad_route", "badroute", &test.SomeStruct{}, nil, false, route.ErrInvalidRoute},
		{"no_server_type", "bla.bla", &test.SomeStruct{}, nil, false, constants.ErrNoServerTypeChosenForRPC},
		{"loopback_rpc", "testtype.RemoteMock.Bla", &test.SomeStruct{}, &test.SomeStruct{A: 1}, true, nil},
		{"success", "bla.bla.bla", &test.SomeStruct{}, &test.SomeStruct{A: 1}, false, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := context.Background()

			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
			mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
			mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
			messageEncoder := message.NewMessagesEncoder(false)
			sessionPool := sessionmocks.NewMockSessionPool(ctrl)
			router := router.New()
			handlerPool := service.NewHandlerPool()
			svc := service.NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, app.server, sessionPool, pipeline.NewRemoteHooks(), pipeline.NewHandlerHooks(), handlerPool)
			assert.NotNil(t, svc)
			app.remoteService = svc

			remote := RemoteMock{
				res: table.reply.(*test.SomeStruct),
			}
			require.NoError(t, svc.Register(&remote, []component.Option{}))

			if table.err == nil {
				if !table.isLoopback {
					app.server.ID = "notmyserver"
					b, err := proto.Marshal(&test.SomeStruct{A: 1})
					assert.NoError(t, err)
					mockSD.EXPECT().GetServer("myserver").Return(&cluster.Server{}, nil)
					mockRPCClient.EXPECT().Call(ctx, protos.RPCType_User, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&protos.Response{Data: b}, nil)
				}
			}
			err := app.RPCTo(ctx, "myserver", table.routeStr, table.reply, table.arg)
			assert.Equal(t, table.err, err)

			if table.isLoopback {
				assert.True(t, remote.called)
			}
		})
	}
}
