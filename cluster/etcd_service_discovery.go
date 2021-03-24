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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/util"
)

type etcdServiceDiscovery struct {
	cli                    *clientv3.Client
	config                 *config.Config
	syncServersInterval    time.Duration
	heartbeatTTL           time.Duration
	logHeartbeat           bool
	lastHeartbeatTime      time.Time
	leaseID                clientv3.LeaseID
	mapByTypeLock          sync.RWMutex
	serverMapByType        map[string]map[string]*Server
	serverMapByID          sync.Map
	etcdEndpoints          []string
	etcdUser               string
	etcdPass               string
	etcdPrefix             string
	etcdDialTimeout        time.Duration
	running                bool
	server                 *Server
	stopChan               chan bool
	stopLeaseChan          chan bool
	lastSyncTime           time.Time
	listeners              []SDListener
	revokeTimeout          time.Duration
	grantLeaseTimeout      time.Duration
	grantLeaseMaxRetries   int
	grantLeaseInterval     time.Duration
	shutdownDelay          time.Duration
	appDieChan             chan bool
	serverTypesBlacklist   []string
	syncServersParallelism int
}

// NewEtcdServiceDiscovery ctor
func NewEtcdServiceDiscovery(
	config *config.Config,
	server *Server,
	appDieChan chan bool,
	cli ...*clientv3.Client,
) (ServiceDiscovery, error) {
	var client *clientv3.Client
	if len(cli) > 0 {
		client = cli[0]
	}
	sd := &etcdServiceDiscovery{
		config:          config,
		running:         false,
		server:          server,
		serverMapByType: make(map[string]map[string]*Server),
		listeners:       make([]SDListener, 0),
		stopChan:        make(chan bool),
		stopLeaseChan:   make(chan bool),
		appDieChan:      appDieChan,
		cli:             client,
	}

	sd.configure()

	return sd, nil
}

func (sd *etcdServiceDiscovery) configure() {
	sd.etcdEndpoints = sd.config.GetStringSlice("pitaya.cluster.sd.etcd.endpoints")
	sd.etcdUser = sd.config.GetString("pitaya.cluster.sd.etcd.user")
	sd.etcdPass = sd.config.GetString("pitaya.cluster.sd.etcd.pass")
	sd.etcdDialTimeout = sd.config.GetDuration("pitaya.cluster.sd.etcd.dialtimeout")
	sd.etcdPrefix = sd.config.GetString("pitaya.cluster.sd.etcd.prefix")
	sd.heartbeatTTL = sd.config.GetDuration("pitaya.cluster.sd.etcd.heartbeat.ttl")
	sd.logHeartbeat = sd.config.GetBool("pitaya.cluster.sd.etcd.heartbeat.log")
	sd.syncServersInterval = sd.config.GetDuration("pitaya.cluster.sd.etcd.syncservers.interval")
	sd.revokeTimeout = sd.config.GetDuration("pitaya.cluster.sd.etcd.revoke.timeout")
	sd.grantLeaseTimeout = sd.config.GetDuration("pitaya.cluster.sd.etcd.grantlease.timeout")
	sd.grantLeaseMaxRetries = sd.config.GetInt("pitaya.cluster.sd.etcd.grantlease.maxretries")
	sd.grantLeaseInterval = sd.config.GetDuration("pitaya.cluster.sd.etcd.grantlease.retryinterval")
	sd.shutdownDelay = sd.config.GetDuration("pitaya.cluster.sd.etcd.shutdown.delay")
	sd.serverTypesBlacklist = sd.config.GetStringSlice("pitaya.cluster.sd.etcd.servertypeblacklist")
	sd.syncServersParallelism = sd.config.GetInt("pitaya.cluster.sd.etcd.syncserversparallelism")

	if len(sd.serverTypesBlacklist) > 0 {
		logger.Log.Warnf("using server types blacklist: %s", sd.serverTypesBlacklist)
	}
}

