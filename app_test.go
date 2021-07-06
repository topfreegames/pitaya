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

package pitaya

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"go.etcd.io/etcd/integration"
	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/conn/codec"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
)

var (
	tables = []struct {
		isFrontend     bool
		serverType     string
		serverMode     ServerMode
		serverMetadata map[string]string
		cfg            *viper.Viper
	}{
		{true, "sv1", Cluster, map[string]string{"name": "bla"}, viper.New()},
		{false, "sv2", Standalone, map[string]string{}, viper.New()},
	}
	typeOfetcdSD        reflect.Type
	typeOfNatsRPCServer reflect.Type
	typeOfNatsRPCClient reflect.Type
)

func TestMain(m *testing.M) {
	setup()
	exit := m.Run()
	os.Exit(exit)
}

func setup() {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())

	etcdSD, err := cluster.NewEtcdServiceDiscovery(app.config, app.server, app.dieChan)
	if err != nil {
		panic(err)
	}
	typeOfetcdSD = reflect.TypeOf(etcdSD)

	natsRPCServer, err := cluster.NewNatsRPCServer(app.config, app.server, nil, app.dieChan)
	if err != nil {
		panic(err)
	}
	typeOfNatsRPCServer = reflect.TypeOf(natsRPCServer)

	natsRPCClient, err := cluster.NewNatsRPCClient(app.config, app.server, nil, app.dieChan)
	if err != nil {
		panic(err)
	}
	typeOfNatsRPCClient = reflect.TypeOf(natsRPCClient)

	c := integration.NewClusterV3(nil, &integration.ClusterConfig{Size: 1})
	cli := c.RandClient()
	gsi, err := groups.NewEtcdGroupService(app.config, cli)
	if err != nil {
		panic(err)
	}
	InitGroups(gsi)
}

func initApp() {
	app = &App{
		server: &cluster.Server{
			ID:       uuid.New().String(),
			Type:     "game",
			Metadata: map[string]string{},
			Frontend: true,
		},
		debug:         false,
		startAt:       time.Now(),
		dieChan:       make(chan bool),
		acceptors:     []acceptor.Acceptor{},
		packetDecoder: codec.NewPomeloPacketDecoder(),
		packetEncoder: codec.NewPomeloPacketEncoder(),
		serverMode:    Standalone,
		serializer:    json.NewSerializer(),
		configured:    false,
		running:       false,
		router:        router.New(),
	}
}

func TestConfigure(t *testing.T) {
	for _, table := range tables {
		t.Run(table.serverType, func(t *testing.T) {
			initApp()
			Configure(table.isFrontend, table.serverType, table.serverMode, table.serverMetadata, table.cfg)
			assert.Equal(t, table.isFrontend, app.server.Frontend)
			assert.Equal(t, table.serverType, app.server.Type)
			assert.Equal(t, table.serverMode, app.serverMode)
			assert.Equal(t, table.serverMetadata, app.server.Metadata)
			assert.Equal(t, true, app.configured)
		})
	}
}

func TestAddAcceptor(t *testing.T) {
	acc := acceptor.NewTCPAcceptor("0.0.0.0:0")
	for _, table := range tables {
		t.Run(table.serverType, func(t *testing.T) {
			initApp()
			Configure(table.isFrontend, table.serverType, table.serverMode, table.serverMetadata, table.cfg)
			AddAcceptor(acc)
			if table.isFrontend {
				assert.Equal(t, acc, app.acceptors[0])
			} else {
				assert.Equal(t, 0, len(app.acceptors))
			}
		})
	}
}

func TestSetDebug(t *testing.T) {
	SetDebug(true)
	assert.Equal(t, true, app.debug)
	SetDebug(false)
	assert.Equal(t, false, app.debug)
}

func TestSetLogger(t *testing.T) {
	l := logrus.New()
	SetLogger(l)
	assert.Equal(t, l, logger.Log)
}

func TestSetPacketDecoder(t *testing.T) {
	d := codec.NewPomeloPacketDecoder()
	SetPacketDecoder(d)
	assert.Equal(t, d, app.packetDecoder)
}

func TestSetPacketEncoder(t *testing.T) {
	e := codec.NewPomeloPacketEncoder()
	SetPacketEncoder(e)
	assert.Equal(t, e, app.packetEncoder)
}

func TestGetDieChan(t *testing.T) {
	assert.Equal(t, app.dieChan, GetDieChan())
}
func TestGetConfig(t *testing.T) {
	assert.Equal(t, app.config, GetConfig())
}

func TestGetSerializer(t *testing.T) {
	assert.Equal(t, app.serializer, GetSerializer())
}

func TestGetSever(t *testing.T) {
	assert.Equal(t, app.server, GetServer())
}

