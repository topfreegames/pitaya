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

package cluster

import (
	"context"
	"math"
	"testing"
	"time"

	"go.etcd.io/etcd/clientv3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
)

var etcdSDTables = []struct {
	server *Server
}{
	{NewServer("frontend-1", "type1", true, map[string]string{"k1": "v1"})},
	{NewServer("backend-1", "type2", false, map[string]string{"k2": "v2"})},
	{NewServer("backend-2", "type3", false, nil)},
}

var etcdSDTablesMultipleServers = []struct {
	servers []*Server
}{
	{[]*Server{}},
	{[]*Server{
		NewServer("frontend-1", "type1", true, map[string]string{"k1": "v1"}),
		NewServer("backend-1", "type2", false, map[string]string{"k2": "v2"}),
		NewServer("backend-2", "type3", false, nil),
	}},
	{[]*Server{
		NewServer("frontend-1", "type2", true, map[string]string{"k1": "v1"}),
		NewServer("frontend-2", "type2", true, map[string]string{"k2": "v2"}),
		NewServer("frontend-3", "type2", true, map[string]string{"k1": "v1"}),
	}},
}

var etcdSDBlacklistTables = []struct {
	name                string
	server              *Server
	serversToAdd        []*Server
	serverTypeBlacklist []string
}{
	{
		name:   "test1",
		server: NewServer("frontend-1", "type1", true, nil),
		serversToAdd: []*Server{
			NewServer("frontend-1", "type1", true, nil),
		},
		serverTypeBlacklist: nil,
	},
	{
		name:   "test2",
		server: NewServer("frontend-1", "type1", true, nil),
		serversToAdd: []*Server{
			NewServer("backend-1", "type1", false, nil),
			NewServer("backend-2", "type2", false, nil),
			NewServer("backend-3", "type3", false, nil),
		},
		serverTypeBlacklist: []string{"type2"},
	},
	{
		name:   "test3",
		server: NewServer("frontend-1", "type1", true, nil),
		serversToAdd: []*Server{
			NewServer("backend-1", "type1", false, nil),
			NewServer("backend-2", "type2", false, nil),
			NewServer("backend-3", "type3", false, nil),
			NewServer("backend-4", "type4", false, nil),
		},
		serverTypeBlacklist: []string{"type1", "type4"},
	},
}

func getConfig(conf ...*viper.Viper) *config.Config {
	config := config.NewConfig(conf...)
	return config
}

func getEtcdSD(t *testing.T, config *config.Config, server *Server, cli *clientv3.Client) *etcdServiceDiscovery {
	t.Helper()
	appDieChan := make(chan bool)
	e, err := NewEtcdServiceDiscovery(config, server, appDieChan, cli)
	assert.NoError(t, err)
	return e.(*etcdServiceDiscovery)
}

func TestNewEtcdServiceDiscovery(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			assert.NotNil(t, e)
		})
	}
}

func TestEtcdSDBootstrapLease(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			err := e.grantLease()
			assert.NoError(t, err)
			assert.NotEmpty(t, e.leaseID)
		})
	}
}

func TestEtcdSDBootstrapLeaseError(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			err := e.grantLease()
			assert.Error(t, err)
		})
	}
}

func TestEtcdSDBootstrapServer(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.grantLease()
			err := e.bootstrapServer(table.server)
			assert.NoError(t, err)
			v, err := cli.Get(context.TODO(), getKey(table.server.ID, table.server.Type))
			assert.NoError(t, err)
			assert.NotNil(t, v)
			assert.Equal(t, 1, len(v.Kvs))
			generatedSv, ok := e.serverMapByID.Load(table.server.ID)
			assert.True(t, ok)
			assert.Equal(t, table.server, generatedSv)
			val := v.Kvs[0]
			assert.Equal(t, getKey(table.server.ID, table.server.Type), string(val.Key))
			assert.Equal(t, table.server.AsJSONString(), string(val.Value))
		})
	}
}

func TestEtcdSDDeleteServer(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.grantLease()
			err := e.bootstrapServer(table.server)
			assert.NoError(t, err)
			e.deleteServer(table.server.ID)
			e.serverMapByID.Delete(table.server.ID)
			generatedSv, ok := e.serverMapByID.Load(table.server.ID)
			assert.False(t, ok)
			assert.Nil(t, generatedSv)
			_, err = e.GetServersByType(table.server.Type)
			assert.EqualError(t, constants.ErrNoServersAvailableOfType, err.Error())
		})
	}
}

