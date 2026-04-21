package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v3/pkg/client"
	"github.com/topfreegames/pitaya/v3/pkg/serialize"
	"github.com/topfreegames/pitaya/v3/pkg/serialize/json"
	"game-client/internal/proto"
)

type GameClient struct {
	client      *client.Client
	playerID    string
	sessionID   string
	roomID      string
	connected   bool
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	heartbeatCh chan struct{}
	serializer  serialize.Serializer
}

type Config struct {
	ServerAddr string
	PlayerID   string
}

func NewGameClient(config *Config) (*GameClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	gameClient := client.New(logrus.ErrorLevel)
	serializer := json.NewSerializer()

	return &GameClient{
		client:      gameClient,
		playerID:    config.PlayerID,
		connected:   false,
		ctx:         ctx,
		cancel:      cancel,
		heartbeatCh: make(chan struct{}, 10),
		serializer:  serializer,
	}, nil
}

func (gc *GameClient) Connect() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.connected {
		return fmt.Errorf("already connected")
	}

	err := gc.client.ConnectToWS("127.0.0.1:3250", "/")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	gc.connected = true

	go gc.startHeartbeat()

	return nil
}

func (gc *GameClient) Disconnect() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if !gc.connected {
		return fmt.Errorf("not connected")
	}

	gc.cancel()

	gc.client.Disconnect()

	gc.connected = false
	return nil
}

func (gc *GameClient) IsConnected() bool {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.connected
}

func (gc *GameClient) GetPlayerID() string {
	return gc.playerID
}

func (gc *GameClient) GetSessionID() string {
	return gc.sessionID
}

func (gc *GameClient) GetRoomID() string {
	return gc.roomID
}

func (gc *GameClient) startHeartbeat() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if gc.IsConnected() {
				err := gc.SendHeartbeat()
				if err != nil {
					fmt.Printf("Heartbeat error: %v\n", err)
				}
			}
		case <-gc.ctx.Done():
			return
		}
	}
}

func (gc *GameClient) SendHeartbeat() error {
	req := &proto.HeartbeatRequest{}
	resp := &proto.HeartbeatResponse{}

	err := gc.sendRequest("connector.connector.Heartbeat", req, resp)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	return nil
}

func (gc *GameClient) ConnectToGame() error {
	req := &proto.ConnectRequest{
		PlayerId: gc.playerID,
	}
	resp := &proto.ConnectResponse{}

	err := gc.sendRequest("connector.connector.Connect", req, resp)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	gc.sessionID = resp.SessionId
	return nil
}

func (gc *GameClient) CreateRoom(roomType string, maxPlayers int32) (*proto.CreateRoomResponse, error) {
	req := &proto.CreateRoomRequest{
		RoomType:   roomType,
		MaxPlayers: maxPlayers,
	}
	resp := &proto.CreateRoomResponse{}

	err := gc.sendRequest("room.room.CreateRoom", req, resp)
	if err != nil {
		return nil, fmt.Errorf("create room failed: %w", err)
	}

	gc.roomID = resp.RoomId
	return resp, nil
}

func (gc *GameClient) JoinRoom(roomID string) (*proto.JoinRoomResponse, error) {
	req := &proto.JoinRoomRequest{
		RoomId: roomID,
	}
	resp := &proto.JoinRoomResponse{}

	err := gc.sendRequest("room.room.JoinRoom", req, resp)
	if err != nil {
		return nil, fmt.Errorf("join room failed: %w", err)
	}

	gc.roomID = roomID
	return resp, nil
}

func (gc *GameClient) LeaveRoom() (*proto.LeaveRoomResponse, error) {
	req := &proto.LeaveRoomRequest{
		RoomId: gc.roomID,
	}
	resp := &proto.LeaveRoomResponse{}

	err := gc.sendRequest("room.room.LeaveRoom", req, resp)
	if err != nil {
		return nil, fmt.Errorf("leave room failed: %w", err)
	}

	gc.roomID = ""
	return resp, nil
}

