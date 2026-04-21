package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	PlayerInfoTTL     = 1 * time.Hour
	PlayerStateTTL    = 5 * time.Minute
	PlayerCombatTTL   = 10 * time.Minute
	RoomInfoTTL       = 2 * time.Hour
	RoomPlayersTTL    = 2 * time.Hour
	RoomStateTTL      = 2 * time.Hour
	RankingTTL        = 1 * time.Hour
)

type PlayerInfo struct {
	PlayerID   string    `json:"player_id"`
	Name       string    `json:"name"`
	Level      int32     `json:"level"`
	Exp        int64     `json:"exp"`
	Coins      int64     `json:"coins"`
	Gems       int64     `json:"gems"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PlayerState struct {
	PlayerID    string    `json:"player_id"`
	RoomID      string    `json:"room_id"`
	Status      string    `json:"status"`
	Health      int32     `json:"health"`
	PositionX   float32   `json:"position_x"`
	PositionY   float32   `json:"position_y"`
	PositionZ   float32   `json:"position_z"`
	LastAction  time.Time `json:"last_action"`
}

type PlayerCombat struct {
	PlayerID      string    `json:"player_id"`
	CurrentHP     int32     `json:"current_hp"`
	MaxHP         int32     `json:"max_hp"`
	AttackPower   int32     `json:"attack_power"`
	DefensePower  int32     `json:"defense_power"`
	Speed         float64   `json:"speed"`
	Skills        string    `json:"skills"`
	LastUpdated   time.Time `json:"last_updated"`
}

type RoomInfo struct {
	RoomID         string    `json:"room_id"`
	RoomType       string    `json:"room_type"`
	MaxPlayers     int32     `json:"max_players"`
	CurrentPlayers int32     `json:"current_players"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
}

type RoomState struct {
	RoomID           string    `json:"room_id"`
	CurrentWave      int32     `json:"current_wave"`
	EnemiesRemaining int32     `json:"enemies_remaining"`
	Score            int32     `json:"score"`
	GameTime         int64     `json:"game_time"`
	Difficulty       int32     `json:"difficulty"`
	LastUpdate       time.Time `json:"last_update"`
}

type PlayerCache struct {
	cache *RedisCache
}

func NewPlayerCache(cache *RedisCache) *PlayerCache {
	return &PlayerCache{
		cache: cache,
	}
}

func (pc *PlayerCache) GetPlayerInfoKey(playerID string) string {
	return fmt.Sprintf("player:%s:info", playerID)
}

func (pc *PlayerCache) GetPlayerStateKey(playerID string) string {
	return fmt.Sprintf("player:%s:state", playerID)
}

func (pc *PlayerCache) GetPlayerCombatKey(playerID string) string {
	return fmt.Sprintf("player:%s:combat", playerID)
}

func (pc *PlayerCache) SetPlayerInfo(info *PlayerInfo) error {
	key := pc.GetPlayerInfoKey(info.PlayerID)
	return pc.cache.SetJSON(key, info, PlayerInfoTTL)
}

func (pc *PlayerCache) GetPlayerInfo(playerID string) (*PlayerInfo, error) {
	key := pc.GetPlayerInfoKey(playerID)
	var info PlayerInfo
	err := pc.cache.GetJSON(key, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (pc *PlayerCache) SetPlayerState(state *PlayerState) error {
	key := pc.GetPlayerStateKey(state.PlayerID)
	return pc.cache.SetJSON(key, state, PlayerStateTTL)
}

func (pc *PlayerCache) GetPlayerState(playerID string) (*PlayerState, error) {
	key := pc.GetPlayerStateKey(playerID)
	var state PlayerState
	err := pc.cache.GetJSON(key, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (pc *PlayerCache) SetPlayerCombat(combat *PlayerCombat) error {
	key := pc.GetPlayerCombatKey(combat.PlayerID)
	return pc.cache.SetJSON(key, combat, PlayerCombatTTL)
}

func (pc *PlayerCache) GetPlayerCombat(playerID string) (*PlayerCombat, error) {
	key := pc.GetPlayerCombatKey(playerID)
	var combat PlayerCombat
	err := pc.cache.GetJSON(key, &combat)
	if err != nil {
		return nil, err
	}
	return &combat, nil
}

func (pc *PlayerCache) DeletePlayer(playerID string) error {
	keys := []string{
		pc.GetPlayerInfoKey(playerID),
		pc.GetPlayerStateKey(playerID),
		pc.GetPlayerCombatKey(playerID),
	}
	return pc.cache.Del(keys...)
}

func (pc *PlayerCache) UpdatePlayerCoins(playerID string, delta int64) error {
	info, err := pc.GetPlayerInfo(playerID)
	if err != nil {
		return err
	}

	info.Coins += delta
	info.UpdatedAt = time.Now()

	return pc.SetPlayerInfo(info)
}

func (pc *PlayerCache) UpdatePlayerExp(playerID string, delta int64) error {
	info, err := pc.GetPlayerInfo(playerID)
	if err != nil {
		return err
	}

	info.Exp += delta
	info.UpdatedAt = time.Now()

	return pc.SetPlayerInfo(info)
}

func (pc *PlayerCache) UpdatePlayerGems(playerID string, delta int64) error {
	info, err := pc.GetPlayerInfo(playerID)
	if err != nil {
		return err
	}

	info.Gems += delta
	info.UpdatedAt = time.Now()

	return pc.SetPlayerInfo(info)
}

func (pc *PlayerCache) UpdatePlayerLevel(playerID string, level int32) error {
	info, err := pc.GetPlayerInfo(playerID)
	if err != nil {
		return err
	}

	info.Level = level
	info.UpdatedAt = time.Now()

	return pc.SetPlayerInfo(info)
}

func (pc *PlayerCache) BatchGetPlayerInfo(playerIDs []string) (map[string]*PlayerInfo, error) {
	pipe := pc.cache.Pipeline()

	commands := make(map[string]*redis.StringCmd)
	for _, playerID := range playerIDs {
		key := pc.GetPlayerInfoKey(playerID)
		commands[playerID] = pipe.Get(context.Background(), key)
	}

	_, err := pipe.Exec(context.Background())
	if err != nil && err != redis.Nil {
		return nil, err
	}

	infos := make(map[string]*PlayerInfo)
	for playerID, cmd := range commands {
		data, err := cmd.Result()
		if err != nil {
			continue
		}

		var info PlayerInfo
		if err := json.Unmarshal([]byte(data), &info); err == nil {
			infos[playerID] = &info
		}
	}

	return infos, nil
}
