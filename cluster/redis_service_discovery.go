package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
)

const prefix = "server:"

type RedisServiceDiscovery struct {
	redisClient                    *redis.Client
	watchPubSub                    *redis.PubSub
	server                         *Server
	appDieChan                     chan bool
	localCacheLock                 sync.RWMutex
	serverMapByType                map[string]map[string]*Server
	serverMapByID                  map[string]*Server
	redisTTL                       time.Duration
	redisSyncInterval              time.Duration
	redisRefreshExpirationInterval time.Duration
	quitChan                       chan struct{}
	listeners                      []SDListener
}

func NewRedisServiceDiscovery(
	redisAddr, username, password string,
	redisTTL time.Duration,
	redisSyncInterval time.Duration,
	server *Server,
	appDieChan chan bool,
) ServiceDiscovery {
	client := redis.NewClient(&redis.Options{
		Addr:            redisAddr,
		Username:        username,
		Password:        password,
		DB:              0,
		MaxRetries:      0,
		MinRetryBackoff: 0,
		MaxRetryBackoff: 0,
		DialTimeout:     0,
		ReadTimeout:     0,
		WriteTimeout:    0,
		TLSConfig:       nil,
	})
	return &RedisServiceDiscovery{
		server:                         server,
		redisClient:                    client,
		appDieChan:                     appDieChan,
		serverMapByType:                map[string]map[string]*Server{},
		serverMapByID:                  map[string]*Server{},
		redisTTL:                       redisTTL,
		redisSyncInterval:              redisSyncInterval,
		redisRefreshExpirationInterval: redisTTL / 3,
		quitChan:                       make(chan struct{}),
	}
}

func (r *RedisServiceDiscovery) GetServersByType(serverType string) (map[string]*Server, error) {
	if err := r.fillOutLocalCache(); err != nil {
		return nil, fmt.Errorf("fill out local cache: %w", err)
	}
	r.localCacheLock.RLock()
	defer r.localCacheLock.RUnlock()
	if serverIdToServer, ok := r.serverMapByType[serverType]; ok && len(serverIdToServer) > 0 {
		// Create a new map to avoid concurrent read and write access to the
		// map, this also prevents accidental changes to the list of servers
		// kept by the service discovery.
		ret := make(map[string]*Server, len(r.serverMapByType))
		for k, v := range r.serverMapByType[serverType] {
			ret[k] = v
		}
		return ret, nil
	}
	return nil, constants.ErrNoServersAvailableOfType
}

func (r *RedisServiceDiscovery) GetServer(id string) (*Server, error) {
	if err := r.fillOutLocalCache(); err != nil {
		return nil, fmt.Errorf("fill out local cache: %w", err)
	}
	if sv, ok := r.serverMapByID[id]; ok {
		return sv, nil
	}
	return nil, constants.ErrNoServerWithID
}

func (r *RedisServiceDiscovery) GetServers() []*Server {
	if err := r.fillOutLocalCache(); err != nil {
		logger.Log.Errorf("failed to fill local cache: %s", err)
		return nil
	}
	ret := make([]*Server, 0)
	for _, server := range r.serverMapByID {
		ret = append(ret, server)
	}
	return ret
}

func (r *RedisServiceDiscovery) fillOutLocalCache() error {
	if !r.isLocalCacheEmpty() {
		return nil
	}

	if err := r.updateLocalCache(); err != nil {
		r.appDieChan <- true
		return fmt.Errorf("update local cache: %w", err)
	}

	return nil
}

func (r *RedisServiceDiscovery) AddListener(listener SDListener) {
	r.listeners = append(r.listeners, listener)
}

func (r *RedisServiceDiscovery) notifyListeners(act Action, sv *Server) {
	for _, l := range r.listeners {
		if act == DEL {
			l.RemoveServer(sv)
		} else if act == ADD {
			l.AddServer(sv)
		}
	}
}

func (r *RedisServiceDiscovery) isLocalCacheEmpty() bool {
	r.localCacheLock.RLock()
	defer r.localCacheLock.RUnlock()
	return len(r.serverMapByType) == 0
}

func (r *RedisServiceDiscovery) Init() error {
	logger.Log.Debug("intializing redis service discovery")
	ctx := context.TODO()
	if err := r.addServerToRedis(ctx); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	r.watchPubSub = r.redisClient.PSubscribe(ctx, fmt.Sprintf("%s*", getChannelPrefix()))

	go func() {
		// Get the Channel to use
		// Iterate any messages sent on the channel
		channel := r.watchPubSub.Channel()

		for msg := range channel {
			r.processMessage(msg)
		}
	}()

	go r.redisSyncRoutine()
	go r.redisRefreshExpirationRoutine()
	return nil
}

func (r *RedisServiceDiscovery) redisSyncRoutine() {
	// adding a random factor to sync interval to avoid multiple instances running at same time
	rand.Seed(time.Now().UnixNano())
	syncInterval := r.redisSyncInterval + (time.Second * time.Duration(rand.Intn(60)))
	ticker := time.NewTicker(syncInterval)

	for {
		select {
		case <-ticker.C:
			if err := r.updateLocalCache(); err != nil {
				logger.Log.Error("failed to update local cache: %s", err)
				r.appDieChan <- true
				return
			}

		case <-r.quitChan:
			logger.Log.Debug("shutting down redis sync routine")
			ticker.Stop()
			return
		}
	}
}