func (sd *etcdServiceDiscovery) watchLeaseChan(c <-chan *clientv3.LeaseKeepAliveResponse) {
	failedGrantLeaseAttempts := 0
	for {
		select {
		case <-sd.stopChan:
			return
		case <-sd.stopLeaseChan:
			return
		case leaseKeepAliveResponse, ok := <-c:
                        if !ok {
				logger.Log.Error("ETCD lease KeepAlive died, retrying in 10 seconds")
                                time.Sleep(10000 * time.Millisecond)
			}
			if leaseKeepAliveResponse != nil {
				if sd.logHeartbeat {
					logger.Log.Debugf("sd: etcd lease %x renewed", leaseKeepAliveResponse.ID)
				}
				failedGrantLeaseAttempts = 0
				continue
			}
			logger.Log.Warn("sd: error renewing etcd lease, reconfiguring")
			for {
				err := sd.renewLease()
				if err != nil {
					failedGrantLeaseAttempts = failedGrantLeaseAttempts + 1
					if err == constants.ErrEtcdGrantLeaseTimeout {
						logger.Log.Warn("sd: timed out trying to grant etcd lease")
						if sd.appDieChan != nil {
							sd.appDieChan <- true
						}
						return
					}
					if failedGrantLeaseAttempts >= sd.grantLeaseMaxRetries {
						logger.Log.Warn("sd: exceeded max attempts to renew etcd lease")
						if sd.appDieChan != nil {
							sd.appDieChan <- true
						}
						return
					}
					logger.Log.Warnf("sd: error granting etcd lease, will retry in %d seconds", uint64(sd.grantLeaseInterval.Seconds()))
					time.Sleep(sd.grantLeaseInterval)
					continue
				}
				return
			}
		}
	}
}

// renewLease reestablishes connection with etcd
func (sd *etcdServiceDiscovery) renewLease() error {
	c := make(chan error)
	go func() {
		defer close(c)
		logger.Log.Infof("waiting for etcd lease")
		err := sd.grantLease()
		if err != nil {
			c <- err
			return
		}
		err = sd.bootstrapServer(sd.server)
		c <- err
	}()
	select {
	case err := <-c:
		return err
	case <-time.After(sd.grantLeaseTimeout):
		return constants.ErrEtcdGrantLeaseTimeout
	}
}

func (sd *etcdServiceDiscovery) grantLease() error {
	// grab lease
	l, err := sd.cli.Grant(context.TODO(), int64(sd.heartbeatTTL.Seconds()))
	if err != nil {
		return err
	}
	sd.leaseID = l.ID
	logger.Log.Debugf("sd: got leaseID: %x", l.ID)
	// this will keep alive forever, when channel c is closed
	// it means we probably have to rebootstrap the lease
	c, err := sd.cli.KeepAlive(context.TODO(), sd.leaseID)
	if err != nil {
		return err
	}
	// need to receive here as per etcd docs
	<-c
	go sd.watchLeaseChan(c)
	return nil
}

func (sd *etcdServiceDiscovery) addServerIntoEtcd(server *Server) error {
	_, err := sd.cli.Put(
		context.TODO(),
		getKey(server.ID, server.Type),
		server.AsJSONString(),
		clientv3.WithLease(sd.leaseID),
	)
	return err
}

func (sd *etcdServiceDiscovery) bootstrapServer(server *Server) error {
	if err := sd.addServerIntoEtcd(server); err != nil {
		return err
	}

	sd.SyncServers()
	return nil
}

// AddListener adds a listener to etcd service discovery
func (sd *etcdServiceDiscovery) AddListener(listener SDListener) {
	sd.listeners = append(sd.listeners, listener)
}

// AfterInit executes after Init
func (sd *etcdServiceDiscovery) AfterInit() {
}

func (sd *etcdServiceDiscovery) notifyListeners(act Action, sv *Server) {
	for _, l := range sd.listeners {
		if act == DEL {
			l.RemoveServer(sv)
		} else if act == ADD {
			l.AddServer(sv)
		}
	}
}

func (sd *etcdServiceDiscovery) writeLockScope(f func()) {
	sd.mapByTypeLock.Lock()
	defer sd.mapByTypeLock.Unlock()
	f()
}

func (sd *etcdServiceDiscovery) deleteServer(serverID string) {
	if actual, ok := sd.serverMapByID.Load(serverID); ok {
		sv := actual.(*Server)
		sd.serverMapByID.Delete(sv.ID)
		sd.writeLockScope(func() {
			if svMap, ok := sd.serverMapByType[sv.Type]; ok {
				delete(svMap, sv.ID)
			}
		})
		sd.notifyListeners(DEL, sv)
	}
}

