package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type RoomPlayerRepository struct {
	db *MySQLDatabase
}

func NewRoomPlayerRepository(db *MySQLDatabase) *RoomPlayerRepository {
	return &RoomPlayerRepository{db: db}
}

func (rpr *RoomPlayerRepository) Create(ctx context.Context, roomPlayer *RoomPlayer) error {
	query := `
		INSERT INTO room_players (room_id, player_id, score, waves_survived, joined_at, left_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := rpr.db.ExecContext(ctx, query,
		roomPlayer.RoomID,
		roomPlayer.PlayerID,
		roomPlayer.Score,
		roomPlayer.WavesSurvived,
		roomPlayer.JoinedAt,
		roomPlayer.LeftAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	roomPlayer.ID = id
	return nil
}

func (rpr *RoomPlayerRepository) GetByID(ctx context.Context, id int64) (*RoomPlayer, error) {
	query := `
		SELECT id, room_id, player_id, score, waves_survived, joined_at, left_at
		FROM room_players
		WHERE id = ?
	`

	var roomPlayer RoomPlayer
	err := rpr.db.QueryRowContext(ctx, query, id).Scan(
		&roomPlayer.ID,
		&roomPlayer.RoomID,
		&roomPlayer.PlayerID,
		&roomPlayer.Score,
		&roomPlayer.WavesSurvived,
		&roomPlayer.JoinedAt,
		&roomPlayer.LeftAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("room player not found: %d", id)
		}
		return nil, err
	}

	return &roomPlayer, nil
}

func (rpr *RoomPlayerRepository) GetByRoomID(ctx context.Context, roomID string) ([]*RoomPlayer, error) {
	query := `
		SELECT id, room_id, player_id, score, waves_survived, joined_at, left_at
		FROM room_players
		WHERE room_id = ?
		ORDER BY joined_at ASC
	`

	rows, err := rpr.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roomPlayers := make([]*RoomPlayer, 0)
	for rows.Next() {
		var roomPlayer RoomPlayer
		err := rows.Scan(
			&roomPlayer.ID,
			&roomPlayer.RoomID,
			&roomPlayer.PlayerID,
			&roomPlayer.Score,
			&roomPlayer.WavesSurvived,
			&roomPlayer.JoinedAt,
			&roomPlayer.LeftAt,
		)
		if err != nil {
			return nil, err
		}
		roomPlayers = append(roomPlayers, &roomPlayer)
	}

	return roomPlayers, nil
}

func (rpr *RoomPlayerRepository) GetByPlayerID(ctx context.Context, playerID string) ([]*RoomPlayer, error) {
	query := `
		SELECT id, room_id, player_id, score, waves_survived, joined_at, left_at
		FROM room_players
		WHERE player_id = ?
		ORDER BY joined_at DESC
	`

	rows, err := rpr.db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roomPlayers := make([]*RoomPlayer, 0)
	for rows.Next() {
		var roomPlayer RoomPlayer
		err := rows.Scan(
			&roomPlayer.ID,
			&roomPlayer.RoomID,
			&roomPlayer.PlayerID,
			&roomPlayer.Score,
			&roomPlayer.WavesSurvived,
			&roomPlayer.JoinedAt,
			&roomPlayer.LeftAt,
		)
		if err != nil {
			return nil, err
		}
		roomPlayers = append(roomPlayers, &roomPlayer)
	}

	return roomPlayers, nil
}

func (rpr *RoomPlayerRepository) GetByRoomAndPlayer(ctx context.Context, roomID, playerID string) (*RoomPlayer, error) {
	query := `
		SELECT id, room_id, player_id, score, waves_survived, joined_at, left_at
		FROM room_players
		WHERE room_id = ? AND player_id = ?
	`

	var roomPlayer RoomPlayer
	err := rpr.db.QueryRowContext(ctx, query, roomID, playerID).Scan(
		&roomPlayer.ID,
		&roomPlayer.RoomID,
		&roomPlayer.PlayerID,
		&roomPlayer.Score,
		&roomPlayer.WavesSurvived,
		&roomPlayer.JoinedAt,
		&roomPlayer.LeftAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("room player not found: room_id=%s, player_id=%s", roomID, playerID)
		}
		return nil, err
	}

	return &roomPlayer, nil
}

func (rpr *RoomPlayerRepository) Update(ctx context.Context, roomPlayer *RoomPlayer) error {
	query := `
		UPDATE room_players
		SET score = ?, waves_survived = ?, left_at = ?
		WHERE id = ?
	`

	result, err := rpr.db.ExecContext(ctx, query,
		roomPlayer.Score,
		roomPlayer.WavesSurvived,
		roomPlayer.LeftAt,
		roomPlayer.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room player not found: %d", roomPlayer.ID)
	}

	return nil
}

func (rpr *RoomPlayerRepository) UpdateScore(ctx context.Context, roomID, playerID string, score int64) error {
	query := `UPDATE room_players SET score = ? WHERE room_id = ? AND player_id = ?`

	result, err := rpr.db.ExecContext(ctx, query, score, roomID, playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room player not found: room_id=%s, player_id=%s", roomID, playerID)
	}

	return nil
}

func (rpr *RoomPlayerRepository) UpdateWaves(ctx context.Context, roomID, playerID string, waves int32) error {
	query := `UPDATE room_players SET waves_survived = ? WHERE room_id = ? AND player_id = ?`

	result, err := rpr.db.ExecContext(ctx, query, waves, roomID, playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room player not found: room_id=%s, player_id=%s", roomID, playerID)
	}

	return nil
}

func (rpr *RoomPlayerRepository) LeaveRoom(ctx context.Context, roomID, playerID string) error {
	now := time.Now()
	query := `UPDATE room_players SET left_at = ? WHERE room_id = ? AND player_id = ? AND left_at IS NULL`

	result, err := rpr.db.ExecContext(ctx, query, now, roomID, playerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room player not found or already left: room_id=%s, player_id=%s", roomID, playerID)
	}

	return nil
}

func (rpr *RoomPlayerRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM room_players WHERE id = ?`

	result, err := rpr.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("room player not found: %d", id)
	}

	return nil
}

