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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/session"
)

var tables = []struct {
	isFrontend     bool
	serverType     string
	serverMode     ServerMode
	serverMetadata map[string]string
	cfg            *viper.Viper
}{
	{true, "sv1", Cluster, map[string]string{"name": "bla"}, viper.New()},
	{false, "sv2", Standalone, map[string]string{}, viper.New()},
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

func TestSetHeartbeatInterval(t *testing.T) {
	inter := 35 * time.Millisecond
	SetHeartbeatTime(inter)
	assert.Equal(t, inter, app.heartbeat)
}

func TestSetRPCServer(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewNatsRPCServer(app.config, app.server)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	SetRPCServer(r)
	assert.Equal(t, r, app.rpcServer)
}

func TestSetRPCClient(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewNatsRPCClient(app.config, app.server)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetRPCClient(r)
	assert.Equal(t, r, app.rpcClient)
}

func TestSetServiceDiscovery(t *testing.T) {
	initApp()
	Configure(true, "testtype", Cluster, map[string]string{}, viper.New())
	r, err := cluster.NewEtcdServiceDiscovery(app.config, app.server)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	SetServiceDiscoveryClient(r)
	assert.Equal(t, r, app.serviceDiscovery)
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
	err := AddRoute("somesv", func(session *session.Session, route *route.Route, servers map[string]*cluster.Server) (*cluster.Server, error) {
		return nil, nil
	})
	assert.EqualError(t, constants.ErrRouterNotInitialized, err.Error())

	app.router = router.New()
	err = AddRoute("somesv", func(session *session.Session, route *route.Route, servers map[string]*cluster.Server) (*cluster.Server, error) {
		return nil, nil
	})
	assert.NoError(t, err)

	app.running = true
	err = AddRoute("somesv", func(session *session.Session, route *route.Route, servers map[string]*cluster.Server) (*cluster.Server, error) {
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
