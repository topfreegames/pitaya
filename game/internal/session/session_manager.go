package session

import (
	"context"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/session"
)

const (
	SessionStateTTL = 5 * time.Minute
)

type SessionState struct {
	SessionID  int64
	UserID     string
	RoomID     string
	GameState  []byte
	LastActive time.Time
	CreatedAt  time.Time
}

type SessionManager struct {
	app   pitaya.Pitaya
	mu    sync.RWMutex
	states map[string]*SessionState
	ttl   time.Duration
}

func NewSessionManager(app pitaya.Pitaya) *SessionManager {
	return &SessionManager{
		app:    app,
		states: make(map[string]*SessionState),
		ttl:    SessionStateTTL,
	}
}

func (sm *SessionManager) OnSessionBind(ctx context.Context, s session.Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[s.UID()] = &SessionState{
		SessionID:  s.ID(),
		UserID:     s.UID(),
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
	}
}

func (sm *SessionManager) OnSessionClose(s session.Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if state, ok := sm.states[s.UID()]; ok {
		state.LastActive = time.Now()
	}
}

func (sm *SessionManager) SaveSessionState(ctx context.Context, sessionID int64, userID, roomID string, gameState []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	state := &SessionState{
		SessionID:  sessionID,
		UserID:     userID,
		RoomID:     roomID,
		GameState:  gameState,
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
	}
	sm.states[userID] = state
	return nil
}

func (sm *SessionManager) RestoreSessionState(ctx context.Context, userID string) (*SessionState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	state, ok := sm.states[userID]
	if !ok {
		return nil, pitaya.Error(nil, "SESSION_NOT_FOUND", map[string]string{"user_id": userID})
	}
	if time.Since(state.LastActive) > sm.ttl {
		return nil, pitaya.Error(nil, "SESSION_EXPIRED", map[string]string{"user_id": userID})
	}
	return state, nil
}

func (sm *SessionManager) UpdateRoomID(userID, roomID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if state, ok := sm.states[userID]; ok {
		state.RoomID = roomID
		state.LastActive = time.Now()
	}
}

func (sm *SessionManager) UpdateGameState(userID string, gameState []byte) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if state, ok := sm.states[userID]; ok {
		state.GameState = gameState
		state.LastActive = time.Now()
	}
}

func (sm *SessionManager) RemoveSessionState(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.states, userID)
}

func (sm *SessionManager) CleanupExpiredSessions() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	now := time.Now()
	expiredCount := 0
	for userID, state := range sm.states {
		if now.Sub(state.LastActive) > sm.ttl {
			delete(sm.states, userID)
			expiredCount++
		}
	}
	return expiredCount
}
