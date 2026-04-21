package database

import (
	"time"
)

type Player struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Level     int32     `db:"level"`
	Exp       int64     `db:"exp"`
	Coins     int64     `db:"coins"`
	Gems      int64     `db:"gems"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Room struct {
	ID         string     `db:"id"`
	RoomType   string     `db:"room_type"`
	MaxPlayers int32      `db:"max_players"`
	Status     string     `db:"status"`
	Score      int64      `db:"score"`
	Waves      int32      `db:"waves"`
	Difficulty int32      `db:"difficulty"`
	CreatedAt  time.Time  `db:"created_at"`
	StartedAt  *time.Time `db:"started_at"`
	FinishedAt *time.Time `db:"finished_at"`
}

type RoomPlayer struct {
	ID             int64     `db:"id"`
	RoomID         string    `db:"room_id"`
	PlayerID       string    `db:"player_id"`
	Score          int64     `db:"score"`
	WavesSurvived  int32     `db:"waves_survived"`
	JoinedAt       time.Time `db:"joined_at"`
	LeftAt         *time.Time `db:"left_at"`
}

type GameRecord struct {
	ID              int64     `db:"id"`
	PlayerID        string    `db:"player_id"`
	RoomID          string    `db:"room_id"`
	Score           int64     `db:"score"`
	WavesSurvived   int32     `db:"waves_survived"`
	EnemiesDefeated int32     `db:"enemies_defeated"`
	DurationSeconds int32     `db:"duration_seconds"`
	PlayedAt        time.Time `db:"played_at"`
}

type PlayerStats struct {
	TotalGames      int64 `db:"total_games"`
	TotalScore      int64 `db:"total_score"`
	TotalWaves      int64 `db:"total_waves"`
	TotalEnemies    int64 `db:"total_enemies"`
	BestScore       int64 `db:"best_score"`
	BestWaves       int32 `db:"best_waves"`
	AverageScore    float64 `db:"average_score"`
	AverageWaves    float64 `db:"average_waves"`
}

type RoomStats struct {
	TotalGames      int64 `db:"total_games"`
	TotalPlayers    int64 `db:"total_players"`
	TotalScore      int64 `db:"total_score"`
	TotalWaves      int64 `db:"total_waves"`
	BestScore       int64 `db:"best_score"`
	BestWaves       int32 `db:"best_waves"`
	AverageScore    float64 `db:"average_score"`
	AverageWaves    float64 `db:"average_waves"`
}