func (sd *etcdServiceDiscovery) deleteLocalInvalidServers(actualServers []string) {
	sd.serverMapByID.Range(func(key interface{}, value interface{}) bool {
		k := key.(string)
		if !util.SliceContainsString(actualServers, k) {
			logger.Log.Warnf("deleting invalid local server %s", k)
			sd.deleteServer(k)
		}
		return true
	})
}

func getKey(serverID, serverType string) string {
	return fmt.Sprintf("servers/%s/%s", serverType, serverID)
}

func getServerFromEtcd(cli *clientv3.Client, serverType, serverID string) (*Server, error) {
	svKey := getKey(serverID, serverType)
	svEInfo, err := cli.Get(context.TODO(), svKey)
	if err != nil {
		return nil, fmt.Errorf("error getting server: %s from etcd, error: %s", svKey, err.Error())
	}
	if len(svEInfo.Kvs) == 0 {
		return nil, fmt.Errorf("didn't found server: %s in etcd", svKey)
	}
	return parseServer(svEInfo.Kvs[0].Value)
}

// GetServersByType returns a slice with all the servers of a certain type
func (sd *etcdServiceDiscovery) GetServersByType(serverType string) (map[string]*Server, error) {
	sd.mapByTypeLock.RLock()
	defer sd.mapByTypeLock.RUnlock()
	if m, ok := sd.serverMapByType[serverType]; ok && len(m) > 0 {
		// Create a new map to avoid concurrent read and write access to the
		// map, this also prevents accidental changes to the list of servers
		// kept by the service discovery.
		ret := make(map[string]*Server, len(sd.serverMapByType))
		for k, v := range sd.serverMapByType[serverType] {
			ret[k] = v
		}
		return ret, nil
	}
	return nil, constants.ErrNoServersAvailableOfType
}

// GetServers returns a slice with all the servers
func (sd *etcdServiceDiscovery) GetServers() []*Server {
	ret := make([]*Server, 0)
	sd.serverMapByID.Range(func(k, v interface{}) bool {
		ret = append(ret, v.(*Server))
		return true
	})
	return ret
}

func (sd *etcdServiceDiscovery) bootstrap() error {
	if err := sd.grantLease(); err != nil {
		return err
	}

	if err := sd.bootstrapServer(sd.server); err != nil {
		return err
	}

	return nil
}

// GetServer returns a server given it's id
func (sd *etcdServiceDiscovery) GetServer(id string) (*Server, error) {
	if sv, ok := sd.serverMapByID.Load(id); ok {
		return sv.(*Server), nil
	}
	return nil, constants.ErrNoServerWithID
}

func (sd *etcdServiceDiscovery) InitETCDClient() error {
        logger.Log.Infof("Initializing ETCD client")
        var cli *clientv3.Client
        var err error
        config := clientv3.Config{
		Endpoints:   sd.etcdEndpoints,
		DialTimeout: sd.etcdDialTimeout,
        }
        if sd.etcdUser != "" && sd.etcdPass != "" {
		config.Username = sd.etcdUser
                config.Password = sd.etcdPass
        }
        cli, err = clientv3.New(config)
        if err != nil {
		logger.Log.Errorf("error initializing etcd client: %s", err.Error())
                return err
        }
        sd.cli = cli

        // namespaced etcd :)
        sd.cli.KV = namespace.NewKV(sd.cli.KV, sd.etcdPrefix)
        sd.cli.Watcher = namespace.NewWatcher(sd.cli.Watcher, sd.etcdPrefix)
        sd.cli.Lease = namespace.NewLease(sd.cli.Lease, sd.etcdPrefix)
        return nil
}

