package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RoomCache struct {
	cache *RedisCache
}

func NewRoomCache(cache *RedisCache) *RoomCache {
	return &RoomCache{
		cache: cache,
	}
}

func (rc *RoomCache) GetRoomInfoKey(roomID string) string {
	return fmt.Sprintf("room:%s:info", roomID)
}

func (rc *RoomCache) GetRoomPlayersKey(roomID string) string {
	return fmt.Sprintf("room:%s:players", roomID)
}

func (rc *RoomCache) GetRoomStateKey(roomID string) string {
	return fmt.Sprintf("room:%s:state", roomID)
}

func (rc *RoomCache) SetRoomInfo(info *RoomInfo) error {
	key := rc.GetRoomInfoKey(info.RoomID)
	return rc.cache.SetJSON(key, info, RoomInfoTTL)
}

func (rc *RoomCache) GetRoomInfo(roomID string) (*RoomInfo, error) {
	key := rc.GetRoomInfoKey(roomID)
	var info RoomInfo
	err := rc.cache.GetJSON(key, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (rc *RoomCache) AddPlayerToRoom(roomID, playerID string) error {
	key := rc.GetRoomPlayersKey(roomID)
	return rc.cache.SAdd(key, playerID)
}

func (rc *RoomCache) RemovePlayerFromRoom(roomID, playerID string) error {
	key := rc.GetRoomPlayersKey(roomID)
	return rc.cache.SRem(key, playerID)
}

func (rc *RoomCache) GetRoomPlayers(roomID string) ([]string, error) {
	key := rc.GetRoomPlayersKey(roomID)
	return rc.cache.SMembers(key)
}

func (rc *RoomCache) IsPlayerInRoom(roomID, playerID string) (bool, error) {
	key := rc.GetRoomPlayersKey(roomID)
	return rc.cache.SIsMember(key, playerID)
}

func (rc *RoomCache) SetRoomState(state *RoomState) error {
	key := rc.GetRoomStateKey(state.RoomID)
	return rc.cache.SetJSON(key, state, RoomStateTTL)
}

func (rc *RoomCache) GetRoomState(roomID string) (*RoomState, error) {
	key := rc.GetRoomStateKey(roomID)
	var state RoomState
	err := rc.cache.GetJSON(key, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (rc *RoomCache) DeleteRoom(roomID string) error {
	keys := []string{
		rc.GetRoomInfoKey(roomID),
		rc.GetRoomPlayersKey(roomID),
		rc.GetRoomStateKey(roomID),
	}
	return rc.cache.Del(keys...)
}

func (rc *RoomCache) UpdateRoomStatus(roomID, status string) error {
	info, err := rc.GetRoomInfo(roomID)
	if err != nil {
		return err
	}

	info.Status = status
	if status == "playing" {
		info.StartedAt = time.Now()
	} else if status == "finished" {
		info.FinishedAt = time.Now()
	}

	return rc.SetRoomInfo(info)
}

func (rc *RoomCache) UpdateRoomScore(roomID string, score int64) error {
	state, err := rc.GetRoomState(roomID)
	if err != nil {
		return err
	}

	state.Score = int32(score)
	state.LastUpdate = time.Now()

	return rc.SetRoomState(state)
}

func (rc *RoomCache) UpdateRoomWaves(roomID string, waves int32) error {
	state, err := rc.GetRoomState(roomID)
	if err != nil {
		return err
	}

	state.CurrentWave = waves
	state.LastUpdate = time.Now()

	return rc.SetRoomState(state)
}

func (rc *RoomCache) BatchGetRoomInfo(roomIDs []string) (map[string]*RoomInfo, error) {
	pipe := rc.cache.Pipeline()

	commands := make(map[string]*redis.StringCmd)
	for _, roomID := range roomIDs {
		key := rc.GetRoomInfoKey(roomID)
		commands[roomID] = pipe.Get(context.Background(), key)
	}

	_, err := pipe.Exec(context.Background())
	if err != nil && err != redis.Nil {
		return nil, err
	}

	infos := make(map[string]*RoomInfo)
	for roomID, cmd := range commands {
		data, err := cmd.Result()
		if err != nil {
			continue
		}

		var info RoomInfo
		if err := json.Unmarshal([]byte(data), &info); err == nil {
			infos[roomID] = &info
		}
	}

	return infos, nil
}

func (rc *RoomCache) GetRoomCount() (int64, error) {
	pattern := "room:*:info"
	keys, err := rc.cache.GetClient().Keys(context.Background(), pattern).Result()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}