func TestGetMetricsReporters(t *testing.T) {
	assert.Equal(t, app.metricsReporters, GetMetricsReporters())
}
func TestGetServerByID(t *testing.T) {
	r, err := cluster.NewEtcdServiceDiscovery(app.config, app.server, app.dieChan)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetServiceDiscoveryClient(r)
	s, err := GetServerByID("id")
	assert.Nil(t, s)
	assert.EqualError(t, constants.ErrNoServerWithID, err.Error())
}

func TestGetServersByType(t *testing.T) {
	r, err := cluster.NewEtcdServiceDiscovery(app.config, app.server, app.dieChan)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetServiceDiscoveryClient(r)
	s, err := GetServersByType("id")
	assert.Nil(t, s)
	assert.EqualError(t, constants.ErrNoServersAvailableOfType, err.Error())
}

func TestSetHeartbeatInterval(t *testing.T) {
	inter := 35 * time.Millisecond
	SetHeartbeatTime(inter)
	assert.Equal(t, inter, app.heartbeat)
}

func TestSetRPCServer(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewNatsRPCServer(app.config, app.server, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	SetRPCServer(r)
	assert.Equal(t, r, app.rpcServer)
}

func TestSetRPCClient(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewNatsRPCClient(app.config, app.server, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetRPCClient(r)
	assert.Equal(t, r, app.rpcClient)
}

func TestSetServiceDiscovery(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewEtcdServiceDiscovery(app.config, app.server, app.dieChan)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetServiceDiscoveryClient(r)
	assert.Equal(t, r, app.serviceDiscovery)
}

func TestAddMetricsReporter(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := metrics.NewStatsdReporter(app.config, app.server.Type, map[string]string{
		"tag1": "value1",
	})
	assert.NoError(t, err)
	assert.NotNil(t, r)
	AddMetricsReporter(r)
	assert.Contains(t, app.metricsReporters, r)
}

func TestSetSerializer(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r := json.NewSerializer()
	assert.NotNil(t, r)
	SetSerializer(r)
	assert.Equal(t, r, app.serializer)
}

func TestInitSysRemotes(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	initSysRemotes()
	assert.NotNil(t, remoteComp[0])
}

func TestSetDictionary(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())

	dict := map[string]uint16{"someroute": 12}
	err := SetDictionary(dict)
	assert.NoError(t, err)
	assert.Equal(t, dict, message.GetDictionary())

	app.running = true
	err = SetDictionary(dict)
	assert.EqualError(t, constants.ErrChangeDictionaryWhileRunning, err.Error())
}

func TestAddRoute(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	app.router = nil
	err := AddRoute("somesv", func(ctx context.Context, route *route.Route, payload []byte, servers map[string]*cluster.Server) (*cluster.Server, error) {
		return nil, nil
	})
	assert.EqualError(t, constants.ErrRouterNotInitialized, err.Error())

	app.router = router.New()
	err = AddRoute("somesv", func(ctx context.Context, route *route.Route, payload []byte, servers map[string]*cluster.Server) (*cluster.Server, error) {
		return nil, nil
	})
	assert.NoError(t, err)

	app.running = true
	err = AddRoute("somesv", func(ctx context.Context, route *route.Route, payload []byte, servers map[string]*cluster.Server) (*cluster.Server, error) {
		return nil, nil
	})
	assert.EqualError(t, constants.ErrChangeRouteWhileRunning, err.Error())
}

func TestShutdown(t *testing.T) {
	initApp()
	go func() {
		Shutdown()
	}()
	<-app.dieChan
}

func TestConfigureDefaultMetricsReporter(t *testing.T) {
	tables := []struct {
		enabled bool
	}{
		{true},
		{false},
	}

	for _, table := range tables {
		t.Run(fmt.Sprintf("%t", table.enabled), func(t *testing.T) {
			initApp()
			cfg := viper.New()
			cfg.Set("pitaya.metrics.prometheus.enabled", table.enabled)
			cfg.Set("pitaya.metrics.statsd.enabled", table.enabled)
			Configure(true, "testtype", Cluster, map[string]string{}, cfg)
			// if statsd is enabled there are 2 metricsReporters, prometheus and statsd
			assert.Equal(t, table.enabled, len(app.metricsReporters) == 2)
		})
	}
}

func TestStartDefaultSD(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	startDefaultSD()
	assert.NotNil(t, app.serviceDiscovery)
	assert.Equal(t, typeOfetcdSD, reflect.TypeOf(app.serviceDiscovery))
}

func TestStartDefaultRPCServer(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	startDefaultRPCServer()
	assert.NotNil(t, app.rpcServer)
	assert.Equal(t, typeOfNatsRPCServer, reflect.TypeOf(app.rpcServer))
}

func TestStartDefaultRPCClient(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	startDefaultRPCClient()
	assert.NotNil(t, app.rpcClient)
	assert.Equal(t, typeOfNatsRPCClient, reflect.TypeOf(app.rpcClient))
}

func TestStartAndListenStandalone(t *testing.T) {
	initApp()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	acc := acceptor.NewTCPAcceptor("0.0.0.0:0")
	AddAcceptor(acc)

	go func() {
		Start()
	}()
	helpers.ShouldEventuallyReturn(t, func() bool {
		return app.running
	}, true)

	assert.NotNil(t, handlerService)
	assert.NotNil(t, timer.GlobalTicker)
	// should be listening
	assert.NotEmpty(t, acc.GetAddr())
	helpers.ShouldEventuallyReturn(t, func() error {
		n, err := net.Dial("tcp", acc.GetAddr())
		defer n.Close()
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)
}

func ConfigureClusterApp() {

}

func TestStartAndListenCluster(t *testing.T) {
	es, cli := helpers.GetTestEtcd(t)
	defer es.Terminate(t)

	ns := helpers.GetTestNatsServer(t)
	nsAddr := ns.Addr().String()

	cfg := viper.New()
	cfg.Set("pitaya.cluster.rpc.client.nats.connect", fmt.Sprintf("nats://%s", nsAddr))
	cfg.Set("pitaya.cluster.rpc.server.nats.connect", fmt.Sprintf("nats://%s", nsAddr))

	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, cfg)

	etcdSD, err := cluster.NewEtcdServiceDiscovery(app.config, app.server, app.dieChan, cli)
	assert.NoError(t, err)
	SetServiceDiscoveryClient(etcdSD)

	acc := acceptor.NewTCPAcceptor("0.0.0.0:0")
	assert.Nil(t, err)
	AddAcceptor(acc)

	go func() {
		Start()
	}()
	helpers.ShouldEventuallyReturn(t, func() bool {
		return app.running
	}, true)

	assert.NotNil(t, handlerService)
	assert.NotNil(t, timer.GlobalTicker)
	// should be listening
	assert.NotEmpty(t, acc.GetAddr())
	helpers.ShouldEventuallyReturn(t, func() error {
		n, err := net.Dial("tcp", acc.GetAddr())
		defer n.Close()
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)
}

func TestError(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name     string
		err      error
		code     string
		metadata map[string]string
	}{
		{"nil_metadata", errors.New(uuid.New().String()), uuid.New().String(), nil},
		{"empty_metadata", errors.New(uuid.New().String()), uuid.New().String(), map[string]string{}},
		{"non_empty_metadata", errors.New(uuid.New().String()), uuid.New().String(), map[string]string{"key": uuid.New().String()}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			var err *e.Error
			if table.metadata != nil {
				err = Error(table.err, table.code, table.metadata)
			} else {
				err = Error(table.err, table.code)
			}
			assert.NotNil(t, err)
			assert.Equal(t, table.code, err.Code)
			assert.Equal(t, table.err.Error(), err.Message)
			assert.Equal(t, table.metadata, err.Metadata)
		})
	}
}

