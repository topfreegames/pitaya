package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(config *RedisConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	return &RedisCache{
		client: client,
		ctx:    context.Background(),
	}
}

func (rc *RedisCache) Ping() error {
	return rc.client.Ping(rc.ctx).Err()
}

func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

func (rc *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	return rc.client.Set(rc.ctx, key, value, expiration).Err()
}

func (rc *RedisCache) Get(key string) (string, error) {
	return rc.client.Get(rc.ctx, key).Result()
}

func (rc *RedisCache) Del(keys ...string) error {
	return rc.client.Del(rc.ctx, keys...).Err()
}

func (rc *RedisCache) Exists(keys ...string) (int64, error) {
	return rc.client.Exists(rc.ctx, keys...).Result()
}

func (rc *RedisCache) Expire(key string, expiration time.Duration) error {
	return rc.client.Expire(rc.ctx, key, expiration).Err()
}

func (rc *RedisCache) TTL(key string) (time.Duration, error) {
	return rc.client.TTL(rc.ctx, key).Result()
}

func (rc *RedisCache) HSet(key string, values ...interface{}) error {
	return rc.client.HSet(rc.ctx, key, values...).Err()
}

func (rc *RedisCache) HGet(key, field string) (string, error) {
	return rc.client.HGet(rc.ctx, key, field).Result()
}

func (rc *RedisCache) HGetAll(key string) (map[string]string, error) {
	return rc.client.HGetAll(rc.ctx, key).Result()
}

func (rc *RedisCache) HDel(key string, fields ...string) error {
	return rc.client.HDel(rc.ctx, key, fields...).Err()
}

func (rc *RedisCache) HExists(key, field string) (bool, error) {
	return rc.client.HExists(rc.ctx, key, field).Result()
}

func (rc *RedisCache) SAdd(key string, members ...interface{}) error {
	return rc.client.SAdd(rc.ctx, key, members...).Err()
}

func (rc *RedisCache) SMembers(key string) ([]string, error) {
	return rc.client.SMembers(rc.ctx, key).Result()
}

func (rc *RedisCache) SRem(key string, members ...interface{}) error {
	return rc.client.SRem(rc.ctx, key, members...).Err()
}

func (rc *RedisCache) SIsMember(key string, member interface{}) (bool, error) {
	return rc.client.SIsMember(rc.ctx, key, member).Result()
}

func (rc *RedisCache) ZAdd(key string, members ...redis.Z) error {
	zs := make([]redis.Z, len(members))
	for i, m := range members {
		zs[i] = m
	}
	return rc.client.ZAdd(rc.ctx, key, zs...).Err()
}

func (rc *RedisCache) ZRange(key string, start, stop int64) ([]string, error) {
	return rc.client.ZRange(rc.ctx, key, start, stop).Result()
}

func (rc *RedisCache) ZRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return rc.client.ZRangeWithScores(rc.ctx, key, start, stop).Result()
}

func (rc *RedisCache) ZRevRange(key string, start, stop int64) ([]string, error) {
	return rc.client.ZRevRange(rc.ctx, key, start, stop).Result()
}

func (rc *RedisCache) ZRevRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return rc.client.ZRevRangeWithScores(rc.ctx, key, start, stop).Result()
}

func (rc *RedisCache) ZScore(key, member string) (float64, error) {
	return rc.client.ZScore(rc.ctx, key, member).Result()
}

func (rc *RedisCache) ZRem(key string, members ...interface{}) error {
	return rc.client.ZRem(rc.ctx, key, members...).Err()
}

func (rc *RedisCache) ZCard(key string) (int64, error) {
	return rc.client.ZCard(rc.ctx, key).Result()
}

func (rc *RedisCache) ZCount(key, min, max string) (int64, error) {
	return rc.client.ZCount(rc.ctx, key, min, max).Result()
}

func (rc *RedisCache) ZRevRank(key, member string) (int64, error) {
	return rc.client.ZRevRank(rc.ctx, key, member).Result()
}

func (rc *RedisCache) SetJSON(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return rc.Set(key, data, expiration)
}

func (rc *RedisCache) GetJSON(key string, dest interface{}) error {
	data, err := rc.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (rc *RedisCache) HSetJSON(key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return rc.HSet(key, field, data)
}

func (rc *RedisCache) HGetJSON(key, field string, dest interface{}) error {
	data, err := rc.HGet(key, field)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (rc *RedisCache) Pipeline() redis.Pipeliner {
	return rc.client.Pipeline()
}

func (rc *RedisCache) TxPipeline() redis.Pipeliner {
	return rc.client.TxPipeline()
}

func (rc *RedisCache) GetClient() *redis.Client {
	return rc.client
}
