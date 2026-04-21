package persistence

import (
	"github.com/topfreegames/pitaya/v3/game/internal/cache"
	"github.com/topfreegames/pitaya/v3/game/internal/database"
)

type DataService struct {
	cacheProtection *CacheProtection
	asyncPersistence *AsyncPersistence
}

func NewDataService(cache *cache.CacheManager, database *database.DatabaseManager) *DataService {
	return &DataService{
		cacheProtection: NewCacheProtection(cache, database),
		asyncPersistence: NewAsyncPersistence(database, 10),
	}
}

func (ds *DataService) GetPlayerInfo(playerID string) (*cache.PlayerInfo, error) {
	return ds.cacheProtection.GetPlayerInfo(playerID)
}

func (ds *DataService) UpdatePlayerCoins(playerID string, delta int64) error {
	err := ds.cacheProtection.GetPlayerCache().UpdatePlayerCoins(playerID, delta)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdatePlayerCoins(playerID, delta)
}

func (ds *DataService) UpdatePlayerExp(playerID string, delta int64) error {
	err := ds.cacheProtection.GetPlayerCache().UpdatePlayerExp(playerID, delta)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdatePlayerExp(playerID, delta)
}

func (ds *DataService) UpdatePlayerGems(playerID string, delta int64) error {
	err := ds.cacheProtection.GetPlayerCache().UpdatePlayerGems(playerID, delta)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdatePlayerGems(playerID, delta)
}

func (ds *DataService) UpdatePlayerLevel(playerID string, level int32) error {
	err := ds.cacheProtection.GetPlayerCache().UpdatePlayerLevel(playerID, level)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdatePlayerLevel(playerID, level)
}

func (ds *DataService) GetRoomInfo(roomID string) (*cache.RoomInfo, error) {
	return ds.cacheProtection.GetRoomInfo(roomID)
}

func (ds *DataService) UpdateRoomStatus(roomID, status string) error {
	err := ds.cacheProtection.GetRoomCache().UpdateRoomStatus(roomID, status)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdateRoomStatus(roomID, status)
}

func (ds *DataService) UpdateRoomScore(roomID string, score int64) error {
	err := ds.cacheProtection.GetRoomCache().UpdateRoomScore(roomID, score)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdateRoomScore(roomID, score)
}

func (ds *DataService) UpdateRoomWaves(roomID string, waves int32) error {
	err := ds.cacheProtection.GetRoomCache().UpdateRoomWaves(roomID, waves)
	if err != nil {
		return err
	}

	return ds.asyncPersistence.UpdateRoomWaves(roomID, waves)
}

func (ds *DataService) CreateGameRecord(playerID, roomID string, score int64, waves, enemies, duration int32) error {
	return ds.asyncPersistence.CreateGameRecord(playerID, roomID, score, waves, enemies, duration)
}

func (ds *DataService) BatchGetPlayerInfo(playerIDs []string) (map[string]*cache.PlayerInfo, error) {
	return ds.cacheProtection.BatchGetPlayerInfo(playerIDs)
}

func (ds *DataService) BatchGetRoomInfo(roomIDs []string) (map[string]*cache.RoomInfo, error) {
	return ds.cacheProtection.BatchGetRoomInfo(roomIDs)
}

func (ds *DataService) InvalidatePlayer(playerID string) error {
	return ds.cacheProtection.InvalidatePlayer(playerID)
}

func (ds *DataService) InvalidateRoom(roomID string) error {
	return ds.cacheProtection.InvalidateRoom(roomID)
}

func (ds *DataService) InvalidateAll() error {
	return ds.cacheProtection.InvalidateAll()
}

func (ds *DataService) GetStats() map[string]interface{} {
	cacheStats := ds.cacheProtection.GetStats()
	asyncStats := ds.asyncPersistence.GetStats()

	return map[string]interface{}{
		"cache":        cacheStats,
		"persistence": asyncStats,
	}
}

func (ds *DataService) GetCacheSize() (int64, error) {
	return ds.cacheProtection.GetCacheSize()
}

func (ds *DataService) GetMemoryUsage() (string, error) {
	return ds.cacheProtection.GetMemoryUsage()
}

func (ds *DataService) WarmupPlayerCache(playerIDs []string) error {
	return ds.cacheProtection.WarmupPlayerCache(playerIDs)
}

func (ds *DataService) WarmupRoomCache(roomIDs []string) error {
	return ds.cacheProtection.WarmupRoomCache(roomIDs)
}

func (ds *DataService) Shutdown() {
	ds.asyncPersistence.Shutdown()
}

func (ds *DataService) IsShutdown() bool {
	return ds.asyncPersistence.IsShutdown()
}

func (ds *DataService) GetSingleFlightStats() map[string]interface{} {
	return ds.cacheProtection.GetSingleFlightStats()
}