func (gc *GameClient) StartGame() (*proto.StartGameResponse, error) {
	req := &proto.StartGameRequest{
		RoomId: gc.roomID,
	}
	resp := &proto.StartGameResponse{}

	err := gc.sendRequest("room.room.StartGame", req, resp)
	if err != nil {
		return nil, fmt.Errorf("start game failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) MovePlayer(x, y, z float32) (*proto.MoveResponse, error) {
	req := &proto.MoveRequest{
		TargetPosition: &proto.Position{
			X: x,
			Y: y,
			Z: z,
		},
		Timestamp: time.Now().Unix(),
	}
	resp := &proto.MoveResponse{}

	err := gc.sendRequest("room.room.Move", req, resp)
	if err != nil {
		return nil, fmt.Errorf("move player failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) Attack(targetID string) (*proto.AttackResponse, error) {
	req := &proto.AttackRequest{
		TargetId: targetID,
		SkillId:  0,
		TargetPosition: &proto.Position{
			X: 0,
			Y: 0,
			Z: 0,
		},
		Timestamp: time.Now().Unix(),
	}
	resp := &proto.AttackResponse{}

	err := gc.sendRequest("room.room.Attack", req, resp)
	if err != nil {
		return nil, fmt.Errorf("attack failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) UseSkill(skillID int32) (*proto.UseSkillResponse, error) {
	req := &proto.UseSkillRequest{
		SkillId: skillID,
		TargetPosition: &proto.Position{
			X: 0,
			Y: 0,
			Z: 0,
		},
		Timestamp: time.Now().Unix(),
	}
	resp := &proto.UseSkillResponse{}

	err := gc.sendRequest("room.room.UseSkill", req, resp)
	if err != nil {
		return nil, fmt.Errorf("use skill failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) GetRoomInfo() (*proto.GetRoomInfoResponse, error) {
	req := &proto.GetRoomInfoRequest{
		RoomId: gc.roomID,
	}
	resp := &proto.GetRoomInfoResponse{}

	err := gc.sendRequest("room.room.GetRoomInfo", req, resp)
	if err != nil {
		return nil, fmt.Errorf("get room info failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) GetGameState() (*proto.GameStateUpdate, error) {
	req := &proto.GetRoomInfoRequest{
		RoomId: gc.roomID,
	}
	resp := &proto.GetRoomInfoResponse{}

	err := gc.sendRequest("room.room.GetRoomInfo", req, resp)
	if err != nil {
		return nil, fmt.Errorf("get game state failed: %w", err)
	}

	players := make([]*proto.PlayerState, 0, len(resp.RoomInfo.Players))
	for _, player := range resp.RoomInfo.Players {
		players = append(players, &proto.PlayerState{
			PlayerId: player.PlayerId,
			Health:   player.Health,
			MaxHealth: player.MaxHealth,
			Position:  player.Position,
			Score:    player.Score,
			Skills:   []*proto.ActiveSkill{},
		})
	}

	gameState := &proto.GameStateUpdate{
		RoomId:           resp.RoomInfo.RoomId,
		CurrentWave:      resp.RoomInfo.CurrentWave,
		EnemiesRemaining: 0,
		Score:            resp.RoomInfo.Score,
		Enemies:          []*proto.EnemyState{},
		Players:          players,
		ServerTime:       time.Now().Unix(),
	}

	return gameState, nil
}

func (gc *GameClient) GetRanking(limit int32) (*proto.RoomListResponse, error) {
	req := &proto.RoomListRequest{
		Limit:  limit,
		Offset: 0,
	}
	resp := &proto.RoomListResponse{}

	err := gc.sendRequest("room.room.RoomList", req, resp)
	if err != nil {
		return nil, fmt.Errorf("get ranking failed: %w", err)
	}

	return resp, nil
}

func (gc *GameClient) GetPlayerRanking() (*proto.RoomListResponse, error) {
	return gc.GetRanking(10)
}

func (gc *GameClient) Close() error {
	if gc.IsConnected() {
		return gc.Disconnect()
	}
	return nil
}

func (gc *GameClient) sendRequest(route string, req interface{}, resp interface{}) error {
	reqData, err := gc.serializer.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	msgID, err := gc.client.SendRequest(route, reqData)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}

	timeout := time.After(5 * time.Second)
	select {
	case msg := <-gc.client.MsgChannel():
		if msg.ID != msgID {
			return fmt.Errorf("unexpected message ID")
		}

		err = gc.serializer.Unmarshal(msg.Data, resp)
		if err != nil {
			return fmt.Errorf("unmarshal response failed: %w", err)
		}

		return nil
	case <-timeout:
		return fmt.Errorf("request timeout")
	case <-gc.ctx.Done():
		return fmt.Errorf("client closed")
	}
}