func TestEtcdSDGetKey(t *testing.T) {
	t.Parallel()
	tables := []struct {
		serverType string
		serverID   string
		ret        string
	}{
		{"type1", "id1", "servers/type1/id1"},
		{"t", "1", "servers/t/1"},
	}

	for _, table := range tables {
		t.Run(table.ret, func(t *testing.T) {
			assert.Equal(t, table.ret, getKey(table.serverID, table.serverType))
		})
	}
}

func TestEtcdSDDeleteLocalInvalidServers(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			_, cli := helpers.GetTestEtcd(t)
			e := getEtcdSD(t, config, table.server, cli)
			invalidServer := &Server{
				ID:   "invalid",
				Type: "bla",
			}
			e.addServer(invalidServer)
			e.deleteLocalInvalidServers([]string{table.server.ID})
			inv, err := e.GetServer(invalidServer.ID)
			assert.EqualError(t, constants.ErrNoServerWithID, err.Error())
			assert.Nil(t, inv)
		})
	}
}

func TestEtcdSDGetServer(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.grantLease()
			e.bootstrapServer(table.server)
			sv, err := e.GetServer(table.server.ID)
			assert.NoError(t, err)
			assert.Equal(t, table.server, sv)
		})
	}
}

func TestEtcdSDGetServers(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTablesMultipleServers {
		config := getConfig()
		c, cli := helpers.GetTestEtcd(t)
		defer c.Terminate(t)
		e := getEtcdSD(t, config, &Server{}, cli)
		e.grantLease()
		for _, server := range table.servers {
			e.bootstrapServer(server)
		}
		serverList := e.GetServers()
		assert.ElementsMatch(t, table.servers, serverList)
	}
}

func TestEtcdSDInit(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			conf := viper.New()
			conf.Set("pitaya.cluster.sd.etcd.syncservers.interval", "30ms")
			config := getConfig(conf)
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.Init()
			// should set running
			assert.True(t, e.running)
			// should have a leaseid
			assert.NotEmpty(t, e.leaseID)
			// should register the server
			sv, err := e.GetServer(table.server.ID)
			assert.NoError(t, err)
			assert.Equal(t, table.server, sv)
			// should heartbeat and sync
			time.Sleep(50 * time.Millisecond)
			// TODO may be flaky
			helpers.ShouldEventuallyReturn(t, func() bool {
				return math.Abs(float64(time.Now().Unix()-e.lastSyncTime.Unix())) < 5
			}, true, 50*time.Millisecond, 2*time.Second)
		})
	}
}

func TestEtcdBeforeShutdown(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.Init()
			assert.True(t, e.running)
			e.BeforeShutdown()
			assert.True(t, e.running)
			_, err := cli.Revoke(context.TODO(), e.leaseID)
			assert.Error(t, err)
		})
	}
}

func TestEtcdShutdown(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			config := getConfig()
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.Init()
			assert.True(t, e.running)
			e.Shutdown()
			assert.False(t, e.running)
		})
	}
}

func TestEtcdWatchChangesAddNewServers(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			conf := viper.New()
			conf.Set("pitaya.cluster.sd.etcd.syncservers.interval", "10ms")
			config := getConfig(conf)
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.running = true
			e.bootstrapServer(table.server)
			e.watchEtcdChanges()
			serversBefore, err := e.GetServersByType(table.server.Type)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(serversBefore))
			newServer := &Server{
				ID:       "newID",
				Type:     table.server.Type,
				Frontend: false,
			}
			err = e.addServerIntoEtcd(newServer)
			assert.NoError(t, err)
			ss, err := getServerFromEtcd(e.cli, newServer.Type, newServer.ID)
			assert.NoError(t, err)
			assert.Equal(t, newServer, ss)
			helpers.ShouldEventuallyReturn(t, func() int {
				serversNow, _ := e.GetServersByType(table.server.Type)
				return len(serversNow)
			}, 2)
		})
	}
}

