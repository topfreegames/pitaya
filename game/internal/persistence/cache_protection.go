package persistence

import (
	"context"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/cache"
	"github.com/topfreegames/pitaya/v3/game/internal/database"
)

type SingleFlight struct {
	mu     sync.Mutex
	groups map[string]*singleFlightGroup
}

type singleFlightGroup struct {
	mu    sync.Mutex
	calls map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
	dup bool
}

func NewSingleFlight() *SingleFlight {
	return &SingleFlight{
		groups: make(map[string]*singleFlightGroup),
	}
}

func (sf *SingleFlight) getGroup(key string) *singleFlightGroup {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if _, exists := sf.groups[key]; !exists {
		sf.groups[key] = &singleFlightGroup{
			calls: make(map[string]*call),
		}
	}

	return sf.groups[key]
}

func (sf *SingleFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	group := sf.getGroup(key)

	group.mu.Lock()
	if c, ok := group.calls[key]; ok {
		group.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	group.calls[key] = c
	group.mu.Unlock()

	val, err := fn()

	group.mu.Lock()
	delete(group.calls, key)
	group.mu.Unlock()

	c.val = val
	c.err = err
	c.wg.Done()

	return val, err
}

type CacheProtection struct {
	singleFlight *SingleFlight
	cache        *cache.CacheManager
	database     *database.DatabaseManager
	mu           sync.RWMutex
}

func NewCacheProtection(cache *cache.CacheManager, database *database.DatabaseManager) *CacheProtection {
	return &CacheProtection{
		singleFlight: NewSingleFlight(),
		cache:        cache,
		database:     database,
	}
}

func (cp *CacheProtection) GetPlayerInfo(playerID string) (*cache.PlayerInfo, error) {
	key := cp.cache.GetPlayerCache().GetPlayerInfoKey(playerID)

	val, err := cp.singleFlight.Do(key, func() (interface{}, error) {
		info, err := cp.cache.GetPlayerCache().GetPlayerInfo(playerID)
		if err == nil && info != nil {
			return info, nil
		}

		if err != nil {
			return nil, err
		}

		player, err := cp.database.GetPlayerRepo().GetByID(context.Background(), playerID)
		if err != nil {
			return nil, err
		}

		cacheInfo := &cache.PlayerInfo{
			PlayerID:  player.ID,
			Name:      player.Name,
			Level:     player.Level,
			Exp:       player.Exp,
			Coins:     player.Coins,
			Gems:      player.Gems,
			CreatedAt: player.CreatedAt,
			UpdatedAt: player.UpdatedAt,
		}

		cp.cache.GetPlayerCache().SetPlayerInfo(cacheInfo)

		return cacheInfo, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*cache.PlayerInfo), nil
}

func (cp *CacheProtection) GetRoomInfo(roomID string) (*cache.RoomInfo, error) {
	key := cp.cache.GetRoomCache().GetRoomInfoKey(roomID)

	val, err := cp.singleFlight.Do(key, func() (interface{}, error) {
		info, err := cp.cache.GetRoomCache().GetRoomInfo(roomID)
		if err == nil && info != nil {
			return info, nil
		}

		if err != nil {
			return nil, err
		}

		room, err := cp.database.GetRoomRepo().GetByID(context.Background(), roomID)
		if err != nil {
			return nil, err
		}

		var startedAt, finishedAt time.Time
		if room.StartedAt != nil {
			startedAt = *room.StartedAt
		}
		if room.FinishedAt != nil {
			finishedAt = *room.FinishedAt
		}

		cacheInfo := &cache.RoomInfo{
			RoomID:         room.ID,
			RoomType:       room.RoomType,
			MaxPlayers:     room.MaxPlayers,
			CurrentPlayers: 0,
			Status:         room.Status,
			CreatedAt:      room.CreatedAt,
			StartedAt:      startedAt,
			FinishedAt:     finishedAt,
		}

		cp.cache.GetRoomCache().SetRoomInfo(cacheInfo)

		return cacheInfo, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*cache.RoomInfo), nil
}

func (cp *CacheProtection) BatchGetPlayerInfo(playerIDs []string) (map[string]*cache.PlayerInfo, error) {
	if len(playerIDs) == 0 {
		return make(map[string]*cache.PlayerInfo), nil
	}

	infos := make(map[string]*cache.PlayerInfo)
	missingIDs := make([]string, 0)

	for _, playerID := range playerIDs {
		info, err := cp.cache.GetPlayerCache().GetPlayerInfo(playerID)
		if err == nil && info != nil {
			infos[playerID] = info
		} else {
			missingIDs = append(missingIDs, playerID)
		}
	}

	if len(missingIDs) > 0 {
		players, err := cp.database.GetPlayerRepo().BatchGet(context.Background(), missingIDs)
		if err != nil {
			return nil, err
		}

		for playerID, player := range players {
			cacheInfo := &cache.PlayerInfo{
				PlayerID:  player.ID,
				Name:      player.Name,
				Level:     player.Level,
				Exp:       player.Exp,
				Coins:     player.Coins,
				Gems:      player.Gems,
				CreatedAt:  player.CreatedAt,
				UpdatedAt: player.UpdatedAt,
			}

			cp.cache.GetPlayerCache().SetPlayerInfo(cacheInfo)
			infos[playerID] = cacheInfo
		}
	}

	return infos, nil
}

func (cp *CacheProtection) BatchGetRoomInfo(roomIDs []string) (map[string]*cache.RoomInfo, error) {
	if len(roomIDs) == 0 {
		return make(map[string]*cache.RoomInfo), nil
	}

	infos := make(map[string]*cache.RoomInfo)
	missingIDs := make([]string, 0)

	for _, roomID := range roomIDs {
		info, err := cp.cache.GetRoomCache().GetRoomInfo(roomID)
		if err == nil && info != nil {
			infos[roomID] = info
		} else {
			missingIDs = append(missingIDs, roomID)
		}
	}

	if len(missingIDs) > 0 {
		rooms, err := cp.database.GetRoomRepo().BatchGet(context.Background(), missingIDs)
		if err != nil {
			return nil, err
		}

		for roomID, room := range rooms {
			var startedAt, finishedAt time.Time
			if room.StartedAt != nil {
				startedAt = *room.StartedAt
			}
			if room.FinishedAt != nil {
				finishedAt = *room.FinishedAt
			}

			cacheInfo := &cache.RoomInfo{
				RoomID:         room.ID,
				RoomType:       room.RoomType,
				MaxPlayers:     room.MaxPlayers,
				CurrentPlayers: 0,
				Status:         room.Status,
				CreatedAt:      room.CreatedAt,
				StartedAt:      startedAt,
				FinishedAt:     finishedAt,
			}

			cp.cache.GetRoomCache().SetRoomInfo(cacheInfo)
			infos[roomID] = cacheInfo
		}
	}

	return infos, nil
}

func (cp *CacheProtection) InvalidatePlayer(playerID string) error {
	return cp.cache.GetPlayerCache().DeletePlayer(playerID)
}

func (cp *CacheProtection) InvalidateRoom(roomID string) error {
	return cp.cache.GetRoomCache().DeleteRoom(roomID)
}

func (cp *CacheProtection) InvalidateAll() error {
	return cp.cache.ClearAll()
}

func (cp *CacheProtection) GetStats() map[string]interface{} {
	return cp.cache.GetStats()
}

func (cp *CacheProtection) GetCacheSize() (int64, error) {
	return cp.cache.GetCacheSize()
}

func (cp *CacheProtection) GetMemoryUsage() (string, error) {
	return cp.cache.GetMemoryUsage()
}

func (cp *CacheProtection) WarmupPlayerCache(playerIDs []string) error {
	_, err := cp.BatchGetPlayerInfo(playerIDs)
	return err
}

func (cp *CacheProtection) WarmupRoomCache(roomIDs []string) error {
	_, err := cp.BatchGetRoomInfo(roomIDs)
	return err
}

func (cp *CacheProtection) GetSingleFlightStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := make(map[string]interface{})
	for key, group := range cp.singleFlight.groups {
		group.mu.Lock()
		stats[key] = len(group.calls)
		group.mu.Unlock()
	}

	return stats
}

func (cp *CacheProtection) GetPlayerCache() *cache.PlayerCache {
	return cp.cache.GetPlayerCache()
}

func (cp *CacheProtection) GetRoomCache() *cache.RoomCache {
	return cp.cache.GetRoomCache()
}