func TestGetSessionFromCtx(t *testing.T) {
	ss := &session.Session{}
	ctx := context.WithValue(context.Background(), constants.SessionCtxKey, ss)
	s := GetSessionFromCtx(ctx)
	assert.Equal(t, ss, s)
}

func TestAddMetricTagsToPropagateCtx(t *testing.T) {
	ctx := AddMetricTagsToPropagateCtx(context.Background(), map[string]string{
		"key": "value",
	})
	val := ctx.Value(constants.PropagateCtxKey)
	assert.Equal(t, map[string]interface{}{
		constants.MetricTagsKey: map[string]string{
			"key": "value",
		},
	}, val)
}

func TestAddToPropagateCtx(t *testing.T) {
	ctx := AddToPropagateCtx(context.Background(), "key", "val")
	val := ctx.Value(constants.PropagateCtxKey)
	assert.Equal(t, map[string]interface{}{"key": "val"}, val)
}

func TestGetFromPropagateCtx(t *testing.T) {
	ctx := AddToPropagateCtx(context.Background(), "key", "val")
	val := GetFromPropagateCtx(ctx, "key")
	assert.Equal(t, "val", val)
}

func TestExtractSpan(t *testing.T) {
	span := opentracing.StartSpan("op", opentracing.ChildOf(nil))
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	spanCtx, err := ExtractSpan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, span.Context(), spanCtx)
}

func TestDescriptor(t *testing.T) {
	bts, err := Descriptor("kick.proto")
	assert.NoError(t, err)
	assert.NotNil(t, bts)

	bts, err = Descriptor("not_exists.proto")
	assert.Nil(t, bts)
	assert.EqualError(t, constants.ErrProtodescriptor, err.Error())
}

