package database

import (
	"context"
	"database/sql"
	"fmt"
)

type GameRecordRepository struct {
	db *MySQLDatabase
}

func NewGameRecordRepository(db *MySQLDatabase) *GameRecordRepository {
	return &GameRecordRepository{db: db}
}

func (grr *GameRecordRepository) Create(ctx context.Context, record *GameRecord) error {
	query := `
		INSERT INTO game_records (player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := grr.db.ExecContext(ctx, query,
		record.PlayerID,
		record.RoomID,
		record.Score,
		record.WavesSurvived,
		record.EnemiesDefeated,
		record.DurationSeconds,
		record.PlayedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	record.ID = id
	return nil
}

func (grr *GameRecordRepository) GetByID(ctx context.Context, id int64) (*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		WHERE id = ?
	`

	var record GameRecord
	err := grr.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID,
		&record.PlayerID,
		&record.RoomID,
		&record.Score,
		&record.WavesSurvived,
		&record.EnemiesDefeated,
		&record.DurationSeconds,
		&record.PlayedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game record not found: %d", id)
		}
		return nil, err
	}

	return &record, nil
}

func (grr *GameRecordRepository) GetByPlayerID(ctx context.Context, playerID string, limit, offset int) ([]*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		WHERE player_id = ?
		ORDER BY played_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := grr.db.QueryContext(ctx, query, playerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*GameRecord, 0)
	for rows.Next() {
		var record GameRecord
		err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.RoomID,
			&record.Score,
			&record.WavesSurvived,
			&record.EnemiesDefeated,
			&record.DurationSeconds,
			&record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (grr *GameRecordRepository) GetByRoomID(ctx context.Context, roomID string, limit, offset int) ([]*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		WHERE room_id = ?
		ORDER BY played_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := grr.db.QueryContext(ctx, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*GameRecord, 0)
	for rows.Next() {
		var record GameRecord
		err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.RoomID,
			&record.Score,
			&record.WavesSurvived,
			&record.EnemiesDefeated,
			&record.DurationSeconds,
			&record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (grr *GameRecordRepository) GetBestScore(ctx context.Context, playerID string) (int64, error) {
	query := `
		SELECT COALESCE(MAX(score), 0)
		FROM game_records
		WHERE player_id = ?
	`

	var score int64
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&score)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (grr *GameRecordRepository) GetBestWaves(ctx context.Context, playerID string) (int32, error) {
	query := `
		SELECT COALESCE(MAX(waves_survived), 0)
		FROM game_records
		WHERE player_id = ?
	`

	var waves int32
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&waves)
	if err != nil {
		return 0, err
	}

	return waves, nil
}

func (grr *GameRecordRepository) GetTotalGames(ctx context.Context, playerID string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM game_records
		WHERE player_id = ?
	`

	var count int64
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (grr *GameRecordRepository) GetAverageScore(ctx context.Context, playerID string) (float64, error) {
	query := `
		SELECT COALESCE(AVG(score), 0)
		FROM game_records
		WHERE player_id = ?
	`

	var avg float64
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&avg)
	if err != nil {
		return 0, err
	}

	return avg, nil
}

func (grr *GameRecordRepository) GetAverageWaves(ctx context.Context, playerID string) (float64, error) {
	query := `
		SELECT COALESCE(AVG(waves_survived), 0)
		FROM game_records
		WHERE player_id = ?
	`

	var avg float64
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&avg)
	if err != nil {
		return 0, err
	}

	return avg, nil
}

func (grr *GameRecordRepository) List(ctx context.Context, limit, offset int) ([]*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		ORDER BY played_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := grr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*GameRecord, 0)
	for rows.Next() {
		var record GameRecord
		err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.RoomID,
			&record.Score,
			&record.WavesSurvived,
			&record.EnemiesDefeated,
			&record.DurationSeconds,
			&record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (grr *GameRecordRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM game_records`

	var count int64
	err := grr.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (grr *GameRecordRepository) CountByPlayer(ctx context.Context, playerID string) (int64, error) {
	query := `SELECT COUNT(*) FROM game_records WHERE player_id = ?`

	var count int64
	err := grr.db.QueryRowContext(ctx, query, playerID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (grr *GameRecordRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM game_records WHERE id = ?`

	result, err := grr.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("game record not found: %d", id)
	}

	return nil
}

func (grr *GameRecordRepository) DeleteByPlayerID(ctx context.Context, playerID string) error {
	query := `DELETE FROM game_records WHERE player_id = ?`

	_, err := grr.db.ExecContext(ctx, query, playerID)
	return err
}

func (grr *GameRecordRepository) DeleteByRoomID(ctx context.Context, roomID string) error {
	query := `DELETE FROM game_records WHERE room_id = ?`

	_, err := grr.db.ExecContext(ctx, query, roomID)
	return err
}

func (grr *GameRecordRepository) GetTopPlayersByScore(ctx context.Context, limit int) ([]*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		ORDER BY score DESC
		LIMIT ?
	`

	rows, err := grr.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*GameRecord, 0)
	for rows.Next() {
		var record GameRecord
		err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.RoomID,
			&record.Score,
			&record.WavesSurvived,
			&record.EnemiesDefeated,
			&record.DurationSeconds,
			&record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (grr *GameRecordRepository) GetTopPlayersByWaves(ctx context.Context, limit int) ([]*GameRecord, error) {
	query := `
		SELECT id, player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at
		FROM game_records
		ORDER BY waves_survived DESC
		LIMIT ?
	`

	rows, err := grr.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*GameRecord, 0)
	for rows.Next() {
		var record GameRecord
		err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.RoomID,
			&record.Score,
			&record.WavesSurvived,
			&record.EnemiesDefeated,
			&record.DurationSeconds,
			&record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (grr *GameRecordRepository) BatchCreate(ctx context.Context, records []*GameRecord) error {
	if len(records) == 0 {
		return nil
	}

	query := `
		INSERT INTO game_records (player_id, room_id, score, waves_survived, enemies_defeated, duration_seconds, played_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := grr.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, record := range records {
		_, err := stmt.ExecContext(ctx,
			record.PlayerID,
			record.RoomID,
			record.Score,
			record.WavesSurvived,
			record.EnemiesDefeated,
			record.DurationSeconds,
			record.PlayedAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
