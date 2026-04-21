package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMySQLDatabase_Ping(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	db, err := NewMySQLDatabase(config)
	assert.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	assert.NoError(t, err)
}

func TestDatabaseManager_InitializeSchema(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)
}

func TestPlayerRepository_CreateAndGet(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	playerRepo := dm.GetPlayerRepo()

	player := &Player{
		ID:        "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = playerRepo.Create(ctx, player)
	assert.NoError(t, err)

	result, err := playerRepo.GetByID(ctx, "player1")
	assert.NoError(t, err)
	assert.Equal(t, player.ID, result.ID)
	assert.Equal(t, player.Name, result.Name)
	assert.Equal(t, player.Level, result.Level)
}

func TestPlayerRepository_Update(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	playerRepo := dm.GetPlayerRepo()

	player := &Player{
		ID:        "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = playerRepo.Create(ctx, player)
	assert.NoError(t, err)

	player.Name = "Updated Player"
	player.Level = 20

	err = playerRepo.Update(ctx, player)
	assert.NoError(t, err)

	result, err := playerRepo.GetByID(ctx, "player1")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Player", result.Name)
	assert.Equal(t, int32(20), result.Level)
}

func TestPlayerRepository_UpdateCoins(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	playerRepo := dm.GetPlayerRepo()

	player := &Player{
		ID:        "player1",
		Name:      "Test Player",
		Level:     10,
		Exp:       1000,
		Coins:     500,
		Gems:      100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = playerRepo.Create(ctx, player)
	assert.NoError(t, err)

	err = playerRepo.UpdateCoins(ctx, "player1", 100)
	assert.NoError(t, err)

	result, err := playerRepo.GetByID(ctx, "player1")
	assert.NoError(t, err)
	assert.Equal(t, int64(600), result.Coins)
}

func TestRoomRepository_CreateAndGet(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	roomRepo := dm.GetRoomRepo()

	now := time.Now()
	room := &Room{
		ID:         "room1",
		RoomType:   "normal",
		MaxPlayers: 4,
		Status:     "waiting",
		Score:      0,
		Waves:      0,
		Difficulty: 1,
		CreatedAt:  now,
		StartedAt:  nil,
		FinishedAt: nil,
	}

	err = roomRepo.Create(ctx, room)
	assert.NoError(t, err)

	result, err := roomRepo.GetByID(ctx, "room1")
	assert.NoError(t, err)
	assert.Equal(t, room.ID, result.ID)
	assert.Equal(t, room.RoomType, result.RoomType)
	assert.Equal(t, room.MaxPlayers, result.MaxPlayers)
}

func TestRoomRepository_UpdateStatus(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	roomRepo := dm.GetRoomRepo()

	now := time.Now()
	room := &Room{
		ID:         "room1",
		RoomType:   "normal",
		MaxPlayers: 4,
		Status:     "waiting",
		Score:      0,
		Waves:      0,
		Difficulty: 1,
		CreatedAt:  now,
		StartedAt:  nil,
		FinishedAt: nil,
	}

	err = roomRepo.Create(ctx, room)
	assert.NoError(t, err)

	err = roomRepo.UpdateStatus(ctx, "room1", "playing")
	assert.NoError(t, err)

	result, err := roomRepo.GetByID(ctx, "room1")
	assert.NoError(t, err)
	assert.Equal(t, "playing", result.Status)
}

func TestGameRecordRepository_CreateAndGet(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	gameRecordRepo := dm.GetGameRecordRepo()

	record := &GameRecord{
		PlayerID:        "player1",
		RoomID:          "room1",
		Score:           1000,
		WavesSurvived:   5,
		EnemiesDefeated: 50,
		DurationSeconds: 300,
		PlayedAt:        time.Now(),
	}

	err = gameRecordRepo.Create(ctx, record)
	assert.NoError(t, err)
	assert.Greater(t, record.ID, int64(0))

	result, err := gameRecordRepo.GetByID(ctx, record.ID)
	assert.NoError(t, err)
	assert.Equal(t, record.PlayerID, result.PlayerID)
	assert.Equal(t, record.RoomID, result.RoomID)
	assert.Equal(t, record.Score, result.Score)
}

func TestRoomPlayerRepository_CreateAndGet(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	ctx := context.Background()
	err = dm.InitializeSchema(ctx)
	assert.NoError(t, err)

	err = dm.TruncateAll(ctx)
	assert.NoError(t, err)

	roomPlayerRepo := dm.GetRoomPlayerRepo()

	roomPlayer := &RoomPlayer{
		RoomID:        "room1",
		PlayerID:      "player1",
		Score:         100,
		WavesSurvived: 5,
		JoinedAt:      time.Now(),
		LeftAt:        nil,
	}

	err = roomPlayerRepo.Create(ctx, roomPlayer)
	assert.NoError(t, err)
	assert.Greater(t, roomPlayer.ID, int64(0))

	result, err := roomPlayerRepo.GetByID(ctx, roomPlayer.ID)
	assert.NoError(t, err)
	assert.Equal(t, roomPlayer.RoomID, result.RoomID)
	assert.Equal(t, roomPlayer.PlayerID, result.PlayerID)
}

func TestDatabaseManager_HealthCheck(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	err = dm.HealthCheck()
	assert.NoError(t, err)
}

func TestDatabaseManager_GetStats(t *testing.T) {
	config := &MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		User:     "gameuser",
		Password: "gamepassword",
		Database: "gamedb",
	}

	dm, err := NewDatabaseManager(config)
	assert.NoError(t, err)
	defer dm.Close()

	stats := dm.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "max_open_connections")
	assert.Contains(t, stats, "open_connections")
}
