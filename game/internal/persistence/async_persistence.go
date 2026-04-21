package persistence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/database"
)

type PersistenceEvent struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

type PlayerUpdateEvent struct {
	PlayerID string
	Field    string
	Value    interface{}
}

type RoomUpdateEvent struct {
	RoomID string
	Field  string
	Value  interface{}
}

type GameRecordEvent struct {
	PlayerID        string
	RoomID          string
	Score           int64
	WavesSurvived   int32
	EnemiesDefeated int32
	DurationSeconds int32
}

type RoomPlayerEvent struct {
	RoomID   string
	PlayerID string
	Field    string
	Value    interface{}
}

type AsyncPersistence struct {
	eventChan       chan PersistenceEvent
	workerCount     int
	database        *database.DatabaseManager
	playerRepo      *database.PlayerRepository
	roomRepo        *database.RoomRepository
	gameRecordRepo  *database.GameRecordRepository
	roomPlayerRepo  *database.RoomPlayerRepository
	shutdown        chan struct{}
	done            chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

func NewAsyncPersistence(db *database.DatabaseManager, workerCount int) *AsyncPersistence {
	ap := &AsyncPersistence{
		eventChan:       make(chan PersistenceEvent, 10000),
		workerCount:      workerCount,
		database:         db,
		playerRepo:       db.GetPlayerRepo(),
		roomRepo:         db.GetRoomRepo(),
		gameRecordRepo:   db.GetGameRecordRepo(),
		roomPlayerRepo:   db.GetRoomPlayerRepo(),
		shutdown:         make(chan struct{}),
		done:             make(chan struct{}),
	}

	ap.startWorkers()
	return ap
}

func (ap *AsyncPersistence) startWorkers() {
	for i := 0; i < ap.workerCount; i++ {
		ap.wg.Add(1)
		go ap.worker(i)
	}
}

func (ap *AsyncPersistence) worker(id int) {
	defer ap.wg.Done()

	for {
		select {
		case event := <-ap.eventChan:
			ap.processEvent(event)

		case <-ap.shutdown:
			fmt.Printf("Persistence worker %d shutting down\n", id)
			return
		}
	}
}

func (ap *AsyncPersistence) processEvent(event PersistenceEvent) {
	ctx := context.Background()

	switch event.Type {
	case "player_update":
		ap.processPlayerUpdate(ctx, event)
	case "room_update":
		ap.processRoomUpdate(ctx, event)
	case "game_record":
		ap.processGameRecord(ctx, event)
	case "room_player":
		ap.processRoomPlayer(ctx, event)
	default:
		fmt.Printf("Unknown event type: %s\n", event.Type)
	}
}

func (ap *AsyncPersistence) processPlayerUpdate(ctx context.Context, event PersistenceEvent) {
	data, ok := event.Data.(PlayerUpdateEvent)
	if !ok {
		return
	}

	switch data.Field {
	case "coins":
		delta, ok := data.Value.(int64)
		if !ok {
			return
		}
		ap.playerRepo.UpdateCoins(ctx, data.PlayerID, delta)

	case "exp":
		delta, ok := data.Value.(int64)
		if !ok {
			return
		}
		ap.playerRepo.UpdateExp(ctx, data.PlayerID, delta)

	case "gems":
		delta, ok := data.Value.(int64)
		if !ok {
			return
		}
		ap.playerRepo.UpdateGems(ctx, data.PlayerID, delta)

	case "level":
		level, ok := data.Value.(int32)
		if !ok {
			return
		}
		ap.playerRepo.UpdateLevel(ctx, data.PlayerID, level)
	}
}

func (ap *AsyncPersistence) processRoomUpdate(ctx context.Context, event PersistenceEvent) {
	data, ok := event.Data.(RoomUpdateEvent)
	if !ok {
		return
	}

	switch data.Field {
	case "status":
		status, ok := data.Value.(string)
		if !ok {
			return
		}
		ap.roomRepo.UpdateStatus(ctx, data.RoomID, status)

	case "score":
		score, ok := data.Value.(int64)
		if !ok {
			return
		}
		ap.roomRepo.UpdateScore(ctx, data.RoomID, score)

	case "waves":
		waves, ok := data.Value.(int32)
		if !ok {
			return
		}
		ap.roomRepo.UpdateWaves(ctx, data.RoomID, waves)
	}
}

func (ap *AsyncPersistence) processGameRecord(ctx context.Context, event PersistenceEvent) {
	data, ok := event.Data.(GameRecordEvent)
	if !ok {
		return
	}

	record := &database.GameRecord{
		PlayerID:        data.PlayerID,
		RoomID:          data.RoomID,
		Score:           data.Score,
		WavesSurvived:   data.WavesSurvived,
		EnemiesDefeated: data.EnemiesDefeated,
		DurationSeconds: data.DurationSeconds,
		PlayedAt:        time.Now(),
	}

	ap.gameRecordRepo.Create(ctx, record)
}

func (ap *AsyncPersistence) processRoomPlayer(ctx context.Context, event PersistenceEvent) {
	data, ok := event.Data.(RoomPlayerEvent)
	if !ok {
		return
	}

	switch data.Field {
	case "score":
		score, ok := data.Value.(int64)
		if !ok {
			return
		}
		ap.roomPlayerRepo.UpdateScore(ctx, data.RoomID, data.PlayerID, score)

	case "waves":
		waves, ok := data.Value.(int32)
		if !ok {
			return
		}
		ap.roomPlayerRepo.UpdateWaves(ctx, data.RoomID, data.PlayerID, waves)

	case "leave":
		ap.roomPlayerRepo.LeaveRoom(ctx, data.RoomID, data.PlayerID)
	}
}

func (ap *AsyncPersistence) UpdatePlayerCoins(playerID string, delta int64) error {
	event := PersistenceEvent{
		Type:      "player_update",
		Data:      PlayerUpdateEvent{PlayerID: playerID, Field: "coins", Value: delta},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdatePlayerExp(playerID string, delta int64) error {
	event := PersistenceEvent{
		Type:      "player_update",
		Data:      PlayerUpdateEvent{PlayerID: playerID, Field: "exp", Value: delta},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdatePlayerGems(playerID string, delta int64) error {
	event := PersistenceEvent{
		Type:      "player_update",
		Data:      PlayerUpdateEvent{PlayerID: playerID, Field: "gems", Value: delta},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdatePlayerLevel(playerID string, level int32) error {
	event := PersistenceEvent{
		Type:      "player_update",
		Data:      PlayerUpdateEvent{PlayerID: playerID, Field: "level", Value: level},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdateRoomStatus(roomID, status string) error {
	event := PersistenceEvent{
		Type:      "room_update",
		Data:      RoomUpdateEvent{RoomID: roomID, Field: "status", Value: status},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdateRoomScore(roomID string, score int64) error {
	event := PersistenceEvent{
		Type:      "room_update",
		Data:      RoomUpdateEvent{RoomID: roomID, Field: "score", Value: score},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdateRoomWaves(roomID string, waves int32) error {
	event := PersistenceEvent{
		Type:      "room_update",
		Data:      RoomUpdateEvent{RoomID: roomID, Field: "waves", Value: waves},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) CreateGameRecord(playerID, roomID string, score int64, waves, enemies, duration int32) error {
	event := PersistenceEvent{
		Type:      "game_record",
		Data:      GameRecordEvent{
			PlayerID:        playerID,
			RoomID:          roomID,
			Score:           score,
			WavesSurvived:   waves,
			EnemiesDefeated: enemies,
			DurationSeconds: duration,
		},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdateRoomPlayerScore(roomID, playerID string, score int64) error {
	event := PersistenceEvent{
		Type:      "room_player",
		Data:      RoomPlayerEvent{RoomID: roomID, PlayerID: playerID, Field: "score", Value: score},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) UpdateRoomPlayerWaves(roomID, playerID string, waves int32) error {
	event := PersistenceEvent{
		Type:      "room_player",
		Data:      RoomPlayerEvent{RoomID: roomID, PlayerID: playerID, Field: "waves", Value: waves},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) PlayerLeaveRoom(roomID, playerID string) error {
	event := PersistenceEvent{
		Type:      "room_player",
		Data:      RoomPlayerEvent{RoomID: roomID, PlayerID: playerID, Field: "leave", Value: nil},
		Timestamp: time.Now(),
	}

	select {
	case ap.eventChan <- event:
		return nil
	case <-ap.shutdown:
		return fmt.Errorf("async persistence is shutting down")
	default:
		return fmt.Errorf("event channel is full")
	}
}

func (ap *AsyncPersistence) GetEventQueueSize() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return len(ap.eventChan)
}

func (ap *AsyncPersistence) Shutdown() {
	close(ap.shutdown)
	ap.wg.Wait()
	close(ap.done)
}

func (ap *AsyncPersistence) IsShutdown() bool {
	select {
	case <-ap.done:
		return true
	default:
		return false
	}
}

func (ap *AsyncPersistence) GetStats() map[string]interface{} {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":    ap.workerCount,
		"event_queue_size": len(ap.eventChan),
		"is_shutdown":      ap.IsShutdown(),
	}
}
