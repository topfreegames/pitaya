package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/logger"

	"github.com/topfreegames/pitaya/constants"

	"github.com/go-redis/redis/v8"
)

//
// Redis format:
//
// - Well store each new server as a key in redis with an expiration in the following format:
//    - server-type/server-id, server-json
//
// - We'll store a cached hashmap of a specific server type on redis as well, in order to avoid a scan
//   over all the set of keys:
//     - hashmap<server-type/server-id, server-json>
//
type RedisServiceDiscovery struct {
	redisClient *redis.Client
	server      *Server
	appDieChan  chan bool

	mapByTypeLock   sync.RWMutex
	serverMapByType map[string]map[string]*Server
	serverMapByID   sync.Map
}

func NewRedisServiceDiscovery(
	redisAddr, username, password string,
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
		server:      server,
		redisClient: client,
		appDieChan:  appDieChan,
	}
}

func (r *RedisServiceDiscovery) GetServersByType(serverType string) (map[string]*Server, error) {
	if err := r.fillOutLocalCache(); err != nil {
		return nil, fmt.Errorf("fill out local cache: %w", err)
	}

	r.mapByTypeLock.RLock()
	defer r.mapByTypeLock.RUnlock()
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

	if sv, ok := r.serverMapByID.Load(id); ok {
		return sv.(*Server), nil
	}
	return nil, constants.ErrNoServerWithID
}

func (r *RedisServiceDiscovery) GetServers() []*Server {
	if err := r.fillOutLocalCache(); err != nil {
		logger.Log.Errorf("failed to fill local cache: %s", err)
		return nil
	}

	ret := make([]*Server, 0)
	r.serverMapByID.Range(func(k, v interface{}) bool {
		ret = append(ret, v.(*Server))
		return true
	})

	return ret
}

func (r *RedisServiceDiscovery) fillOutLocalCache() error {
	if !r.isLocalCacheEmpty() {
		return nil
	}

	ctx := context.TODO()
	allKeysRes := r.redisClient.Keys(ctx, "*")

	if err := allKeysRes.Err(); err != nil {
		r.appDieChan <- true
		return fmt.Errorf("list all keys: %w", err)
	}

	allKeys := allKeysRes.Val()
	for _, key := range allKeys {
		var server *Server
		if err := json.Unmarshal([]byte(key), server); err != nil {

		}
	}

	return nil
}

func (r *RedisServiceDiscovery) AddListener(listener SDListener) {
	panic("implement me")
}

func (r *RedisServiceDiscovery) isLocalCacheEmpty() bool {
	r.mapByTypeLock.RLock()
	defer r.mapByTypeLock.RUnlock()
	return len(r.serverMapByType) == 0
}

func (r *RedisServiceDiscovery) Init() error {
	ctx := context.TODO()

	if err := r.addServerToRedis(ctx); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
}

func (r *RedisServiceDiscovery) AfterInit() {
	// No implementation.
}

func (r *RedisServiceDiscovery) BeforeShutdown() {
	// No implementation.
}

func (r *RedisServiceDiscovery) Shutdown() error {
	// No implementation.
}

func (r *RedisServiceDiscovery) addServerToRedis(ctx context.Context) error {
	// TODO: remove hardcoded params
	redisKey := getServerRedisKey(r.server)
	status := r.redisClient.Set(ctx, redisKey, "not used", time.Hour*4)
	if err := status.Err(); err != nil {
		return fmt.Errorf("add server to redis: %w", err)
	}
	return nil
}

func getServerRedisKey(server *Server) string {
	return fmt.Sprintf("%s", server.AsJSONString())
}
