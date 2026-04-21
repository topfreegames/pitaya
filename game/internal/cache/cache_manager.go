package cache

import (
	"fmt"
	"time"
)

type CacheManager struct {
	redisCache    *RedisCache
	playerCache   *PlayerCache
	roomCache     *RoomCache
	rankingCache  *RankingCache
}

func NewCacheManager(config *RedisConfig) (*CacheManager, error) {
	redisCache := NewRedisCache(config)

	if err := redisCache.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheManager{
		redisCache:   redisCache,
		playerCache:  NewPlayerCache(redisCache),
		roomCache:    NewRoomCache(redisCache),
		rankingCache: NewRankingCache(redisCache),
	}, nil
}

func (cm *CacheManager) GetRedisCache() *RedisCache {
	return cm.redisCache
}

func (cm *CacheManager) GetPlayerCache() *PlayerCache {
	return cm.playerCache
}

func (cm *CacheManager) GetRoomCache() *RoomCache {
	return cm.roomCache
}

func (cm *CacheManager) GetRankingCache() *RankingCache {
	return cm.rankingCache
}

func (cm *CacheManager) Close() error {
	return cm.redisCache.Close()
}

func (cm *CacheManager) HealthCheck() error {
	return cm.redisCache.Ping()
}

func (cm *CacheManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	playerCount, _ := cm.redisCache.GetClient().DBSize(cm.redisCache.ctx).Result()
	stats["total_keys"] = playerCount

	playerInfoCount, _ := cm.redisCache.GetClient().Keys(cm.redisCache.ctx, "player:*:info").Result()
	stats["player_info_count"] = len(playerInfoCount)

	roomInfoCount, _ := cm.redisCache.GetClient().Keys(cm.redisCache.ctx, "room:*:info").Result()
	stats["room_info_count"] = len(roomInfoCount)

	rankingCount, _ := cm.redisCache.GetClient().Keys(cm.redisCache.ctx, "ranking:*").Result()
	stats["ranking_count"] = len(rankingCount)

	return stats
}

func (cm *CacheManager) ClearAll() error {
	patterns := []string{
		"player:*",
		"room:*",
		"ranking:*",
	}

	for _, pattern := range patterns {
		keys, err := cm.redisCache.GetClient().Keys(cm.redisCache.ctx, pattern).Result()
		if err != nil {
			continue
		}

		if len(keys) > 0 {
			cm.redisCache.Del(keys...)
		}
	}

	return nil
}

func (cm *CacheManager) WarmupPlayerCache(playerIDs []string) error {
	_, err := cm.playerCache.BatchGetPlayerInfo(playerIDs)
	return err
}

func (cm *CacheManager) WarmupRoomCache(roomIDs []string) error {
	_, err := cm.roomCache.BatchGetRoomInfo(roomIDs)
	return err
}

func (cm *CacheManager) GetCacheSize() (int64, error) {
	return cm.redisCache.GetClient().DBSize(cm.redisCache.ctx).Result()
}

func (cm *CacheManager) GetMemoryUsage() (string, error) {
	info, err := cm.redisCache.GetClient().Info(cm.redisCache.ctx, "memory").Result()
	if err != nil {
		return "", err
	}
	return info, nil
}

func (cm *CacheManager) SetDefaultTTL(ttl time.Duration) {
}

func (cm *CacheManager) GetDefaultTTL() time.Duration {
	return 1 * time.Hour
}