func TestDocumentation(t *testing.T) {
	doc, err := Documentation(false)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"handlers": map[string]interface{}{},
		"remotes": map[string]interface{}{
			"testtype.sys.bindsession": map[string]interface{}{
				"input": map[string]interface{}{
					"uid":  "string",
					"data": "[]byte",
					"id":   "int64",
				},
				"output": []interface{}{
					map[string]interface{}{
						"error": map[string]interface{}{
							"msg":      "string",
							"code":     "string",
							"metadata": "map[string]string",
						},
						"data": "[]byte",
					},
					"error",
				},
			},
			"testtype.sys.kick": map[string]interface{}{
				"input": map[string]interface{}{
					"userId": "string",
				},
				"output": []interface{}{
					map[string]interface{}{
						"kicked": "bool",
					},
					"error",
				},
			},
			"testtype.sys.pushsession": map[string]interface{}{
				"input": map[string]interface{}{
					"data": "[]byte",
					"id":   "int64",
					"uid":  "string",
				},
				"output": []interface{}{
					map[string]interface{}{
						"error": map[string]interface{}{
							"code":     "string",
							"metadata": "map[string]string",
							"msg":      "string",
						},
						"data": "[]byte",
					},
					"error",
				},
			},
		},
	}, doc)
}

func TestDocumentationTrue(t *testing.T) {
	doc, err := Documentation(true)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"remotes": map[string]interface{}{
			"testtype.sys.bindsession": map[string]interface{}{
				"input": map[string]interface{}{
					"*protos.Session": map[string]interface{}{
						"data": "[]byte",
						"id":   "int64",
						"uid":  "string",
					},
				},
				"output": []interface{}{map[string]interface{}{
					"*protos.Response": map[string]interface{}{
						"data": "[]byte",
						"error": map[string]interface{}{
							"*protos.Error": map[string]interface{}{
								"code":     "string",
								"metadata": "map[string]string",
								"msg":      "string",
							},
						},
					},
				},
					"error",
				},
			},
			"testtype.sys.kick": map[string]interface{}{
				"input": map[string]interface{}{
					"*protos.KickMsg": map[string]interface{}{
						"userId": "string",
					},
				},
				"output": []interface{}{map[string]interface{}{
					"*protos.KickAnswer": map[string]interface{}{
						"kicked": "bool",
					},
				},
					"error",
				},
			},
			"testtype.sys.pushsession": map[string]interface{}{
				"input": map[string]interface{}{
					"*protos.Session": map[string]interface{}{
						"data": "[]byte",
						"id":   "int64",
						"uid":  "string",
					},
				},
				"output": []interface{}{map[string]interface{}{
					"*protos.Response": map[string]interface{}{
						"data": "[]byte",
						"error": map[string]interface{}{
							"*protos.Error": map[string]interface{}{
								"code":     "string",
								"metadata": "map[string]string",
								"msg":      "string",
							},
						},
					},
				},
					"error",
				},
			},
		},
		"handlers": map[string]interface{}{},
	}, doc)
}

func TestAddGRPCInfoToMetadata(t *testing.T) {
	t.Parallel()

	metadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	metadata = AddGRPCInfoToMetadata(metadata, "region", "host", "port", "external-host", "external-port")

	assert.Equal(t, map[string]string{
		"key1":                        "value1",
		"key2":                        "value2",
		"key3":                        "value3",
		constants.GRPCHostKey:         "host",
		constants.GRPCPortKey:         "port",
		constants.GRPCExternalHostKey: "external-host",
		constants.GRPCExternalPortKey: "external-port",
		constants.RegionKey:           "region",
	}, metadata)
}

func TestStartWorker(t *testing.T) {
	cfg := viper.New()
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, cfg)

	err := StartWorker(GetConfig())
	assert.NoError(t, err)
	assert.True(t, app.worker.Started())
}

func TestRegisterRPCJob(t *testing.T) {
	t.Run("register_once", func(t *testing.T) {
		cfg := viper.New()
		initApp()
		Configure(true, "testtype", Cluster, map[string]string{}, cfg)
		err := StartWorker(GetConfig())
		assert.NoError(t, err)

		err = RegisterRPCJob(nil)
		assert.NoError(t, err)
	})

	t.Run("register_twice", func(t *testing.T) {
		cfg := viper.New()
		initApp()
		Configure(true, "testtype", Cluster, map[string]string{}, cfg)
		err := StartWorker(GetConfig())
		assert.NoError(t, err)

		err = RegisterRPCJob(nil)
		assert.NoError(t, err)

		err = RegisterRPCJob(nil)
		assert.Equal(t, constants.ErrRPCJobAlreadyRegistered, err)
	})
}