func (rpr *RoomPlayerRepository) DeleteByRoomID(ctx context.Context, roomID string) error {
	query := `DELETE FROM room_players WHERE room_id = ?`

	_, err := rpr.db.ExecContext(ctx, query, roomID)
	return err
}

func (rpr *RoomPlayerRepository) DeleteByPlayerID(ctx context.Context, playerID string) error {
	query := `DELETE FROM room_players WHERE player_id = ?`

	_, err := rpr.db.ExecContext(ctx, query, playerID)
	return err
}

func (rpr *RoomPlayerRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM room_players`

	var count int64
	err := rpr.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (rpr *RoomPlayerRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	query := `SELECT COUNT(*) FROM room_players WHERE room_id = ?`

	var count int64
	err := rpr.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (rpr *RoomPlayerRepository) CountByPlayerID(ctx context.Context, playerID string) (int64, error) {
	query := `SELECT COUNT(*) FROM room_players WHERE player_id = ?`

	var count int64
	err := rpr.db.QueryRowContext(ctx, query, playerID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (rpr *RoomPlayerRepository) GetActivePlayers(ctx context.Context, roomID string) ([]*RoomPlayer, error) {
	query := `
		SELECT id, room_id, player_id, score, waves_survived, joined_at, left_at
		FROM room_players
		WHERE room_id = ? AND left_at IS NULL
		ORDER BY joined_at ASC
	`

	rows, err := rpr.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roomPlayers := make([]*RoomPlayer, 0)
	for rows.Next() {
		var roomPlayer RoomPlayer
		err := rows.Scan(
			&roomPlayer.ID,
			&roomPlayer.RoomID,
			&roomPlayer.PlayerID,
			&roomPlayer.Score,
			&roomPlayer.WavesSurvived,
			&roomPlayer.JoinedAt,
			&roomPlayer.LeftAt,
		)
		if err != nil {
			return nil, err
		}
		roomPlayers = append(roomPlayers, &roomPlayer)
	}

	return roomPlayers, nil
}

func (rpr *RoomPlayerRepository) BatchCreate(ctx context.Context, roomPlayers []*RoomPlayer) error {
	if len(roomPlayers) == 0 {
		return nil
	}

	query := `
		INSERT INTO room_players (room_id, player_id, score, waves_survived, joined_at, left_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	stmt, err := rpr.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, roomPlayer := range roomPlayers {
		_, err := stmt.ExecContext(ctx,
			roomPlayer.RoomID,
			roomPlayer.PlayerID,
			roomPlayer.Score,
			roomPlayer.WavesSurvived,
			roomPlayer.JoinedAt,
			roomPlayer.LeftAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
