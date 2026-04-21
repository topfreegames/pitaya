package cache

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisCache_Ping(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	err := cache.Ping()
	assert.NoError(t, err)
}

func TestRedisCache_SetAndGet(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	key := "test_key"
	value := "test_value"

	err := cache.Set(key, value, 1*time.Minute)
	assert.NoError(t, err)

	result, err := cache.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestRedisCache_SetJSONAndGetJSON(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	key := "test_json_key"
	value := &TestStruct{
		Name:  "test",
		Value: 123,
	}

	err := cache.SetJSON(key, value, 1*time.Minute)
	assert.NoError(t, err)

	var result TestStruct
	err = cache.GetJSON(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value.Name, result.Name)
	assert.Equal(t, value.Value, result.Value)
}

func TestRedisCache_HashOperations(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	key := "test_hash"

	err := cache.HSet(key, "field1", "value1", "field2", "value2")
	assert.NoError(t, err)

	result, err := cache.HGet(key, "field1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", result)

	all, err := cache.HGetAll(key)
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	exists, err := cache.HExists(key, "field1")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = cache.HDel(key, "field1")
	assert.NoError(t, err)

	exists, err = cache.HExists(key, "field1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCache_SetOperations(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	key := "test_set"

	err := cache.SAdd(key, "member1", "member2", "member3")
	assert.NoError(t, err)

	members, err := cache.SMembers(key)
	assert.NoError(t, err)
	assert.Len(t, members, 3)

	exists, err := cache.SIsMember(key, "member1")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = cache.SRem(key, "member1")
	assert.NoError(t, err)

	exists, err = cache.SIsMember(key, "member1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCache_SortedSetOperations(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config)
	defer cache.Close()

	key := "test_zset"

	err := cache.ZAdd(key, redis.Z{Score: 100, Member: "player1"}, redis.Z{Score: 200, Member: "player2"})
	assert.NoError(t, err)

	members, err := cache.ZRange(key, 0, -1)
	assert.NoError(t, err)
	assert.Len(t, members, 2)

	membersWithScores, err := cache.ZRangeWithScores(key, 0, -1)
	assert.NoError(t, err)
	assert.Len(t, membersWithScores, 2)

	score, err := cache.ZScore(key, "player1")
	assert.NoError(t, err)
	assert.Equal(t, float64(100), score)

	count, err := cache.ZCard(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	err = cache.ZRem(key, "player1")
	assert.NoError(t, err)

	count, err = cache.ZCard(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestPlayerCache_SetAndGetPlayerInfo(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	playerCache := cacheManager.GetPlayerCache()

	info := &PlayerInfo{
		PlayerID:  "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = playerCache.SetPlayerInfo(info)
	assert.NoError(t, err)

	result, err := playerCache.GetPlayerInfo("player1")
	assert.NoError(t, err)
	assert.Equal(t, info.PlayerID, result.PlayerID)
	assert.Equal(t, info.Name, result.Name)
	assert.Equal(t, info.Level, result.Level)
}

func TestPlayerCache_UpdatePlayerCoins(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	playerCache := cacheManager.GetPlayerCache()

	info := &PlayerInfo{
		PlayerID:  "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = playerCache.SetPlayerInfo(info)
	assert.NoError(t, err)

	err = playerCache.UpdatePlayerCoins("player1", 100)
	assert.NoError(t, err)

	result, err := playerCache.GetPlayerInfo("player1")
	assert.NoError(t, err)
	assert.Equal(t, int64(600), result.Coins)
}

func TestRoomCache_SetAndGetRoomInfo(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	roomCache := cacheManager.GetRoomCache()

	info := &RoomInfo{
		RoomID:         "room1",
		RoomType:       "normal",
		MaxPlayers:     4,
		CurrentPlayers: 2,
		Status:         "waiting",
		CreatedAt:      time.Now(),
	}

	err = roomCache.SetRoomInfo(info)
	assert.NoError(t, err)

	result, err := roomCache.GetRoomInfo("room1")
	assert.NoError(t, err)
	assert.Equal(t, info.RoomID, result.RoomID)
	assert.Equal(t, info.RoomType, result.RoomType)
	assert.Equal(t, info.MaxPlayers, result.MaxPlayers)
}

func TestRoomCache_AddAndRemovePlayer(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	roomCache := cacheManager.GetRoomCache()

	err = roomCache.AddPlayerToRoom("room1", "player1")
	assert.NoError(t, err)

	err = roomCache.AddPlayerToRoom("room1", "player2")
	assert.NoError(t, err)

	players, err := roomCache.GetRoomPlayers("room1")
	assert.NoError(t, err)
	assert.Len(t, players, 2)

	inRoom, err := roomCache.IsPlayerInRoom("room1", "player1")
	assert.NoError(t, err)
	assert.True(t, inRoom)

	err = roomCache.RemovePlayerFromRoom("room1", "player1")
	assert.NoError(t, err)

	players, err = roomCache.GetRoomPlayers("room1")
	assert.NoError(t, err)
	assert.Len(t, players, 1)
}

func TestRankingCache_UpdateAndGetRanking(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	rankingCache := cacheManager.GetRankingCache()

	err = rankingCache.UpdatePlayerScore("player1", 1000)
	assert.NoError(t, err)

	err = rankingCache.UpdatePlayerScore("player2", 2000)
	assert.NoError(t, err)

	err = rankingCache.UpdatePlayerScore("player3", 1500)
	assert.NoError(t, err)

	ranking, err := rankingCache.GetPlayerScoreRanking(0, 10)
	assert.NoError(t, err)
	assert.Len(t, ranking, 3)
	assert.Equal(t, int32(1), ranking[0].Rank)
	assert.Equal(t, "player2", ranking[0].PlayerID)
	assert.Equal(t, int64(2000), ranking[0].Score)

	rank, err := rankingCache.GetPlayerScoreRank("player1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), rank)

	score, err := rankingCache.GetPlayerScore("player1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), score)
}

func TestCacheManager_HealthCheck(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	err = cacheManager.HealthCheck()
	assert.NoError(t, err)
}

func TestCacheManager_GetStats(t *testing.T) {
	config := &RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := NewCacheManager(config)
	assert.NoError(t, err)
	defer cacheManager.Close()

	stats := cacheManager.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_keys")
}