// Init starts the service discovery client
func (sd *etcdServiceDiscovery) Init() error {
	sd.running = true
	var err error

        if sd.cli == nil { 
		sd.InitETCDClient()
        } else {
		sd.cli.KV = namespace.NewKV(sd.cli.KV, sd.etcdPrefix)
		sd.cli.Watcher = namespace.NewWatcher(sd.cli.Watcher, sd.etcdPrefix)
		sd.cli.Lease = namespace.NewLease(sd.cli.Lease, sd.etcdPrefix)
	}

	if err = sd.bootstrap(); err != nil {
		return err
	}

	// update servers
	syncServersTicker := time.NewTicker(sd.syncServersInterval)
	go func() {
		for sd.running {
			select {
			case <-syncServersTicker.C:
				err := sd.SyncServers()
				if err != nil {
					logger.Log.Errorf("error resyncing servers: %s", err.Error())
				}
			case <-sd.stopChan:
				return
			}
		}
	}()
 
	go sd.watchEtcdChanges()
	return nil
}

func parseEtcdKey(key string) (string, string, error) {
	splittedServer := strings.Split(key, "/")
	if len(splittedServer) != 3 {
		return "", "", fmt.Errorf("error parsing etcd key %s (server name can't contain /)", key)
	}
	svType := splittedServer[1]
	svID := splittedServer[2]
	return svType, svID, nil
}

func parseServer(value []byte) (*Server, error) {
	var sv *Server
	err := json.Unmarshal(value, &sv)
	if err != nil {
		logger.Log.Warnf("failed to load server %s, error: %s", sv, err.Error())
	}
	return sv, nil
}

func (sd *etcdServiceDiscovery) printServers() {
	sd.mapByTypeLock.RLock()
	defer sd.mapByTypeLock.RUnlock()
	for k, v := range sd.serverMapByType {
		logger.Log.Debugf("type: %s, servers: %+v", k, v)
	}
}

// Struct that encapsulates a parallel/concurrent etcd get
// it spawns goroutines and receives work requests through a channel
type parallelGetterWork struct {
	serverType string
	serverID   string
}

type parallelGetter struct {
	cli         *clientv3.Client
	numWorkers  int
	wg          *sync.WaitGroup
	resultMutex sync.Mutex
	result      *[]*Server
	workChan    chan parallelGetterWork
}

func newParallelGetter(cli *clientv3.Client, numWorkers int) parallelGetter {
	if numWorkers <= 0 {
		numWorkers = 10
	}
	p := parallelGetter{
		cli:        cli,
		numWorkers: numWorkers,
		workChan:   make(chan parallelGetterWork),
		wg:         new(sync.WaitGroup),
		result:     new([]*Server),
	}
	p.start()
	return p
}

func (p *parallelGetter) start() {
	for i := 0; i < p.numWorkers; i++ {
		go func() {
			for work := range p.workChan {
				logger.Log.Debugf("loading info from missing server: %s/%s", work.serverType, work.serverID)

				sv, err := getServerFromEtcd(p.cli, work.serverType, work.serverID)
				if err != nil {
					logger.Log.Errorf("error getting server from etcd: %s, error: %s", work.serverID, err.Error())
					p.wg.Done()
					continue
				}

				p.resultMutex.Lock()
				*p.result = append(*p.result, sv)
				p.resultMutex.Unlock()

				p.wg.Done()
			}
		}()
	}
}

func (p *parallelGetter) waitAndGetResult() []*Server {
	p.wg.Wait()
	close(p.workChan)
	return *p.result
}

func (p *parallelGetter) addWork(serverType, serverID string) {
	p.wg.Add(1)
	p.workChan <- parallelGetterWork{
		serverType: serverType,
		serverID:   serverID,
	}
}

// SyncServers gets all servers from etcd
func (sd *etcdServiceDiscovery) SyncServers() error {
	keys, err := sd.cli.Get(
		context.TODO(),
		"servers/",
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
	)
	if err != nil {
		return err
	}

	// delete invalid servers (local ones that are not in etcd)
	var allIds = make([]string, 0)

	// Spawn worker goroutines that will work in parallel
	parallelGetter := newParallelGetter(sd.cli, sd.syncServersParallelism)

	for _, kv := range keys.Kvs {
		svType, svID, err := parseEtcdKey(string(kv.Key))
		if err != nil {
			logger.Log.Warnf("failed to parse etcd key %s, error: %s", kv.Key, err.Error())
			continue
		}

		// Check whether the server type is blacklisted or not
		if sd.isServerTypeBlacklisted(svType) && svID != sd.server.ID {
			logger.Log.Debug("ignoring blacklisted server type '%s'", svType)
			continue
		}

		allIds = append(allIds, svID)

		if _, ok := sd.serverMapByID.Load(svID); !ok {
			// Add new work to the channel
			parallelGetter.addWork(svType, svID)
		}
	}

	// Wait until all goroutines are finished
	servers := parallelGetter.waitAndGetResult()

	for _, server := range servers {
		logger.Log.Debugf("adding server %s", server)
		sd.addServer(server)
	}

	sd.deleteLocalInvalidServers(allIds)

	sd.printServers()
	sd.lastSyncTime = time.Now()
	return nil
}

