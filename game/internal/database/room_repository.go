package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type RoomRepository struct {
	db *MySQLDatabase
}

func NewRoomRepository(db *MySQLDatabase) *RoomRepository {
	return &RoomRepository{db: db}
}

func (rr *RoomRepository) Create(ctx context.Context, room *Room) error {
	query := `
		INSERT INTO rooms (id, room_type, max_players, status, score, waves, difficulty, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := rr.db.ExecContext(ctx, query,
		room.ID,
		room.RoomType,
		room.MaxPlayers,
		room.Status,
		room.Score,
		room.Waves,
		room.Difficulty,
		room.CreatedAt,
		room.StartedAt,
		room.FinishedAt,
	)

	return err
}

func (rr *RoomRepository) GetByID(ctx context.Context, roomID string) (*Room, error) {
	query := `
		SELECT id, room_type, max_players, status, score, waves, difficulty, created_at, started_at, finished_at
		FROM rooms
		WHERE id = ?
	`

	var room Room
	err := rr.db.QueryRowContext(ctx, query, roomID).Scan(
		&room.ID,
		&room.RoomType,
		&room.MaxPlayers,
		&room.Status,
		&room.Score,
		&room.Waves,
		&room.Difficulty,
		&room.CreatedAt,
		&room.StartedAt,
		&room.FinishedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("room not found: %s", roomID)
		}
		return nil, err
	}

	return &room, nil
}

func (rr *RoomRepository) Update(ctx context.Context, room *Room) error {
	query := `
		UPDATE rooms
		SET status = ?, score = ?, waves = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`

	result, err := rr.db.ExecContext(ctx, query,
		room.Status,
		room.Score,
		room.Waves,
		room.StartedAt,
		room.FinishedAt,
		room.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", room.ID)
	}

	return nil
}

func (rr *RoomRepository) UpdateStatus(ctx context.Context, roomID, status string) error {
	query := `UPDATE rooms SET status = ? WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, status, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) UpdateScore(ctx context.Context, roomID string, score int64) error {
	query := `UPDATE rooms SET score = ? WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, score, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) UpdateWaves(ctx context.Context, roomID string, waves int32) error {
	query := `UPDATE rooms SET waves = ? WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, waves, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) StartGame(ctx context.Context, roomID string) error {
	now := time.Now()
	query := `UPDATE rooms SET status = 'playing', started_at = ? WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, now, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) FinishGame(ctx context.Context, roomID string) error {
	now := time.Now()
	query := `UPDATE rooms SET status = 'finished', finished_at = ? WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, now, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) Delete(ctx context.Context, roomID string) error {
	query := `DELETE FROM rooms WHERE id = ?`

	result, err := rr.db.ExecContext(ctx, query, roomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room not found: %s", roomID)
	}

	return nil
}

func (rr *RoomRepository) List(ctx context.Context, limit, offset int) ([]*Room, error) {
	query := `
		SELECT id, room_type, max_players, status, score, waves, difficulty, created_at, started_at, finished_at
		FROM rooms
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := rr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(
			&room.ID,
			&room.RoomType,
			&room.MaxPlayers,
			&room.Status,
			&room.Score,
			&room.Waves,
			&room.Difficulty,
			&room.CreatedAt,
			&room.StartedAt,
			&room.FinishedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, nil
}

func (rr *RoomRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*Room, error) {
	query := `
		SELECT id, room_type, max_players, status, score, waves, difficulty, created_at, started_at, finished_at
		FROM rooms
		WHERE status = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := rr.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(
			&room.ID,
			&room.RoomType,
			&room.MaxPlayers,
			&room.Status,
			&room.Score,
			&room.Waves,
			&room.Difficulty,
			&room.CreatedAt,
			&room.StartedAt,
			&room.FinishedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, nil
}

func (rr *RoomRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM rooms`

	var count int64
	err := rr.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (rr *RoomRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	query := `SELECT COUNT(*) FROM rooms WHERE status = ?`

	var count int64
	err := rr.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (rr *RoomRepository) GetStats(ctx context.Context, roomID string) (*RoomStats, error) {
	query := `
		SELECT
			COUNT(DISTINCT rp.id) as total_games,
			COUNT(DISTINCT rp.player_id) as total_players,
			COALESCE(SUM(r.score), 0) as total_score,
			COALESCE(SUM(r.waves), 0) as total_waves,
			COALESCE(MAX(r.score), 0) as best_score,
			COALESCE(MAX(r.waves), 0) as best_waves,
			COALESCE(AVG(r.score), 0) as average_score,
			COALESCE(AVG(r.waves), 0) as average_waves
		FROM rooms r
		LEFT JOIN room_players rp ON r.id = rp.room_id
		WHERE r.id = ?
	`

	var stats RoomStats
	err := rr.db.QueryRowContext(ctx, query, roomID).Scan(
		&stats.TotalGames,
		&stats.TotalPlayers,
		&stats.TotalScore,
		&stats.TotalWaves,
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

func (rr *RoomRepository) BatchGet(ctx context.Context, roomIDs []string) (map[string]*Room, error) {
	if len(roomIDs) == 0 {
		return make(map[string]*Room), nil
	}

	query := `
		SELECT id, room_type, max_players, status, score, waves, difficulty, created_at, started_at, finished_at
		FROM rooms
		WHERE id IN (` + placeholders(len(roomIDs)) + `)
	`

	args := make([]interface{}, len(roomIDs))
	for i, id := range roomIDs {
		args[i] = id
	}

	rows, err := rr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make(map[string]*Room)
	for rows.Next() {
		var room Room
		err := rows.Scan(
			&room.ID,
			&room.RoomType,
			&room.MaxPlayers,
			&room.Status,
			&room.Score,
			&room.Waves,
			&room.Difficulty,
			&room.CreatedAt,
			&room.StartedAt,
			&room.FinishedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms[room.ID] = &room
	}

	return rooms, nil
}

func (rr *RoomRepository) Exists(ctx context.Context, roomID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)`

	var exists bool
	err := rr.db.QueryRowContext(ctx, query, roomID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