func TestEtcdWatchChangesDeleteServers(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDTables {
		t.Run(table.server.ID, func(t *testing.T) {
			conf := viper.New()
			conf.Set("pitaya.cluster.sd.etcd.syncservers.interval", "10ms")
			config := getConfig(conf)
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.running = true
			e.bootstrapServer(table.server)
			e.watchEtcdChanges()
			serversBefore, err := e.GetServersByType(table.server.Type)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(serversBefore))
			newServer := &Server{
				ID:       "newID",
				Type:     table.server.Type,
				Frontend: false,
			}
			err = e.addServerIntoEtcd(newServer)
			assert.NoError(t, err)
			ss, err := getServerFromEtcd(e.cli, newServer.Type, newServer.ID)
			assert.NoError(t, err)
			assert.Equal(t, newServer, ss)
			helpers.ShouldEventuallyReturn(t, func() int {
				serversNow, _ := e.GetServersByType(table.server.Type)
				return len(serversNow)
			}, 2)
			_, err = cli.Delete(context.TODO(), getKey(newServer.ID, newServer.Type))
			assert.NoError(t, err)
			helpers.ShouldEventuallyReturn(t, func() int {
				serversNow, _ := e.GetServersByType(table.server.Type)
				return len(serversNow)
			}, 1)
		})
	}
}

func TestEtcdWatchChangesWithBlacklist(t *testing.T) {
	t.Parallel()
	for _, table := range etcdSDBlacklistTables {
		t.Run(table.name, func(t *testing.T) {
			conf := viper.New()
			conf.Set("pitaya.cluster.sd.etcd.syncservers.interval", "10ms")
			conf.Set("pitaya.cluster.sd.etcd.serverTypeBlacklist", table.serverTypeBlacklist)
			config := getConfig(conf)
			c, cli := helpers.GetTestEtcd(t)
			defer c.Terminate(t)
			e := getEtcdSD(t, config, table.server, cli)
			e.running = true
			_ = e.bootstrapServer(table.server)
			e.watchEtcdChanges()

			serversBefore, err := e.GetServersByType(table.server.Type)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(serversBefore))

			// Add all servers to ETCD
			for _, serverToAdd := range table.serversToAdd {
				_, err := cli.Put(
					context.TODO(),
					getKey(serverToAdd.ID, serverToAdd.Type),
					serverToAdd.AsJSONString(),
				)
				assert.NoError(t, err)
			}

			numRequiredServersInSd := func() int {
				num := 0
				for _, serverToAdd := range table.serversToAdd {
					if !e.isServerTypeBlacklisted(serverToAdd.Type) {
						num++
					}
				}
				return num + 1 // The server itself is always added to the cache, that's why we add one here
			}

			numServersInSd := func() int {
				num := 0

				_, err := e.GetServer(table.server.ID)
				if err == nil {
					num++
				}
				for _, serverToAdd := range table.serversToAdd {
					_, err := e.GetServer(serverToAdd.ID)
					if err == nil {
						num++
					}
				}
				return num
			}

			helpers.ShouldEventuallyReturn(t, numServersInSd, numRequiredServersInSd())
		})
	}
}

func TestParallelGetter(t *testing.T) {
	c, cli := helpers.GetTestEtcd(t)
	defer c.Terminate(t)

	serversToAdd := []*Server{
		NewServer("frontend-1", "type1", true, nil),
		NewServer("frontend-2", "type2", true, nil),
		NewServer("frontend-3", "type3", true, nil),
		NewServer("frontend-4", "type4", true, nil),
		NewServer("frontend-5", "type5", true, nil),
		NewServer("frontend-6", "type6", true, nil),
		NewServer("frontend-7", "type7", true, nil),
		NewServer("frontend-8", "type8", true, nil),
		NewServer("frontend-9", "type9", true, nil),
	}

	// Add server
	for _, serverToAdd := range serversToAdd {
		_, err := cli.Put(
			context.TODO(),
			getKey(serverToAdd.ID, serverToAdd.Type),
			serverToAdd.AsJSONString(),
		)
		assert.NoError(t, err)
	}

	parallelGetter := newParallelGetter(cli, 5)
	for _, serverToAdd := range serversToAdd {
		parallelGetter.addWork(serverToAdd.Type, serverToAdd.ID)
	}

	servers := parallelGetter.waitAndGetResult()
	assert.Equal(t, len(serversToAdd), len(servers))

	// We add the returned servers to a map first, since the results may be out the order they were added.
	serversMap := map[string]*Server{}
	for _, sv := range servers {
		_, ok := serversMap[sv.ID]
		assert.False(t, ok)
		serversMap[sv.ID] = sv
	}

	for _, serverToAdd := range serversToAdd {
		sv, ok := serversMap[serverToAdd.ID]
		assert.True(t, ok)
		assert.Equal(t, serverToAdd.Type, sv.Type)
		assert.Equal(t, serverToAdd.ID, sv.ID)
		assert.Equal(t, serverToAdd.Frontend, sv.Frontend)
	}
}