func (r *RedisServiceDiscovery) redisRefreshExpirationRoutine() {
	ticker := time.NewTicker(r.redisRefreshExpirationInterval)
	for {
		select {
		case <-ticker.C:
			logger.Log.Debug("running refresh expiration routine")
			ctx := context.TODO()
			res := r.redisClient.Expire(ctx, getServerRedisKey(r.server), r.redisTTL)
			if err := res.Err(); err != nil {
				logger.Log.Errorf("failed to refresh expiration time: %s", err)
				r.appDieChan <- true
				return
			}
			if ok := res.Val(); !ok {
				logger.Log.Warnf("failed to update expiration time for server: %s", r.server.AsJSONString())
			}
		case <-r.quitChan:
			logger.Log.Debug("shutting down redis refresh expiration routine")
			ticker.Stop()
			return
		}
	}
}

func (r *RedisServiceDiscovery) getAllServersFromRedis() ([]*Server, error) {
	ctx := context.TODO()
	allKeysRes := r.redisClient.Keys(ctx, fmt.Sprintf("%s*", prefix))
	if err := allKeysRes.Err(); err != nil {
		r.appDieChan <- true
		return nil, fmt.Errorf("list all keys: %w", err)
	}

	var servers []*Server

	allKeys := allKeysRes.Val()
	for _, key := range allKeys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		key = key[len(prefix):]

		server := &Server{}
		if err := json.Unmarshal([]byte(key), server); err != nil {
			logger.Log.Warnf("failed to parse server from redis cache. %s", err)
			continue
		}
		if server.ID == "" {
			logger.Log.Warnf("server has empty id. Hostname %s", server.Hostname)
			continue
		}

		servers = append(servers, server)
	}
	return servers, nil
}

func (r *RedisServiceDiscovery) updateLocalCache() error {
	r.localCacheLock.Lock()
	defer r.localCacheLock.Unlock()

	logger.Log.Debug("updating local cache")

	servers, err := r.getAllServersFromRedis()
	if err != nil {
		return fmt.Errorf("get all servers from redis: %w", err)
	}

	for _, server := range servers {
		logger.Log.Debugf("adding server %s", server)
		r.addServerToLocalCache(server)
	}

	r.removeLocalInvalidServers(servers)

	return nil
}

func hasServer(servers []*Server, wantServer *Server) bool {
	for _, server := range servers {
		if server.ID == wantServer.ID {
			return true
		}
	}
	return false
}

func (r *RedisServiceDiscovery) removeLocalInvalidServers(allValidServers []*Server) {
	for _, server := range r.serverMapByID {
		if !hasServer(allValidServers, server) {
			logger.Log.Warnf("removing invalid local server %s", server.ID)
			r.removeServerFromLocalCache(server)
		}
	}
}

func (r *RedisServiceDiscovery) processMessage(msg *redis.Message) {
	channelPrefix := getChannelPrefix()
	keyValue := msg.Channel[len(channelPrefix):]

	server := &Server{}
	if err := json.Unmarshal([]byte(keyValue), server); err != nil {
		logger.Log.Warnf("failed to parse server from redis keyspace notification. %s", err)
		return
	}

	if msg.Payload == "expired" || msg.Payload == "del" {
		r.removeServerFromLocalCache(server)
	} else if msg.Payload == "set" {
		r.addServerToLocalCache(server)
	}
}

func (r *RedisServiceDiscovery) removeServerFromLocalCache(sv *Server) {
	r.localCacheLock.Lock()
	defer r.localCacheLock.Unlock()

	if _, ok := r.serverMapByID[sv.ID]; ok {
		logger.Log.Debugf("removing server from local cache: id=%s, type=%s", sv.ID, sv.Type)
		delete(r.serverMapByID, sv.ID)
		if svMap, ok := r.serverMapByType[sv.Type]; ok {
			delete(svMap, sv.ID)
		}
		r.notifyListeners(DEL, sv)
	}
}

func (r *RedisServiceDiscovery) addServerToLocalCache(sv *Server) {
	r.localCacheLock.Lock()
	defer r.localCacheLock.Unlock()

	if _, ok := r.serverMapByID[sv.ID]; ok {
		return
	}

	logger.Log.Debugf("adding server to local cache: id=%s, type=%s", sv.ID, sv.Type)
	r.serverMapByID[sv.ID] = sv
	mapSvByType, ok := r.serverMapByType[sv.Type]
	if !ok {
		mapSvByType = make(map[string]*Server)
		r.serverMapByType[sv.Type] = mapSvByType
	}
	mapSvByType[sv.ID] = sv

	r.notifyListeners(ADD, sv)
}

func (r *RedisServiceDiscovery) AfterInit() {
	// No implementation.
}

func (r *RedisServiceDiscovery) BeforeShutdown() {
	logger.Log.Debug("removing server from redis")
	close(r.quitChan)
	r.removeServerFromRedis()
}

func (r *RedisServiceDiscovery) Shutdown() error {
	logger.Log.Debug("shutting down redis client")
	if err := r.watchPubSub.Close(); err != nil {
		return fmt.Errorf("redis pubsub close: %w", err)
	}
	if err := r.redisClient.Close(); err != nil {
		return fmt.Errorf("redis client close: %w", err)
	}
	return nil
}

func (r *RedisServiceDiscovery) removeServerFromRedis() {
	ctx := context.TODO()
	res := r.redisClient.Del(ctx, getServerRedisKey(r.server))
	if err := res.Err(); err != nil {
		logger.Log.Errorf("failed to remove server from redis: %s", err)
	}
}

func (r *RedisServiceDiscovery) addServerToRedis(ctx context.Context) error {
	redisKey := getServerRedisKey(r.server)
	status := r.redisClient.Set(ctx, redisKey, "not used", r.redisTTL)
	if err := status.Err(); err != nil {
		return fmt.Errorf("add server to redis: %w", err)
	}
	return nil
}

func getServerRedisKey(server *Server) string {
	return fmt.Sprintf("%s%s", prefix, server.AsJSONString())
}

func getChannelPrefix() string {
	return fmt.Sprintf("__keyspace@0__:%s", prefix)
}
