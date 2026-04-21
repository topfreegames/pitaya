package database

import (
	"context"
	"database/sql"
	"fmt"
)

type DatabaseManager struct {
	mysqlDB              *MySQLDatabase
	playerRepo           *PlayerRepository
	roomRepo             *RoomRepository
	roomPlayerRepo       *RoomPlayerRepository
	gameRecordRepo       *GameRecordRepository
}

func NewDatabaseManager(config *MySQLConfig) (*DatabaseManager, error) {
	mysqlDB, err := NewMySQLDatabase(config)
	if err != nil {
		return nil, err
	}

	return &DatabaseManager{
		mysqlDB:        mysqlDB,
		playerRepo:     NewPlayerRepository(mysqlDB),
		roomRepo:       NewRoomRepository(mysqlDB),
		roomPlayerRepo: NewRoomPlayerRepository(mysqlDB),
		gameRecordRepo: NewGameRecordRepository(mysqlDB),
	}, nil
}

func (dm *DatabaseManager) GetMySQLDB() *MySQLDatabase {
	return dm.mysqlDB
}

func (dm *DatabaseManager) GetPlayerRepo() *PlayerRepository {
	return dm.playerRepo
}

func (dm *DatabaseManager) GetRoomRepo() *RoomRepository {
	return dm.roomRepo
}

func (dm *DatabaseManager) GetRoomPlayerRepo() *RoomPlayerRepository {
	return dm.roomPlayerRepo
}

func (dm *DatabaseManager) GetGameRecordRepo() *GameRecordRepository {
	return dm.gameRecordRepo
}

func (dm *DatabaseManager) Close() error {
	return dm.mysqlDB.Close()
}

func (dm *DatabaseManager) HealthCheck() error {
	return dm.mysqlDB.Ping()
}

func (dm *DatabaseManager) GetStats() map[string]interface{} {
	return dm.mysqlDB.GetStats()
}

func (dm *DatabaseManager) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return dm.mysqlDB.BeginTx()
}

func (dm *DatabaseManager) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return dm.mysqlDB.ExecContext(ctx, query, args...)
}

func (dm *DatabaseManager) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return dm.mysqlDB.QueryContext(ctx, query, args...)
}

func (dm *DatabaseManager) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return dm.mysqlDB.QueryRowContext(ctx, query, args...)
}

func (dm *DatabaseManager) InitializeSchema(ctx context.Context) error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(64) NOT NULL,
			level INT DEFAULT 1,
			exp BIGINT DEFAULT 0,
			coins BIGINT DEFAULT 0,
			gems BIGINT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_level (level),
			INDEX idx_coins (coins)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS rooms (
			id VARCHAR(64) PRIMARY KEY,
			room_type VARCHAR(32) NOT NULL,
			max_players INT NOT NULL,
			status VARCHAR(32) NOT NULL,
			score BIGINT DEFAULT 0,
			waves INT DEFAULT 0,
			difficulty INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP NULL,
			finished_at TIMESTAMP NULL,
			INDEX idx_status (status),
			INDEX idx_score (score)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS room_players (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			room_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			score BIGINT DEFAULT 0,
			waves_survived INT DEFAULT 0,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			left_at TIMESTAMP NULL,
			INDEX idx_room_id (room_id),
			INDEX idx_player_id (player_id),
			UNIQUE KEY uk_room_player (room_id, player_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS game_records (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			player_id VARCHAR(64) NOT NULL,
			room_id VARCHAR(64) NOT NULL,
			score BIGINT DEFAULT 0,
			waves_survived INT DEFAULT 0,
			enemies_defeated INT DEFAULT 0,
			duration_seconds INT DEFAULT 0,
			played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_player_id (player_id),
			INDEX idx_played_at (played_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, schema := range schemas {
		_, err := dm.mysqlDB.ExecContext(ctx, schema)
		if err != nil {
			return fmt.Errorf("failed to initialize schema: %w", err)
		}
	}

	return nil
}

func (dm *DatabaseManager) DropSchema(ctx context.Context) error {
	schemas := []string{
		"DROP TABLE IF EXISTS game_records",
		"DROP TABLE IF EXISTS room_players",
		"DROP TABLE IF EXISTS rooms",
		"DROP TABLE IF EXISTS players",
	}

	for _, schema := range schemas {
		_, err := dm.mysqlDB.ExecContext(ctx, schema)
		if err != nil {
			return fmt.Errorf("failed to drop schema: %w", err)
		}
	}

	return nil
}

func (dm *DatabaseManager) TruncateAll(ctx context.Context) error {
	tables := []string{
		"game_records",
		"room_players",
		"rooms",
		"players",
	}

	for _, table := range tables {
		_, err := dm.mysqlDB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table))
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}
