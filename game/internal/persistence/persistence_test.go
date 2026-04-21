package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/topfreegames/pitaya/v3/game/internal/cache"
	"github.com/topfreegames/pitaya/v3/game/internal/database"
)

func TestAsyncPersistence_UpdatePlayerCoins(t *testing.T) {
	cacheConfig := &cache.RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := cache.NewCacheManager(cacheConfig)
	assert.NoError(t, err)
	defer cacheManager.Close()

	dbConfig := &database.MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dbManager, err := database.NewDatabaseManager(dbConfig)
	assert.NoError(t, err)
	defer dbManager.Close()

	ctx := context.Background()
	err = dbManager.InitializeSchema(ctx)
	assert.NoError(t, err)

	for _, playerID := range []string{"player1", "player2", "player3"} {
		dbManager.GetPlayerRepo().Delete(ctx, playerID)
	}

	dataService := NewDataService(cacheManager, dbManager)
	defer dataService.Shutdown()

	player := &database.Player{
		ID:        "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = dbManager.GetPlayerRepo().Create(ctx, player)
	assert.NoError(t, err)

	info, err := dataService.GetPlayerInfo("player1")
	assert.NoError(t, err)
	assert.Equal(t, player.ID, info.PlayerID)

	err = dataService.UpdatePlayerCoins("player1", 100)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	result, err := dbManager.GetPlayerRepo().GetByID(ctx, "player1")
	assert.NoError(t, err)
	assert.Equal(t, int64(600), result.Coins)
}

func TestDataService_BatchGet(t *testing.T) {
	cacheConfig := &cache.RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	cacheManager, err := cache.NewCacheManager(cacheConfig)
	assert.NoError(t, err)
	defer cacheManager.Close()

	dbConfig := &database.MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dbManager, err := database.NewDatabaseManager(dbConfig)
	assert.NoError(t, err)
	defer dbManager.Close()

	ctx := context.Background()
	err = dbManager.InitializeSchema(ctx)
	assert.NoError(t, err)

	for _, playerID := range []string{"player1", "player2", "player3"} {
		dbManager.GetPlayerRepo().Delete(ctx, playerID)
	}

	dataService := NewDataService(cacheManager, dbManager)
	defer dataService.Shutdown()

	players := []*database.Player{
		{
			ID:        "player1",
			Name:      "Player 1",
			Level:     10,
			Exp:       1000,
			Coins:     500,
			Gems:      100,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "player2",
			Name:      "Player 2",
			Level:     20,
			Exp:       2000,
			Coins:     1000,
			Gems:      200,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "player3",
			Name:      "Player 3",
			Level:     30,
			Exp:       3000,
			Coins:     1500,
			Gems:      300,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, player := range players {
		err = dbManager.GetPlayerRepo().Create(ctx, player)
		assert.NoError(t, err)
	}

	playerIDs := []string{"player1", "player2", "player3"}
	infos, err := dataService.BatchGetPlayerInfo(playerIDs)
	assert.NoError(t, err)
	assert.Len(t, infos, 3)
	assert.Contains(t, infos, "player1")
	assert.Contains(t, infos, "player2")
	assert.Contains(t, infos, "player3")
}
