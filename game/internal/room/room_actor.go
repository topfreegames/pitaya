package room

import (
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

const (
	GameTickRate = 100 * time.Millisecond
	MaxPlayersPerRoom = 4
	RoomTTL = 2 * time.Hour
)

type RoomEvent interface {
	Process(*RoomState) error
}

type RoomState struct {
	RoomID         string
	RoomType       string
	MaxPlayers     int32
	CurrentPlayers int32
	Status         proto.RoomStatus
	Players        map[string]*PlayerState
	CurrentWave    int32
	Score          int32
	GameTime       time.Duration
	Difficulty     int32
	Tick           uint64
	LastUpdate     time.Time
	CreatedAt      time.Time
	StartedAt      time.Time
	FinishedAt     time.Time
}

type PlayerState struct {
	PlayerID    string
	Name        string
	Level       int32
	Health      int32
	MaxHealth   int32
	Position    *proto.Position
	Score       int32
	LastAction  time.Time
	JoinedAt    time.Time
}

type RoomActor struct {
	roomID    string
	eventChan chan RoomEvent
	state     *RoomState
	gameLoop  *GameLoop
	shutdown  chan struct{}
	done      chan struct{}
	mu        sync.RWMutex
}

func NewRoomActor(roomID, roomType string, maxPlayers int32, difficulty int32) *RoomActor {
	actor := &RoomActor{
		roomID:    roomID,
		eventChan: make(chan RoomEvent, 1000),
		state: &RoomState{
			RoomID:         roomID,
			RoomType:       roomType,
			MaxPlayers:     maxPlayers,
			CurrentPlayers: 0,
			Status:         proto.RoomStatus_WAITING,
			Players:        make(map[string]*PlayerState),
			CurrentWave:    0,
			Score:          0,
			GameTime:       0,
			Difficulty:     difficulty,
			Tick:           0,
			LastUpdate:     time.Now(),
			CreatedAt:      time.Now(),
		},
		gameLoop:  NewGameLoop(GameTickRate),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}

	actor.gameLoop.SetState(actor.state)
	actor.gameLoop.Start()
	go actor.run()

	return actor
}

func (ra *RoomActor) run() {
	defer close(ra.done)
	defer ra.gameLoop.Shutdown()

	for {
		select {
		case event := <-ra.eventChan:
			if err := event.Process(ra.state); err != nil {
				fmt.Printf("Error processing event in room %s: %v\n", ra.roomID, err)
			}

		case <-ra.shutdown:
			fmt.Printf("Room actor %s shutting down\n", ra.roomID)
			return
		}
	}
}

func (ra *RoomActor) SendEvent(event RoomEvent) error {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	select {
	case ra.eventChan <- event:
		return nil
	case <-ra.shutdown:
		return fmt.Errorf("room %s is shutting down", ra.roomID)
	default:
		return fmt.Errorf("room %s event channel is full", ra.roomID)
	}
}

func (ra *RoomActor) GetState() *RoomState {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return ra.state
}

func (ra *RoomActor) Shutdown() {
	close(ra.shutdown)
	<-ra.done
}

type GameLoop struct {
	ticker      *time.Ticker
	shutdown    chan struct{}
	done        chan struct{}
	state       *RoomState
	tickRate    time.Duration
	currentTick uint64
}

func NewGameLoop(tickRate time.Duration) *GameLoop {
	return &GameLoop{
		ticker:      time.NewTicker(tickRate),
		shutdown:    make(chan struct{}),
		done:        make(chan struct{}),
		tickRate:    tickRate,
		currentTick: 0,
	}
}

func (gl *GameLoop) SetState(state *RoomState) {
	gl.state = state
}

func (gl *GameLoop) Start() {
	go gl.run()
}

func (gl *GameLoop) run() {
	defer close(gl.done)
	defer gl.ticker.Stop()

	for {
		select {
		case <-gl.ticker.C:
			gl.processTick()

		case <-gl.shutdown:
			fmt.Printf("Game loop shutting down\n")
			return
		}
	}
}

func (gl *GameLoop) processTick() {
	gl.currentTick++
	if gl.state != nil {
		gl.state.Tick = gl.currentTick
		gl.state.LastUpdate = time.Now()
		gl.state.GameTime += gl.tickRate
	}
}

func (gl *GameLoop) Shutdown() {
	close(gl.shutdown)
	select {
	case <-gl.done:
	case <-time.After(100 * time.Millisecond):
	}
}
