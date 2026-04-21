package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RankingEntry struct {
	Rank      int32  `json:"rank"`
	PlayerID  string `json:"player_id"`
	PlayerName string `json:"player_name"`
	Score     int64  `json:"score"`
	Level     int32  `json:"level"`
}

type RankingCache struct {
	cache *RedisCache
}

func NewRankingCache(cache *RedisCache) *RankingCache {
	return &RankingCache{
		cache: cache,
	}
}

func (rc *RankingCache) GetPlayerScoreRankingKey() string {
	return "ranking:players:score"
}

func (rc *RankingCache) GetPlayerWavesRankingKey() string {
	return "ranking:players:waves"
}

func (rc *RankingCache) GetRoomScoreRankingKey() string {
	return "ranking:rooms:score"
}

func (rc *RankingCache) UpdatePlayerScore(playerID string, score int64) error {
	key := rc.GetPlayerScoreRankingKey()
	return rc.cache.ZAdd(key, redis.Z{
		Score:  float64(score),
		Member: playerID,
	})
}

func (rc *RankingCache) UpdatePlayerWaves(playerID string, waves int64) error {
	key := rc.GetPlayerWavesRankingKey()
	return rc.cache.ZAdd(key, redis.Z{
		Score:  float64(waves),
		Member: playerID,
	})
}

func (rc *RankingCache) UpdateRoomScore(roomID string, score int64) error {
	key := rc.GetRoomScoreRankingKey()
	return rc.cache.ZAdd(key, redis.Z{
		Score:  float64(score),
		Member: roomID,
	})
}

func (rc *RankingCache) GetPlayerScoreRanking(offset, limit int64) ([]*RankingEntry, error) {
	key := rc.GetPlayerScoreRankingKey()
	start := offset
	stop := offset + limit - 1

	members, err := rc.cache.ZRevRangeWithScores(key, start, stop)
	if err != nil {
		return nil, err
	}

	entries := make([]*RankingEntry, 0, len(members))
	for i, member := range members {
		entries = append(entries, &RankingEntry{
			Rank:     int32(offset + int64(i) + 1),
			PlayerID: member.Member.(string),
			Score:    int64(member.Score),
		})
	}

	return entries, nil
}

func (rc *RankingCache) GetPlayerWavesRanking(offset, limit int64) ([]*RankingEntry, error) {
	key := rc.GetPlayerWavesRankingKey()
	start := offset
	stop := offset + limit - 1

	members, err := rc.cache.ZRevRangeWithScores(key, start, stop)
	if err != nil {
		return nil, err
	}

	entries := make([]*RankingEntry, 0, len(members))
	for i, member := range members {
		entries = append(entries, &RankingEntry{
			Rank:     int32(offset + int64(i) + 1),
			PlayerID: member.Member.(string),
			Score:    int64(member.Score),
		})
	}

	return entries, nil
}

func (rc *RankingCache) GetRoomScoreRanking(offset, limit int64) ([]*RankingEntry, error) {
	key := rc.GetRoomScoreRankingKey()
	start := offset
	stop := offset + limit - 1

	members, err := rc.cache.ZRevRangeWithScores(key, start, stop)
	if err != nil {
		return nil, err
	}

	entries := make([]*RankingEntry, 0, len(members))
	for i, member := range members {
		entries = append(entries, &RankingEntry{
			Rank:     int32(offset + int64(i) + 1),
			PlayerID: member.Member.(string),
			Score:    int64(member.Score),
		})
	}

	return entries, nil
}

func (rc *RankingCache) GetPlayerScoreRank(playerID string) (int64, error) {
	key := rc.GetPlayerScoreRankingKey()
	rank, err := rc.cache.ZRevRank(key, playerID)
	if err != nil {
		return 0, err
	}
	return rank + 1, nil
}

func (rc *RankingCache) GetPlayerWavesRank(playerID string) (int64, error) {
	key := rc.GetPlayerWavesRankingKey()
	rank, err := rc.cache.ZRevRank(key, playerID)
	if err != nil {
		return 0, err
	}
	return rank + 1, nil
}

func (rc *RankingCache) GetRoomScoreRank(roomID string) (int64, error) {
	key := rc.GetRoomScoreRankingKey()
	rank, err := rc.cache.ZRevRank(key, roomID)
	if err != nil {
		return 0, err
	}
	return rank + 1, nil
}

func (rc *RankingCache) GetPlayerScore(playerID string) (int64, error) {
	key := rc.GetPlayerScoreRankingKey()
	score, err := rc.cache.ZScore(key, playerID)
	if err != nil {
		return 0, err
	}
	return int64(score), nil
}

func (rc *RankingCache) GetPlayerWaves(playerID string) (int64, error) {
	key := rc.GetPlayerWavesRankingKey()
	waves, err := rc.cache.ZScore(key, playerID)
	if err != nil {
		return 0, err
	}
	return int64(waves), nil
}

func (rc *RankingCache) GetRoomScore(roomID string) (int64, error) {
	key := rc.GetRoomScoreRankingKey()
	score, err := rc.cache.ZScore(key, roomID)
	if err != nil {
		return 0, err
	}
	return int64(score), nil
}

func (rc *RankingCache) GetPlayerScoreCount() (int64, error) {
	key := rc.GetPlayerScoreRankingKey()
	return rc.cache.ZCard(key)
}

func (rc *RankingCache) GetPlayerWavesCount() (int64, error) {
	key := rc.GetPlayerWavesRankingKey()
	return rc.cache.ZCard(key)
}

func (rc *RankingCache) GetRoomScoreCount() (int64, error) {
	key := rc.GetRoomScoreRankingKey()
	return rc.cache.ZCard(key)
}

func (rc *RankingCache) RemovePlayerFromRankings(playerID string) error {
	pipe := rc.cache.Pipeline()

	pipe.ZRem(context.Background(), rc.GetPlayerScoreRankingKey(), playerID)
	pipe.ZRem(context.Background(), rc.GetPlayerWavesRankingKey(), playerID)

	_, err := pipe.Exec(context.Background())
	return err
}

func (rc *RankingCache) RemoveRoomFromRankings(roomID string) error {
	return rc.cache.ZRem(rc.GetRoomScoreRankingKey(), roomID)
}

func (rc *RankingCache) GetTopPlayersByScore(limit int64) ([]*RankingEntry, error) {
	return rc.GetPlayerScoreRanking(0, limit)
}

func (rc *RankingCache) GetTopPlayersByWaves(limit int64) ([]*RankingEntry, error) {
	return rc.GetPlayerWavesRanking(0, limit)
}

func (rc *RankingCache) GetTopRoomsByScore(limit int64) ([]*RankingEntry, error) {
	return rc.GetRoomScoreRanking(0, limit)
}

func (rc *RankingCache) BatchUpdatePlayerScores(scores map[string]int64) error {
	key := rc.GetPlayerScoreRankingKey()
	pipe := rc.cache.Pipeline()

	for playerID, score := range scores {
		pipe.ZAdd(context.Background(), key, redis.Z{
			Score:  float64(score),
			Member: playerID,
		})
	}

	_, err := pipe.Exec(context.Background())
	return err
}

func (rc *RankingCache) BatchUpdatePlayerWaves(waves map[string]int64) error {
	key := rc.GetPlayerWavesRankingKey()
	pipe := rc.cache.Pipeline()

	for playerID, wave := range waves {
		pipe.ZAdd(context.Background(), key, redis.Z{
			Score:  float64(wave),
			Member: playerID,
		})
	}

	_, err := pipe.Exec(context.Background())
	return err
}
