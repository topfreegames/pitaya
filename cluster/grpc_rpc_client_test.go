package cluster

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/conn/message"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/helpers"
	"github.com/topfreegames/pitaya/v2/interfaces"
	"github.com/topfreegames/pitaya/v2/interfaces/mocks"
	"github.com/topfreegames/pitaya/v2/metrics"
	"github.com/topfreegames/pitaya/v2/protos"
	protosmocks "github.com/topfreegames/pitaya/v2/protos/mocks"
	"github.com/topfreegames/pitaya/v2/route"
	sessionmocks "github.com/topfreegames/pitaya/v2/session/mocks"
	"google.golang.org/grpc"
)

func getRPCClient(c config.GRPCClientConfig) (*GRPCClient, error) {
	sv := getServer()
	return NewGRPCClient(c, sv, []metrics.Reporter{}, nil, nil)
}

func TestNewGRPCClient(t *testing.T) {
	c := config.NewDefaultGRPCClientConfig()
	g, err := getRPCClient(c)
	assert.NoError(t, err)
	assert.NotNil(t, g)
}

func TestCall(t *testing.T) {
	c := config.NewDefaultGRPCClientConfig()
	g, err := getRPCClient(c)
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPitayaClient := protosmocks.NewMockPitayaClient(ctrl)
	g.clientMap.Store(g.server.ID, &grpcClient{
		cli:       mockPitayaClient,
		connected: true,
	})

	uid := "someuid"
	msg := &message.Message{
		Type:  0,
		ID:    0,
		Route: "sv.svc.meth",
		Data:  []byte{0x01},
		Err:   false,
	}

	ctx := context.Background()
	rpcType := protos.RPCType_Sys
	r := route.NewRoute("sv", "svc", "meth")

	sess := sessionmocks.NewMockSession(ctrl)
	sess.EXPECT().ID().Return(int64(1)).Times(2)
	sess.EXPECT().UID().Return(uid).Times(2)
	sess.EXPECT().GetDataEncoded().Return(nil).Times(2)

	expected, err := buildRequest(ctx, rpcType, r, sess, msg, g.server)
	assert.NoError(t, err)

	mockPitayaClient.EXPECT().Call(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, in *protos.Request, opts ...grpc.CallOption) (*protos.Response, error) {
		assert.Equal(t, expected.FrontendID, in.FrontendID)
		assert.Equal(t, expected.Type, in.Type)
		assert.Equal(t, expected.Msg, in.Msg)
		return &protos.Response{
			Data:  []byte{0x01},
			Error: nil,
		}, nil
	})

	res, err := g.Call(ctx, rpcType, r, sess, msg, g.server)

	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestBroadcastSessionBind(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBindingStorage := mocks.NewMockBindingStorage(ctrl)
	mockPitayaClient := protosmocks.NewMockPitayaClient(ctrl)
	tables := []struct {
		name           string
		bindingStorage interfaces.BindingStorage
		err            error
	}{{"shouldrun", mockBindingStorage, nil}, {"shoulderror", nil, constants.ErrNoBindingStorageModule}}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			c := config.NewDefaultGRPCClientConfig()
			g, err := getRPCClient(c)
			assert.NoError(t, err)
			uid := "someuid"
			//mockPitayaClient := protosmocks.NewMockPitayaClient(ctrl)

			if table.bindingStorage != nil {
				g.clientMap.Store(g.server.ID, &grpcClient{connected: true, cli: mockPitayaClient})

				g.bindingStorage = mockBindingStorage
				mockBindingStorage.EXPECT().GetUserFrontendID(uid, gomock.Any()).DoAndReturn(func(u, svType string) (string, error) {
					assert.Equal(t, uid, u)
					assert.Equal(t, g.server.Type, svType)

					return g.server.ID, nil
				})

				mockPitayaClient.EXPECT().SessionBindRemote(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, msg *protos.BindMsg) {
					assert.Equal(t, uid, msg.Uid, g.server.ID, msg.Fid)
				})
			}

			err = g.BroadcastSessionBind(uid)
			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendKick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBindingStorage := mocks.NewMockBindingStorage(ctrl)
	mockPitayaClient := protosmocks.NewMockPitayaClient(ctrl)
	tables := []struct {
		name           string
		userID         string
		bindingStorage interfaces.BindingStorage
		sv             *Server
		err            error
	}{
		{"bindingstorage", "uid", mockBindingStorage, &Server{
			Type:     "tp",
			Frontend: true}, nil,
		},
		{"nobindingstorage", "uid", nil, &Server{
			Type:     "tp",
			Frontend: true,
		}, constants.ErrNoBindingStorageModule},
		{"nobindingstorage", "", mockBindingStorage, &Server{
			Type:     "tp",
			Frontend: true,
		}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			c := config.NewDefaultGRPCClientConfig()
			g, err := getRPCClient(c)
			assert.NoError(t, err)

			if table.bindingStorage != nil {
				g.clientMap.Store(table.sv.ID, &grpcClient{connected: true, cli: mockPitayaClient})
				g.bindingStorage = table.bindingStorage
				mockBindingStorage.EXPECT().GetUserFrontendID(table.userID, gomock.Any()).DoAndReturn(func(u, svType string) (string, error) {
					assert.Equal(t, table.userID, u)
					assert.Equal(t, table.sv.Type, svType)
					return table.sv.ID, nil
				})

				mockPitayaClient.EXPECT().KickUser(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, msg *protos.KickMsg) {
					assert.Equal(t, table.userID, msg.UserId)
				})
			}

			kick := &protos.KickMsg{
				UserId: table.userID,
			}

			err = g.SendKick(table.userID, table.sv.Type, kick)
			if table.err != nil {
				assert.Equal(t, err, table.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendPush(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBindingStorage := mocks.NewMockBindingStorage(ctrl)
	mockPitayaClient := protosmocks.NewMockPitayaClient(ctrl)
	tables := []struct {
		name           string
		bindingStorage interfaces.BindingStorage
		sv             *Server
		err            error
	}{
		{"bindingstorage-no-fid", mockBindingStorage, &Server{
			Type:     "tp",
			Frontend: true,
		}, nil},
		{"nobindingstorage-no-fid", nil, &Server{
			Type:     "tp",
			Frontend: true,
		}, constants.ErrNoBindingStorageModule},
		{"nobindingstorage-fid", nil, &Server{
			ID:       "someID",
			Type:     "tp",
			Frontend: true,
		}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g, err := getRPCClient(config.NewDefaultGRPCClientConfig())
			assert.NoError(t, err)
			uid := "someuid"

			if table.bindingStorage != nil && table.sv.ID == "" {
				g.clientMap.Store(table.sv.ID, &grpcClient{connected: true, cli: mockPitayaClient})
				g.bindingStorage = table.bindingStorage
				mockBindingStorage.EXPECT().GetUserFrontendID(uid, gomock.Any()).DoAndReturn(func(u, svType string) (string, error) {
					assert.Equal(t, uid, u)
					assert.Equal(t, table.sv.Type, svType)
					return table.sv.ID, nil
				})

				mockPitayaClient.EXPECT().PushToUser(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, msg *protos.Push) {
					assert.Equal(t, uid, msg.Uid)
					assert.Equal(t, msg.Route, "sv.svc.mth")
					assert.Equal(t, msg.Data, []byte{0x01})
				})
			} else if table.bindingStorage == nil && table.sv.ID != "" {
				g.clientMap.Store(table.sv.ID, &grpcClient{connected: true, cli: mockPitayaClient})
				mockPitayaClient.EXPECT().PushToUser(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, msg *protos.Push) {
					assert.Equal(t, uid, msg.Uid)
					assert.Equal(t, msg.Route, "sv.svc.mth")
					assert.Equal(t, msg.Data, []byte{0x01})
				})
			}

			push := &protos.Push{
				Route: "sv.svc.mth",
				Uid:   uid,
				Data:  []byte{0x01},
			}

			err = g.SendPush(uid, table.sv, push)

			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddServer(t *testing.T) {
	t.Run("try-connect", func(t *testing.T) {
		// listen
		clientConfig := config.NewDefaultGRPCClientConfig()

		serverConfig := config.NewDefaultGRPCServerConfig()
		serverConfig.Port = helpers.GetFreePort(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		server := &Server{
			ID:   "someid",
			Type: "sometype",
			Metadata: map[string]string{
				constants.GRPCHostKey: "localhost",
				constants.GRPCPortKey: fmt.Sprintf("%d", serverConfig.Port),
			},
			Frontend: false,
		}
		gs, err := NewGRPCServer(serverConfig, server, []metrics.Reporter{})
		assert.NoError(t, err)

		mockPitayaServer := protosmocks.NewMockPitayaServer(ctrl)
		gs.SetPitayaServer(mockPitayaServer)

		err = gs.Init()
		assert.NoError(t, err)
		// --- should connect to the server and add it to the client map
		g, err := getRPCClient(clientConfig)
		assert.NoError(t, err)
		g.AddServer(server)

		sv, ok := g.clientMap.Load(server.ID)
		assert.NotNil(t, sv)
		assert.True(t, ok)
		cli := sv.(*grpcClient)
		assert.True(t, cli.connected)
		assert.NotNil(t, cli.cli)
	})

	t.Run("lazy", func(t *testing.T) {
		// listen
		clientConfig := config.NewDefaultGRPCClientConfig()
		clientConfig.LazyConnection = true

		serverConfig := config.NewDefaultGRPCServerConfig()
		serverConfig.Port = helpers.GetFreePort(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		server := &Server{
			ID:   "someid",
			Type: "sometype",
			Metadata: map[string]string{
				constants.GRPCHostKey: "localhost",
				constants.GRPCPortKey: fmt.Sprintf("%d", serverConfig.Port),
			},
			Frontend: false,
		}
		gs, err := NewGRPCServer(serverConfig, server, []metrics.Reporter{})
		assert.NoError(t, err)

		mockPitayaServer := protosmocks.NewMockPitayaServer(ctrl)
		gs.SetPitayaServer(mockPitayaServer)

		err = gs.Init()
		assert.NoError(t, err)
		g, err := getRPCClient(clientConfig)
		assert.NoError(t, err)
		// --- should not connect to the server and add it to the client map
		g.AddServer(server)

		sv, ok := g.clientMap.Load(server.ID)
		assert.NotNil(t, sv)
		assert.True(t, ok)
		cli := sv.(*grpcClient)
		assert.False(t, cli.connected)
		assert.Nil(t, cli.cli)
	})
}

func TestGetServerHost(t *testing.T) {
	t.Parallel()

	var (
		host         = "host"
		externalHost = "externalHost"
		region       = "region"
	)

	tables := map[string]struct {
		metadata        map[string]string
		clientRegion    string
		expectedHost    string
		expectedPortKey string
	}{
		"test_has_no_region_and_no_external_host": {
			metadata: map[string]string{
				constants.GRPCHostKey: host,
			},
			expectedHost:    host,
			expectedPortKey: constants.GRPCPortKey,
		},
		"test_has_no_region_and_external_host": {
			metadata: map[string]string{
				constants.GRPCExternalHostKey: externalHost,
			},
			expectedHost:    externalHost,
			expectedPortKey: constants.GRPCExternalPortKey,
		},
		"test_has_region_and_same_region": {
			metadata: map[string]string{
				constants.GRPCHostKey: host,
				constants.RegionKey:   region,
			},
			clientRegion:    region,
			expectedHost:    host,
			expectedPortKey: constants.GRPCPortKey,
		},
		"test_has_region_and_other_region": {
			metadata: map[string]string{
				constants.GRPCExternalHostKey: externalHost,
				constants.RegionKey:           region,
			},
			clientRegion:    "other-region",
			expectedHost:    externalHost,
			expectedPortKey: constants.GRPCExternalPortKey,
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			config := config.NewDefaultConfigInfoRetrieverConfig()
			config.Region = table.clientRegion
			infoRetriever := NewConfigInfoRetriever(config)
			gs := &GRPCClient{infoRetriever: infoRetriever}

			host, portKey := gs.getServerHost(&Server{
				Metadata: table.metadata,
			})

			assert.Equal(t, table.expectedHost, host)
			assert.Equal(t, table.expectedPortKey, portKey)
		})
	}
}

func TestRemoveServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clientConfig := config.NewDefaultGRPCClientConfig()

	serverConfig := config.NewDefaultGRPCServerConfig()
	serverConfig.Port = helpers.GetFreePort(t)

	server := &Server{
		ID:   "someid",
		Type: "sometype",
		Metadata: map[string]string{
			constants.GRPCHostKey: "localhost",
			constants.GRPCPortKey: fmt.Sprintf("%d", serverConfig.Port),
		},
		Frontend: false,
	}
	gs, err := NewGRPCServer(serverConfig, server, []metrics.Reporter{})
	assert.NoError(t, err)
	mockPitayaServer := protosmocks.NewMockPitayaServer(ctrl)
	gs.SetPitayaServer(mockPitayaServer)
	err = gs.Init()
	assert.NoError(t, err)

	gc, err := NewGRPCClient(clientConfig, server, []metrics.Reporter{}, nil, nil)
	assert.NoError(t, err)
	gc.AddServer(server)

	sv, ok := gc.clientMap.Load(server.ID)
	assert.NotNil(t, sv)
	assert.True(t, ok)

	gc.RemoveServer(server)

	sv, ok = gc.clientMap.Load(server.ID)
	assert.Nil(t, sv)
	assert.False(t, ok)
}
