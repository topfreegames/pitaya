package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PlayerRepository struct {
	db *MySQLDatabase
}

func NewPlayerRepository(db *MySQLDatabase) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (pr *PlayerRepository) Create(ctx context.Context, player *Player) error {
	query := `
		INSERT INTO players (id, name, level, exp, coins, gems, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := pr.db.ExecContext(ctx, query,
		player.ID,
		player.Name,
		player.Level,
		player.Exp,
		player.Coins,
		player.Gems,
		player.CreatedAt,
		player.UpdatedAt,
	)

	return err
}

func (pr *PlayerRepository) GetByID(ctx context.Context, playerID string) (*Player, error) {
	query := `
		SELECT id, name, level, exp, coins, gems, created_at, updated_at
		FROM players
		WHERE id = ?
	`

	var player Player
	err := pr.db.QueryRowContext(ctx, query, playerID).Scan(
		&player.ID,
		&player.Name,
		&player.Level,
		&player.Exp,
		&player.Coins,
		&player.Gems,
		&player.CreatedAt,
		&player.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", playerID)
		}
		return nil, err
	}

	return &player, nil
}

func (pr *PlayerRepository) Update(ctx context.Context, player *Player) error {
	query := `
		UPDATE players
		SET name = ?, level = ?, exp = ?, coins = ?, gems = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query,
		player.Name,
		player.Level,
		player.Exp,
		player.Coins,
		player.Gems,
		time.Now(),
		player.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", player.ID)
	}

	return nil
}

func (pr *PlayerRepository) UpdateCoins(ctx context.Context, playerID string, delta int64) error {
	query := `
		UPDATE players
		SET coins = coins + ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query, delta, time.Now(), playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	return nil
}

func (pr *PlayerRepository) UpdateExp(ctx context.Context, playerID string, delta int64) error {
	query := `
		UPDATE players
		SET exp = exp + ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query, delta, time.Now(), playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	return nil
}

func (pr *PlayerRepository) UpdateGems(ctx context.Context, playerID string, delta int64) error {
	query := `
		UPDATE players
		SET gems = gems + ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query, delta, time.Now(), playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	return nil
}

func (pr *PlayerRepository) UpdateLevel(ctx context.Context, playerID string, level int32) error {
	query := `
		UPDATE players
		SET level = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query, level, time.Now(), playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	return nil
}

func (pr *PlayerRepository) Delete(ctx context.Context, playerID string) error {
	query := `DELETE FROM players WHERE id = ?`

	result, err := pr.db.ExecContext(ctx, query, playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	return nil
}

func (pr *PlayerRepository) List(ctx context.Context, limit, offset int) ([]*Player, error) {
	query := `
		SELECT id, name, level, exp, coins, gems, created_at, updated_at
		FROM players
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := pr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]*Player, 0)
	for rows.Next() {
		var player Player
		err := rows.Scan(
			&player.ID,
			&player.Name,
			&player.Level,
			&player.Exp,
			&player.Coins,
			&player.Gems,
			&player.CreatedAt,
			&player.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		players = append(players, &player)
	}

	return players, nil
}

func (pr *PlayerRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM players`

	var count int64
	err := pr.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (pr *PlayerRepository) GetStats(ctx context.Context, playerID string) (*PlayerStats, error) {
	query := `
		SELECT
			COUNT(*) as total_games,
			COALESCE(SUM(score), 0) as total_score,
			COALESCE(SUM(waves_survived), 0) as total_waves,
			COALESCE(SUM(enemies_defeated), 0) as total_enemies,
			COALESCE(MAX(score), 0) as best_score,
			COALESCE(MAX(waves_survived), 0) as best_waves,
			COALESCE(AVG(score), 0) as average_score,
			COALESCE(AVG(waves_survived), 0) as average_waves
		FROM game_records
		WHERE player_id = ?
	`

	var stats PlayerStats
	err := pr.db.QueryRowContext(ctx, query, playerID).Scan(
		&stats.TotalGames,
		&stats.TotalScore,
		&stats.TotalWaves,
		&stats.TotalEnemies,
		&stats.BestScore,
		&stats.BestWaves,
		&stats.AverageScore,
		&stats.AverageWaves,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (pr *PlayerRepository) BatchGet(ctx context.Context, playerIDs []string) (map[string]*Player, error) {
	if len(playerIDs) == 0 {
		return make(map[string]*Player), nil
	}

	query := `
		SELECT id, name, level, exp, coins, gems, created_at, updated_at
		FROM players
		WHERE id IN (` + placeholders(len(playerIDs)) + `)
	`

	args := make([]interface{}, len(playerIDs))
	for i, id := range playerIDs {
		args[i] = id
	}

	rows, err := pr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make(map[string]*Player)
	for rows.Next() {
		var player Player
		err := rows.Scan(
			&player.ID,
			&player.Name,
			&player.Level,
			&player.Exp,
			&player.Coins,
			&player.Gems,
			&player.CreatedAt,
			&player.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		players[player.ID] = &player
	}

	return players, nil
}

func (pr *PlayerRepository) Exists(ctx context.Context, playerID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM players WHERE id = ?)`

	var exists bool
	err := pr.db.QueryRowContext(ctx, query, playerID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}

	result := "?"
	for i := 1; i < n; i++ {
		result += ",?"
	}
	return result
}