// BeforeShutdown executes before shutting down and will remove the server from the list
func (sd *etcdServiceDiscovery) BeforeShutdown() {
	sd.revoke()
	time.Sleep(sd.shutdownDelay) // Sleep for a short while to ensure shutdown has propagated
}

// Shutdown executes on shutdown and will clean etcd
func (sd *etcdServiceDiscovery) Shutdown() error {
	sd.running = false
	close(sd.stopChan)
	return nil
}

// revoke prevents Pitaya from crashing when etcd is not available
func (sd *etcdServiceDiscovery) revoke() error {
	close(sd.stopLeaseChan)
	c := make(chan error)
	defer close(c)
	go func() {
		logger.Log.Debug("waiting for etcd revoke")
		_, err := sd.cli.Revoke(context.TODO(), sd.leaseID)
		c <- err
		logger.Log.Debug("finished waiting for etcd revoke")
	}()
	select {
	case err := <-c:
		return err // completed normally
	case <-time.After(sd.revokeTimeout):
		logger.Log.Warn("timed out waiting for etcd revoke")
		return nil // timed out
	}
}

func (sd *etcdServiceDiscovery) addServer(sv *Server) {
	if _, loaded := sd.serverMapByID.LoadOrStore(sv.ID, sv); !loaded {
		sd.writeLockScope(func() {
			mapSvByType, ok := sd.serverMapByType[sv.Type]
			if !ok {
				mapSvByType = make(map[string]*Server)
				sd.serverMapByType[sv.Type] = mapSvByType
			}
			mapSvByType[sv.ID] = sv
		})
		if sv.ID != sd.server.ID {
			sd.notifyListeners(ADD, sv)
		}
	}
}

func (sd *etcdServiceDiscovery) watchEtcdChanges() {
	w := sd.cli.Watch(context.Background(), "servers/", clientv3.WithPrefix())

	go func(chn clientv3.WatchChan) {
		for sd.running {
			select {
			case wResp, ok := <-chn:
				if wResp.Err() != nil {
					logger.Log.Warnf("etcd watcher response error: %s", wResp.Err())
                                        time.Sleep(100 * time.Millisecond)
				}
				if !ok {
					logger.Log.Error("etcd watcher died, retrying to watch in 1 second")
                                        time.Sleep(1000 * time.Millisecond)
	                                sd.InitETCDClient()
                                        chn = sd.cli.Watch(context.Background(), "servers/", clientv3.WithPrefix())
				}
				for _, ev := range wResp.Events {
					svType, svID, err := parseEtcdKey(string(ev.Kv.Key))
					if err != nil {
						logger.Log.Warnf("failed to parse key from etcd: %s", ev.Kv.Key)
						continue
					}

					if sd.isServerTypeBlacklisted(svType) && sd.server.ID != svID {
						continue
					}

					switch ev.Type {
					case clientv3.EventTypePut:
						var sv *Server
						var err error
						if sv, err = parseServer(ev.Kv.Value); err != nil {
							logger.Log.Errorf("Failed to parse server from etcd: %v", err)
							continue
						}

						sd.addServer(sv)
						logger.Log.Debugf("server %s added", ev.Kv.Key)
						sd.printServers()
					case clientv3.EventTypeDelete:
						sd.deleteServer(svID)
						logger.Log.Debugf("server %s deleted", svID)
						sd.printServers()
					}
				}
			case <-sd.stopChan:
				return
			}
		}
	}(w)
}

func (sd *etcdServiceDiscovery) isServerTypeBlacklisted(svType string) bool {
	for _, blacklistedSv := range sd.serverTypesBlacklist {
		if blacklistedSv == svType {
			return true
		}
	}
	return false
}
