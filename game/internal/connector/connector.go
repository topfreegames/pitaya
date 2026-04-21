package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/game/internal/proto"
	sessionpkg "github.com/topfreegames/pitaya/v3/game/internal/session"
)

const (
	HeartbeatInterval = 15 * time.Second
	HeartbeatTimeout  = 45 * time.Second
)

type HeartbeatManager struct {
	mu              sync.RWMutex
	sessionHeartbeats map[int64]time.Time
	ticker          *time.Ticker
	shutdown        chan struct{}
	done            chan struct{}
	interval        time.Duration
	timeout         time.Duration
}

type HeartbeatConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

func NewHeartbeatManager(config HeartbeatConfig) *HeartbeatManager {
	interval := config.Interval
	if interval == 0 {
		interval = HeartbeatInterval
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = HeartbeatTimeout
	}
	return &HeartbeatManager{
		sessionHeartbeats: make(map[int64]time.Time),
		ticker:          time.NewTicker(interval),
		shutdown:        make(chan struct{}),
		done:            make(chan struct{}),
		interval:        interval,
		timeout:         timeout,
	}
}

func (hm *HeartbeatManager) Start() {
	go hm.run()
}

func (hm *HeartbeatManager) run() {
	defer close(hm.done)
	defer hm.ticker.Stop()
	for {
		select {
		case <-hm.ticker.C:
		case <-hm.shutdown:
			return
		}
	}
}

func (hm *HeartbeatManager) UpdateHeartbeat(sessionID int64) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.sessionHeartbeats[sessionID] = time.Now()
}

func (hm *HeartbeatManager) CheckDeadSessions(cleanFunc func(int64)) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	now := time.Now()
	deadSessions := []int64{}
	for sessionID, lastHeartbeat := range hm.sessionHeartbeats {
		if now.Sub(lastHeartbeat) > hm.timeout {
			deadSessions = append(deadSessions, sessionID)
		}
	}
	for _, sessionID := range deadSessions {
		delete(hm.sessionHeartbeats, sessionID)
		if cleanFunc != nil {
			cleanFunc(sessionID)
		}
	}
}

func (hm *HeartbeatManager) RemoveSession(sessionID int64) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.sessionHeartbeats, sessionID)
}

func (hm *HeartbeatManager) Shutdown() {
	close(hm.shutdown)
	<-hm.done
}

type Connector struct {
	component.Base
	app             pitaya.Pitaya
	HeartbeatManager *HeartbeatManager
	SessionManager   *sessionpkg.SessionManager
}

func NewConnector(app pitaya.Pitaya) *Connector {
	hm := NewHeartbeatManager(HeartbeatConfig{
		Interval: HeartbeatInterval,
		Timeout:  HeartbeatTimeout,
	})
	sm := sessionpkg.NewSessionManager(app)
	return &Connector{
		app:             app,
		HeartbeatManager: hm,
		SessionManager:   sm,
	}
}

func (c *Connector) Init() {
	c.HeartbeatManager.Start()
}

func (c *Connector) AfterInit() {
}

func (c *Connector) BeforeShutdown() {
}

func (c *Connector) Shutdown() {
	c.HeartbeatManager.Shutdown()
}

func (c *Connector) Connect(ctx context.Context, req *proto.ConnectRequest) (*proto.ConnectResponse, error) {
	s := c.app.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(nil, "NO_SESSION", map[string]string{"error": "session not found"})
	}
	if req.PlayerId == "" {
		return nil, pitaya.Error(nil, "INVALID_REQUEST", map[string]string{"error": "player_id is required"})
	}
	if err := s.Bind(ctx, req.PlayerId); err != nil {
		return nil, pitaya.Error(err, "BIND_FAILED", map[string]string{"player_id": req.PlayerId})
	}
	c.HeartbeatManager.UpdateHeartbeat(s.ID())
	c.SessionManager.OnSessionBind(ctx, s)
	return &proto.ConnectResponse{
		SessionId:  fmt.Sprintf("%d", s.ID()),
		ServerTime: time.Now().Unix(),
	}, nil
}

func (c *Connector) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	s := c.app.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(nil, "NO_SESSION", map[string]string{"error": "session not found"})
	}
	c.HeartbeatManager.UpdateHeartbeat(s.ID())
	status := proto.HeartbeatStatus_OK
	return &proto.HeartbeatResponse{
		ServerTime: time.Now().Unix(),
		Status:     status,
	}, nil
}

func (c *Connector) Disconnect(ctx context.Context, req *proto.DisconnectRequest) (*proto.DisconnectResponse, error) {
	s := c.app.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(nil, "NO_SESSION", map[string]string{"error": "session not found"})
	}
	c.HeartbeatManager.RemoveSession(s.ID())
	c.SessionManager.OnSessionClose(s)
	s.Close()
	return &proto.DisconnectResponse{Success: true}, nil
}

func (c *Connector) Reconnect(ctx context.Context, req *proto.ReconnectRequest) (*proto.ReconnectResponse, error) {
	s := c.app.GetSessionFromCtx(ctx)
	if s == nil {
		return nil, pitaya.Error(nil, "NO_SESSION", map[string]string{"error": "session not found"})
	}
	state, err := c.SessionManager.RestoreSessionState(ctx, req.PlayerId)
	if err != nil {
		if err := s.Bind(ctx, req.PlayerId); err != nil {
			return nil, pitaya.Error(err, "BIND_FAILED", map[string]string{"player_id": req.PlayerId})
		}
	c.HeartbeatManager.UpdateHeartbeat(s.ID())
		return &proto.ReconnectResponse{
			Success:    true,
			SessionId:  fmt.Sprintf("%d", s.ID()),
			ServerTime: time.Now().Unix(),
		}, nil
	}
	if err := s.Bind(ctx, req.PlayerId); err != nil {
		return nil, pitaya.Error(err, "BIND_FAILED", map[string]string{"player_id": req.PlayerId})
	}
	c.HeartbeatManager.UpdateHeartbeat(s.ID())
	return &proto.ReconnectResponse{
		Success:    true,
		SessionId:  fmt.Sprintf("%d", s.ID()),
		ServerTime: time.Now().Unix(),
		RoomId:     state.RoomID,
		GameState:  state.GameState,
	}, nil
}
