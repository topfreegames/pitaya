# War Inc Rising Style Game Server Architecture Design

**项目类型：** 策略塔防类多人在线游戏
**游戏模式：** 混合模式（单人闯关 + 多人合作 + 排行榜）
**预期规模：** 大规模（10000+ 同时在线）
**实时性要求：** 高实时性（50-100ms 延迟）
**架构方案：** 经典三层架构（Connector + Room + Worker）

**创建日期：** 2026-04-20
**版本：** 1.0

---

## 1. 整体架构

### 1.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端层                              │
│  (Unity/Unreal/Web/Mobile)                                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Frontend Layer (Connector)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ Connector 1  │  │ Connector 2  │  │ Connector N  │        │
│  │  (WebSocket) │  │  (WebSocket) │  │  (WebSocket) │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│         职─────────────────────────────────────────┐           │
│         │   负载均衡器 (Nginx/HAProxy)          │           │
│         └─────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Backend Layer (Room)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  Room 1      │  │  Room 2      │  │  Room N      │        │
│  │  (Actor模式)  │  │  (Actor模式)  │  │  (Actor模式)  │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│         职─────────────────────────────────────────┐           │
│         │   服务发现                          │           │
│         │   (etcd)                            │           │
│         └─────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Data Layer                                │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │ Redis 集群   │  │ MySQL 集群   │                        │
│  │  (缓存层)     │  │  (持久层)     │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 服务器类型

#### 1.2.1 Connector Server（前端服务器）

**职责：**
- 处理客户端连接（WebSocket）
- 实现心跳机制（15秒间隔，45秒超时）
- 路由请求到后端服务器
- 处理断线重连
- 实现认证和授权

**核心接口：**
```go
- Connect(ctx, req) (*ConnectResponse, error)
- Heartbeat(ctx, req) (*HeartbeatResponse, error)
- RouteToBackend(ctx, req) (*Response, error)
- HandleReconnect(ctx, req) (*ReconnectResponse, error)
```

**技术要点：**
- 使用 WebSocket 连接
- 实现心跳检测和超时清理
- 使用 Protobuf 通信
- 实现会话保留机制

#### 1.2.2 Room Server（后端服务器）

**职责：**
- 管理游戏房间
- 处理游戏逻辑
- 实现状态同步
- 处理战斗系统
- 管理玩家状态

**核心接口：**
```go
- CreateRoom(ctx, req) (*CreateRoomResponse, error)
- JoinRoom(ctx, req) (*JoinRoomResponse, error)
- LeaveRoom(ctx, req) (*LeaveRoomResponse, error)
- ProcessAction(ctx, req) (*ActionResponse, error)
- GetRoomState(ctx, req) (*RoomStateResponse, error)
```

**技术要点：**
- 使用 Actor 模式管理房间
- 使用 Tick 驱动游戏循环（100ms）
- 使用 Redis 缓存房间状态
- 实现状态同步和预测

#### 1.2.3 Worker Server（后台服务器）

**职责：**
- 处理匹配系统
- 更新排行榜
- 处理数据持久化
- 处理定时任务

**核心接口：**
```go
- FindMatch(ctx, req) (*MatchResponse, error)
- UpdateRanking(ctx, req) (*RankingResponse, error)
- PersistData(ctx, req) (*PersistResponse, error)
- RunScheduledTask(ctx, req) (*TaskResponse, error)
```

**技术要点：**
- 使用消息队列（NATS）异步处理
- 实现批量数据持久化
- 使用 Redis SortedSet 管理排行榜
- 实现定时任务调度

---

## 2. 数据存储设计

### 2.1 Redis 数据结构

#### 2.1.1 玩家数据

**玩家基本信息：**
```
Key: player:{player_id}:info
Type: Hash
Fields:
  - name: 玩家名称
  - level: 等级
  - exp: 经验值
  - coins: 金币
  - gems: 钻石
  - created_at: 创建时间
  - updated_at: 更新时间
TTL: 1小时
```

**玩家实时状态：**
```
Key: player:{player_id}:state
Type: Hash
Fields:
  - room_id: 当前房间ID
  - status: 状态（idle, in_room, in_combat）
  - health: 生命值
  - position_x: 位置X
  - position_y: 位置Y
  - last_action: 最后操作时间
TTL: 5分钟
```

**玩家战斗数据：**
```
Key: player:{player_id}:combat
Type: Hash
Fields:
  - current_hp: 当前生命值
  - max_hp: 最大生命值
  - attack_power: 攻击力
  - defense_power: 防御力
  - speed: 移动速度
  - skills: 技能列表（JSON）
TTL: 10分钟
```

#### 2.1.2 房间数据

**房间基本信息：**
```
Key: room:{room_id}:info
Type: Hash
Fields:
  - room_id: 房间ID
  - room_type: 房间类型
  - max_players: 最大玩家数
  - current_players: 当前玩家数
  - status: 状态（waiting, playing, finished）
  - created_at: 创建时间
  - started_at: 开始时间
  - finished_at: 结束时间
TTL: 2小时
```

**房间玩家列表：**
```
Key: room:{room_id}:players
Type: Set
Members: player_id1, player_id2, ...
TTL: 2小时
```

**房间游戏状态：**
```
Key: room:{room_id}:state
Type: Hash
Fields:
  - current_wave: 当前波次
  - enemies_remaining: 剩余敌人
  - score: 当前分数
  - game_time: 游戏时间
  - difficulty: 难度等级
TTL: 2小时
```

#### 2.1.3 排行榜数据

**玩家积分排行榜：**
```
Key: ranking:players:score
Type: SortedSet
Members: score -> player_id
TTL: 1小时
```

**玩家波次排行榜：**
```
Key: ranking:players:waves
Type: SortedSet
Members: waves -> player_id
TTL: 1小时
```

**房间排行榜：**
```
Key: ranking:rooms:score
Type: SortedSet
Members: score -> room_id
TTL: 1小时
```

### 2.2 MySQL 数据结构

#### 2.2.1 玩家表

```sql
CREATE TABLE players (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    level INT DEFAULT 1,
    exp BIGINT DEFAULT 0,
    coins BIGINT DEFAULT 0,
    gems BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_level (level),
    INDEX idx_coins (coins)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 2.2.2 房间表

```sql
CREATE TABLE rooms (
    id VARCHAR(64) PRIMARY KEY,
    room_type VARCHAR(32) NOT NULL,
    max_players INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    score BIGINT DEFAULT 0,
    waves INT DEFAULT 0,
    difficulty INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_score (score)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 2.2.3 房间玩家关联表

```sql
CREATE TABLE room_players (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    room_id VARCHAR(64) NOT NULL,
    player_id VARCHAR(64) NOT NULL,
    score BIGINT DEFAULT 0,
    waves_survived INT DEFAULT 0,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP NULL,
    INDEX idx_room_id (room_id),
    INDEX idx_player_id (player_id),
    UNIQUE KEY uk_room_player (room_id, player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 2.2.4 游戏记录表

```sql
CREATE TABLE game_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    player_id VARCHAR(64) NOT NULL,
    room_id VARCHAR(64) NOT NULL,
    score BIGINT DEFAULT 0,
    waves_survived INT DEFAULT 0,
    enemies_defeated INT DEFAULT 0,
    duration_seconds INT DEFAULT 0,
    played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_player_id (player_id),
    INDEX idx_played_at (played_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2.3 数据访问模式

**缓存优先模式：**
```go
// 读取玩家数据
func GetPlayerInfo(playerID string) (*PlayerInfo, error) {
    // 1. 先从 Redis 读取
    cached, err := redis.GetPlayerInfoFromCache(playerID)
    if err == nil && cached != nil {
        return cached, nil
    }

    // 2. 缓存未命中，从 MySQL 读取
    player, err := mysql.GetPlayerInfo(playerID)
    if err != nil {
        return nil, err
    }

    // 3. 写入 Redis 缓存
    redis.SetPlayerInfoToCache(player, player)

    return player, nil
}

// 更新玩家数据
func UpdatePlayerCoins(playerID string, delta int64) error {
    // 1. 立即更新 Redis
    redis.UpdatePlayerCoins(playerID, delta)

    // 2. 触发异步持久化事件
    event := PersistenceEvent{
        Type: "player_coins_update",
        Data: map[string]interface{}{
            "player_id": playerID,
            "delta":    delta,
        },
    }
    nats.Publish("game.persistence", event)

    return nil
}
```

---

## 3. 通信协议设计

### 3.1 Protobuf 协议定义

#### 3.1.1 通用消息

```protobuf
syntax = "proto3";

package game;

option go_package = "github.com/yourproject/proto/game";

// 通用响应
message Response {
    int32 code = 1;
    string message = 2;
    google.protobuf.Any data = 3;
}

// 错误码
enum ErrorCode {
    SUCCESS = 0;
    INVALID_REQUEST = 1;
    AUTH_FAILED = 2;
    ROOM_FULL = 3;
    PLAYER_NOT_FOUND = 4;
    ROOM_NOT_FOUND = 5;
    GAME_ALREADY_STARTED = 6;
    INSUFFICIENT_RESOURCES = 7;
}
```

#### 3.1.2 连接相关

```protobuf
// 连接请求
message ConnectRequest {
    string player_id = 1;
    string auth_token = 2;
    string client_version = 3;
    string device_id = 4;
}

// 连接响应
message ConnectResponse {
    string session_id = 1;
    int64 server_time = 2;
    repeated string available_rooms = 3;
}

// 心跳请求
message HeartbeatRequest {
    int64 timestamp = 1;
}

// 心跳响应
message HeartbeatResponse {
    int64 server_time = 1;
    HeartbeatStatus status = 2;
}

enum HeartbeatStatus {
    OK = 0;
    WARNING = 1;
    CRITICAL = 2;
}
```

#### 3.1.3 房间相关

```protobuf
// 创建房间请求
message CreateRoomRequest {
    string room_type = 1;
    int32 max_players = 2;
    int32 difficulty = 3;
    string password = 4; // 可选
}

// 创建房间响应
message CreateRoomResponse {
    string room_id = 1;
    string room_token = 2;
    int32 max_players = 3;
}

// 加入房间请求
message JoinRoomRequest {
    string room_id = 1;
    string room_token = 2;
}

// 加入房间响应
message JoinRoomResponse {
    bool success = 1;
    string message = 2;
    RoomInfo room_info = 3;
}

// 房间信息
message RoomInfo {
    string room_id = 1;
    string room_type = 2;
    int32 max_players = 3;
    int32 current_players = 4;
    RoomStatus status = 5;
    int32 current_wave = 6;
    int32 score = 7;
    repeated PlayerInfo players = 8;
}

enum RoomStatus {
    WAITING = 0;
    PLAYING = 1;
    PAUSED = 2;
    FINISHED = 3;
}

// 玩家信息
message PlayerInfo {
    string player_id = 1;
    string name = 2;
    int32 level = 3;
    int32 health = 4;
    int32 max_health = 5;
    Position position = 6;
    int32 score = 7;
}

// 位置信息
message Position {
    float x = 1;
    float y = 2;
    float z = 3;
}
```

#### 3.1.4 游戏动作

```protobuf
// 玩家移动请求
message MoveRequest {
    Position target_position = 1;
    int64 timestamp = 2;
}

// 玩家移动响应
message MoveResponse {
    bool success = 1;
    Position new_position = 2;
    int32 movement_cost = 3;
}

// 攻击请求
message AttackRequest {
    string target_id = 1;
    int32 skill_id = 2;
    Position target_position = 3;
    int64 timestamp = 4;
}

// 攻击响应
message AttackResponse {
    bool success = 1;
    int32 damage_dealt = 2;
    bool target_killed = 3;
    int32 new_score = 4;
}

// 使用技能请求
message UseSkillRequest {
    int32 skill_id = 1;
    Position target_position = 2;
    int64 timestamp = 3;
}

// 使用技能响应
message UseSkillResponse {
    bool success = 1;
    int32 cooldown_remaining = 2;
    int32 mana_cost = 3;
}
```

#### 3.1.5 游戏状态同步

```protobuf
// 游戏状态更新（服务器推送）
message GameStateUpdate {
    string room_id = 1;
    int32 current_wave = 2;
    int32 enemies_remaining = 3;
    int32 score = 4;
    repeated EnemyState enemies = 5;
    repeated PlayerState players = 6;
}

// 敌人状态
message EnemyState {
    string enemy_id = 1;
    string enemy_type = 2;
    Position position = 3;
    int32 health = 4;
    int32 max_health = 5;
    EnemyStatus status = 6;
}

enum EnemyStatus {
    SPAWNING = 0;
    ALIVE = 1;
    DEAD = 2;
    DESPAWNING = 3;
}

// 玩家状态更新
message PlayerStateUpdate {
    string player_id = 1;
    int32 health = 2;
    int32 max_health = 3;
    Position position = 4;
    int32 score = 5;
    repeated ActiveSkill skills = 6;
}

// 激活技能
message ActiveSkill {
    int32 skill_id = 1;
    int32 cooldown_remaining = 2;
    bool is_active = 3;
}
```

#### 3.1.6 排行榜相关

```protobuf
// 获取排行榜请求
message GetRankingRequest {
    RankingType type = 1;
    int32 limit = 2;
    int32 offset = 3;
}

// 获取排行榜响应
message GetRankingResponse {
    repeated RankingEntry entries = 1;
    int32 total_count = 2;
    int32 player_rank = 3;
}

// 排行榜条目
message RankingEntry {
    int32 rank = 1;
    string player_id = 2;
    string player_name = 3;
    int64 score = 4;
    int32 level = 5;
}

enum RankingType {
    SCORE = 0;
    WAVES_SURVIVED = 1;
    ROOM_SCORE = 2;
}
```

### 3.2 消息路由设计

**路由字典：**
```go
// 前端服务器路由
var frontendRoutes = map[string]uint16{
    "connector.connect":           1,
    "connector.heartbeat":         2,
    "connector.reconnect":         3,
    "room.create":                 4,
    "room.join":                   5,
    "room.leave":                   6,
    "room.get_info":                7,
    "ranking.get":                   8,
}

// 后端服务器路由
var backendRoutes = map[string]uint16{
    "room.move":                    1,
    "room.attack":                  2,
    "room.use_skill":               3,
    "room.get_state":               4,
}
```

---

## 4. 游戏循环设计

### 4.1 Tick 驱动的游戏循环

**游戏循环架构：**
```go
type GameLoop struct {
    ticker      *time.Ticker
    shutdown    chan struct{}
    done        chan struct{}
    state       *GameState
    tickRate    time.Duration // 100ms (10 ticks/second)
    currentTick uint64
}

func NewGameLoop(tickRate time.Duration) *GameLoop {
    return &GameLoop{
        ticker:   time.NewTicker(tickRate),
        shutdown: make(chan struct{}),
        done:     make(chan struct{}),
        state:    &GameState{},
        tickRate: tickRate,
    }
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
            // 处理游戏 tick
            gl.processTick()

        case <-gl.shutdown:
            // 优雅关闭
            log.Printf("Game loop shutting down")
            return
        }
    }
}

func (gl *GameLoop) processTick() {
    gl.currentTick++

    // 1. 更新敌人状态
    gl.updateEnemies()

    // 2. 处理玩家动作
    gl.processPlayerActions()

    // 3. 碰撞检测
    gl.processCollisions()

    // 4. 更新游戏状态
    gl.updateGameState()

    // 5. 广播状态更新
    gl.broadcastStateUpdate()
}
```

### 4.2 房间 Actor 模式

**房间 Actor 实现：**
```go
type RoomActor struct {
    roomID     string
    eventChan  chan RoomEvent
    state      *RoomState
    gameLoop   *GameLoop
    shutdown   chan struct{}
    done       chan struct{}
}

type RoomEvent interface {
    Process(*RoomState) error
}

func NewRoomActor(roomID string) *RoomActor {
    actor := &RoomActor{
        roomID:    roomID,
        eventChan: make(chan RoomEvent, 1000),
        state:     &RoomState{},
        gameLoop:  NewGameLoop(100 * time.Millisecond),
        shutdown: make(chan struct{}),
        done:      make(chan struct{}),
    }

    actor.gameLoop.state = actor.state
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
            // 串行处理事件 - 无需锁
            if err := event.Process(ra.state); err != nil {
                log.Printf("Error processing event: %v", err)
            }

        case <-ra.shutdown:
            log.Printf("Room actor %s shutting down", ra.roomID)
            return
        }
    }
}
```

### 4.3 状态同步机制

**状态同步策略：**
```go
// 增量状态更新
type StateSyncManager struct {
    app         pitaya.Pitaya
    roomActors  map[string]*RoomActor
    syncRate    time.Duration // 每 100ms 同步一次
}

func (ssm *StateSyncManager) Start() {
    ticker := time.NewTicker(ssm.syncRate)
    go func() {
        for {
            select {
            case <-ticker.C:
                ssm.broadcastStateUpdates()
            }
        }
    }()
}

func (ssm *StateSyncManager) broadcastStateUpdates() {
    for roomID, actor := range ssm.roomActors {
        // 获取房间状态快照
        state := actor.GetStateSnapshot()

        // 转换为 Protobuf
        protoState := convertToProtobuf(state)

        // 广播给房间内所有玩家
        ssm.app.GroupBroadcast(
            context.Background(),
            "game",
            roomID,
            "on_state_update",
            protoState,
        )
    }
}
```

---

## 5. 性能优化设计

### 5.1 连接池优化

**Redis 连接池：**
```go
type RedisPool struct {
    pool *redis.Pool
}

func NewRedisPool(addr string) *RedisPool {
    return &RedisPool{
        pool: &redis.Pool{
            MaxIdle:     100,              // 最大空闲连接数
            MaxActive:   500,              // 最大活跃连接数
            IdleTimeout: 240 * time.Second, // 空闲连接超时
            Wait:        true,
            Dial: func() (redis.Conn, error) {
                return redis.Dial("tcp", addr)
            },
        },
    }
}
```

**MySQL 连接池：**
```go
type MySQLPool struct {
    db *sql.DB
}

func NewMySQLPool(dsn string) (*MySQLPool, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    // 配置连接池
    db.SetMaxOpenConns(100)        // 最大打开连接数
    db.SetMaxIdleConns(25)         // 最大空闲连接数
    db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期

    return &MySQLPool{db: db}, nil
}
```

### 5.2 批量操作优化

**批量 Redis 操作：**
```go
// 批量获取玩家信息
func BatchGetPlayerInfo(playerIDs []string) (map[string]*PlayerInfo, error) {
    pipe := redisClient.Pipeline()

    for _, playerID := range playerIDs {
        key := fmt.Sprintf("player:%s:info", playerID)
        pipe.HGetAll(context.Background(), key)
    }

    results, err := pipe.Exec(context.Background())
    if err != nil {
        return nil, err
    }

    playerInfos := make(map[string]*PlayerInfo)
    for i, playerID := range playerIDs {
        if i >= len(results) {
            break
        }

        cmd := results[i].(*redis.StringSliceCmd)
        values, err := cmd.Result()
        if err != nil {
            continue
        }

        if len(values) == 0 {
            continue
        }

        playerInfo := &PlayerInfo{}
        // 解析 Redis Hash 数据...
        playerInfos[playerID] = playerInfo
    }

    return playerInfos, nil
}
```

### 5.3 内存优化

**对象池模式：**
```go
var playerInfoPool = sync.Pool{
    New: func() interface{} {
        return &PlayerInfo{}
    },
}

func GetPlayerInfo() *PlayerInfo {
    return playerInfoPool.Get().(*PlayerInfo)
}

func PutPlayerInfo(info *PlayerInfo) {
    // 重置对象
    info.Reset()
    playerInfoPool.Put(info)
}
```

---

## 6. 监控和运维设计

### 6.1 监控指标设计

**核心监控指标：**
```go
type Metrics struct {
    // 连接指标
    ActiveConnections    prometheus.Gauge
    TotalConnections     prometheus.Counter
    ConnectionErrors     prometheus.Counter

    // 房间指标
    ActiveRooms          prometheus.Gauge
    TotalRoomsCreated    prometheus.Counter
    RoomCapacityUsage    prometheus.Gauge

    // 玩家指标
    ActivePlayers        prometheus.Gauge
    TotalPlayers         prometheus.Counter
    PlayerActions         prometheus.Counter

    // 性能指标
    RequestLatency        prometheus.Histogram
    ResponseTime          prometheus.Histogram
    CacheHitRate         prometheus.Gauge
    DatabaseQueryTime     prometheus.Histogram

    // 游戏指标
    GamesPlayed           prometheus.Counter
    WavesCompleted        prometheus.Counter
    EnemiesDefeated       prometheus.Counter
    AverageScore          prometheus.Gauge
}
```

### 6.2 分布式追踪

**OpenTelemetry 集成：**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func InitTracing(serviceName string) error {
    // 创建资源
    res, err := resource.New(
        context.Background(),
        resource.WithAttributes(
            semconv.SchemaVersion,
            serconv.CodeVersion,
            semconv.ServiceName(serviceName),
        ),
    )
    if err != nil {
        return err
    }

    // 创建导出器
    traceExporter, err := otlptracegrpc.New(
        context.Background(),
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithResource(res),
    )
    if err != nil {
        return err
    }

    // 创建追踪提供者
    traceProvider := traces.NewTracerProvider(
        traces.WithBatcher(
            trace.NewBatchExporter(traceExporter),
        ),
        traces.WithResource(res),
    )

    // 注册全局追踪提供者
    otel.SetTracerProvider(traceProvider)

    return nil
}
```

### 6.3 健康检查

**健康检查端点：**
```go
import (
    "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthChecker struct {
    db    *sql.DB
    redis *redis.Client
}

func NewHealthChecker(db *sql.DB, redis *redis.Client) *HealthChecker {
    return &HealthChecker{
        db:    db,
        redis: redis,
    }
}

func (hc *HealthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
    // 检查数据库连接
    if err := hc.db.PingContext(ctx); err != nil {
        return &grpc_health_v1.HealthCheckResponse{
            Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
        }, nil
    }

    // 检查 Redis 连接
    if err := hc.redis.Ping(ctx).Err(); err != nil {
        return &grpc_health_v1.HealthCheckResponse{
            Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
        }, nil
    }

    return &grpc_health_v1.HealthCheckResponse{
        Status: grpc_health_v1.HealthCheckResponse_SERVING,
    }, nil
}
```

---

## 7. 错误处理和容错设计

### 7.1 错误处理规范

**统一错误格式：**
```go
const (
    ErrCodeSuccess           = "SUCCESS"
    ErrCodeInvalidRequest    = "INVALID_REQUEST"
    ErrCodeAuthFailed        = "AUTH_FAILED"
    ErrCodeRoomFull          = "ROOM_FULL"
    ErrCodePlayerNotFound     = "PLAYER_NOT_FOUND"
    ErrCodeRoomNotFound      = "ROOM_NOT_FOUND"
    ErrCodeGameStarted       = "GAME_ALREADY_STARTED"
    ErrCodeInsufficientResources = "INSUFFICIENT_RESOURCES"
    ErrCodeNetworkError      = "NETWORK_ERROR"
    ErrCodeDatabaseError     = "DATABASE_ERROR"
    ErrCodeCacheError        = "CACHE_ERROR"
)
```

### 7.2 重试机制

**指数退避重试：**
```go
type RetryManager struct {
    maxRetries    int
    initialDelay time.Duration
    maxDelay     time.Duration
}

func (rm *RetryManager) RetryWithBackoff(ctx context.Context, operation func() error) error {
    var lastErr error
    delay := rm.initialDelay

    for attempt := 0; attempt <= rm.maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        lastErr = err

        // 检查是否应该重试
        if !rm.shouldRetry(err, attempt) {
            break
        }

        // 等待重试
        select {
        case <-time.After(delay):
            delay = time.Duration(float64(delay) * 1.5) // 指数退避
            if delay > rm.maxDelay {
                delay = rm.maxDelay
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return lastErr
}
```

### 7.3 熔断器模式

**熔断器实现：**
```go
import (
    "github.com/sony/gobreaker"
)

type CircuitBreakerManager struct {
    breakers map[string]*gobreaker.CircuitBreaker
}

func (cbm *CircuitBreakerManager) GetBreaker(serviceName string) *gobreaker.CircuitBreaker {
    if _, exists := cbm.breakers[serviceName]; !exists {
        cbm.breakers[serviceName] = gobreaker.NewCircuitBreaker(
            gobreaker.Settings{
                Name:        serviceName,
                MaxRequests: 1000,
                Interval:    10 * time.Second,
                Timeout:     5 * time.Second,
                ReadyToTrip: 5,
                State:       gobreaker.StateClosed,
            },
        )
    }

    return cbm.breakers[serviceName]
}
```

### 7.4 降级策略

**服务降级实现：**
```go
func (dm *DegradationManager) GetPlayerDataWithFallback(ctx context.Context, playerID string) (*PlayerData, error) {
    // 尝试从缓存获取
    data, err := dm.getPlayerDataFromCache(ctx, playerID)
    if err == nil && data != nil {
        return data, nil
    }

    // 缓存失败，尝试从数据库获取
    result, err := dm.circuitBreaker.ExecuteWithCircuitBreaker("database", func() (interface{}, error) {
        return dm.getPlayerDataFromDB(ctx, playerID)
    })

    if err != nil {
        // 数据库也失败，返回降级数据
        dm.metrics.CacheFallbackCount.Inc()
        return dm.getDegradedPlayerData(playerID), nil
    }

    return result.(*PlayerData), nil
}
```

### 7.5 限流保护

**限流器实现：**
```go
import (
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    limiters map[string]*rate.Limiter
}

func (rl *RateLimiter) GetLimiter(operation string) *rate.Limiter {
    if _, exists := rl.limiters[operation]; !exists {
        switch operation {
        case "create_room":
            rl.limiters[operation] = rate.NewLimiter(10, rate.Minute) // 每分钟10次
        case "join_room":
            rl.limiters[operation] = rate.NewLimiter(20, rate.Minute) // 每分钟20次
        case "player_action":
            rl.limiters[operation] = rate.NewLimiter(100, rate.Second) // 每秒100次
        default:
            rl.limiters[operation] = rate.NewLimiter(1000, rate.Minute) // 默认限制
        }
    }

    return rl.limiters[operation]
}
```

---

## 8. 部署架构

### 8.1 Docker Compose 配置

**服务编排：**
```yaml
version: '3.8'

services:
  # 前端服务器
  connector-1:
    build: .
    command: ["./connector", "--port", "3250", "--type", "connector", "--frontend=true"]
    ports:
      - "3250:3250"
    environment:
      - PITAYA_METRICS_PROMETHEUS_PORT=9090
      - PITAYA_NATS_ADDRESS=nats://nats:4222
      - PITAYA_ETCD_ADDRESS=etcd:2379
    depends_on:
      - nats
      - etcd
    networks:
      - game-network

  connector-2:
    build: .
    command: ["./connector", "--port", "3251", "--type", "connector", "--frontend=true"]
    ports:
      - "3251:3251"
    environment:
      - PITAYA_METRICS_PROMETHEUS_PORT=9091
      - PITAYA_NATS_ADDRESS=nats://nats:4222
      - PITAYA_ETCD_ADDRESS=etcd:2379
    depends_on:
      - nats
      - etcd
    networks:
      - game-network

  # 后端服务器
  room-1:
    build: .
    command: ["./room", "--port", "3260", "--type", "room", "--frontend=false"]
    environment:
      - PITAYA_METRICS_PROMETHEUS_PORT=9092
      - PITAYA_NATS_ADDRESS=nats://nats:4222
      - PITAYA_ETCD_ADDRESS=etcd:2379
    depends_on:
      - nats
      - etcd
    networks:
      - game-network

  room-2:
    build: .
    command: ["./room", "--port", "3261", "--type", "room", "--frontend=false"]
    environment:
      - PITAYA_METRICS_PROMETHEUS_PORT=9093
      - PITAYA_NATS_ADDRESS=nats://nats:4222
      - PITAYA_ETCD_ADDRESS=etcd:2379
    depends_on:
      - nats
      - etcd
    networks:
      - game-network

  # 后台服务器
  worker:
    build: .
    command: ["./worker", "--type", "worker", "--frontend=false"]
    environment:
      - PITAYA_NATS_ADDRESS=nats://nats:4222
      - PITAYA_ETCD_ADDRESS=etcd:2379
    depends_on:
      - nats
      - etcd
    networks:
      - game-network

  # 基础设施
  nats:
    image: nats:latest
    ports:
      - "4222:4222"
    networks:
      - game-network

  etcd:
    image: quay.io/coreos/etcd:v3.5.15
    command:
      - /usr/local/bin/etcd
      - --listen-client-urls=http://0.0.0.0:2379
      - --advertise-client-urls=http://0.0.0.0:2379
    ports:
      - "2379:2379"
    networks:
      - game-network

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
    networks:
      - game-network

  # 监控
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    networks:
      - game-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - ./grafana/dashboards:/var/lib/grafana
    networks:
      - game-network

networks:
  game-network:
    driver: bridge
```

### 8.2 自动扩缩容

**基于负载的自动扩缩容：**
```go
type AutoScaler struct {
    app              pitaya.Pitaya
    metrics          *Metrics
    scaleUpThreshold  float64 // CPU 使用率超过 80% 扩容
    scaleDownThreshold float64 // CPU 使用率低于 30% 缩容
    maxInstances     int
    minInstances     int
}

func (as *AutoScaler) MonitorAndScale() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            as.checkAndScale()
        }
    }
}
```

---

## 9. 开发计划

### 9.1 第一阶段：基础框架（1-2周）

**目标：** 搭建基础架构和核心功能

**任务：**
- [ ] 项目初始化和目录结构创建
- [ ] Protobuf 协议定义和代码生成
- [ ] Connector Server 基础实现
- [ ] 心跳机制实现
- [ ] 基础的连接管理

### 9.2 第二阶段：房间系统（2-3周）

**目标：** 实现房间管理和游戏逻辑

**任务：**
- [ ] Room Server 基础实现
- [ ] 房间创建/销毁功能
- [ ] 玩家加入/离开功能
- [ ] Actor 模式房间管理
- [ ] 基础游戏循环实现

### 9.3 第三阶段：游戏逻辑（3-4周）

**目标：** 实现核心游戏玩法

**任务：**
- [ ] 玩家移动系统
- [ ] 战斗系统实现
- [ ] 技能系统实现
- [ ] 敌人AI系统
- [ ] 波次管理

### 9.4 第四阶段：数据持久化（2-3周）

**目标：** 实现数据存储和缓存

**任务：**
- [ ] Redis 缓存集成
- [ ] MySQL 数据库集成
- [ ] 异步持久化机制
- [ ] 缓存保护机制
- [ ] 数据一致性保证

### 9.5 第五阶段：高级功能（2-3周）

**目标：** 实现高级功能

**任务：**
- [ ] 排行榜系统
- [ ] 匹配系统
- [ ] 断线重连
- [ ] 状态同步优化
- ] 性能优化

### 9.6 第六阶段：监控和运维（1-2周）

**目标：** 完善监控和运维体系

**任务：**
- [ ] 监控指标收集
- [ ] 分布式追踪集成
- [ ] 健康检查实现
- [ ] 日志系统完善
- ] 部署脚本编写

---

## 10. 技术约束和最佳实践

### 10.1 必须遵循的约束

**架构约束：**
- ✅ 严格前后端分离 - Frontend 只做连接和路由，Backend 处理游戏逻辑
- ✅ 所有通信必须使用 Protobuf - 严禁使用 JSON
- ✅ 高频数据必须优先使用 Redis - 严禁直接访问 MySQL
- ✅ 数据修改必须异步持久化 - 严禁在游戏逻辑中同步等待数据库 I/O

**并发约束：**
- ✅ 房间管理必须使用 Actor 模式 - 避免锁竞争
- ✅ 共享状态必须使用读写锁保护 - 防止竞态条件
- ✅ 游戏循环必须使用 Tick 驱动 - 使用 time.Ticker 而非 time.Sleep
- ✅ 必须正确实现组件生命周期 - 防止资源泄漏

**网络约束：**
- ✅ 必须实现心跳机制 - 15秒间隔，45秒超时
- ✅ 必须实现断线重连 - 保留会话状态
- ✅ 必须使用状态预测和补偿 - 减少网络延迟影响
- ✅ 必须实现增量更新 - 减少带宽占用

**数据约束：**
- ✅ 必须使用 singleflight 防止缓存穿透
- ✅ 必须实现批量操作优化 - 提高性能
- ✅ 必须实现连接池管理 - 防止连接泄漏
- ✅ 必须设置合理的 TTL - 防止内存泄漏

### 10.2 性能目标

**性能指标：**
- **延迟：** 50-100ms（P99）
- **并发：** 10000+ 同时在线
- **吞吐：** 10000+ 请求/秒
- **缓存命中率：** >80%
- **数据库连接池：** 最大 100 个连接
- **Redis 连接池：** 最大 500 个连接

### 10.3 可靠性目标

**可靠性指标：**
- **可用性：** 99.9%
- **数据一致性：** 最终一致性
- **故障恢复：** 自动故障转移
- **数据持久化：** 异步批量写入，最多 5 秒延迟

---

## 11. 风险和挑战

### 11.1 技术风险

**高并发风险：**
- **风险：** 10000+ 同时在线可能导致系统过载
- **缓解：** 实现自动扩缩容、限流保护、熔断器模式

**网络延迟风险：**
- **风险：** 网络波动可能影响游戏体验
- **缓解：** 实现状态预测、增量更新、重连机制

**数据一致性风险：**
- **风险：** 异步持久化可能导致数据不一致
- **缓解：** 实现事件溯源、版本控制、冲突解决

### 11.2 运营风险

**资源消耗风险：**
- **风险：** 大规模部署可能消耗大量资源
- **缓解：** 实现资源监控、自动扩缩容、成本优化

**监控复杂度风险：**
- **风险：** 大规模系统监控复杂度高
- **缓解：** 完善监控体系、自动化告警、日志聚合

---

## 12. 成功标准

### 12.1 功能成功标准

- [ ] 支持 10000+ 同时在线玩家
- [ ] 支持单人闯关模式
- [ ] 支持 2-4 人合作模式
- [ ] 实现实时排行榜
- [ ] 实现断线重连
- [ ] 实现心跳检测和超时清理

### 12.2 性能成功标准

- [ ] 平均延迟 < 100ms
- [ ] P99 延迟 < 200ms
- [ ] 缓存命中率 > 80%
- [ ] 数据库操作延迟 < 50ms
- [ ] 内存使用 < 80%

### 12.3 可靠性成功标准

- [ ] 系统可用性 > 99.9%
- [ ] 故障恢复时间 < 1 分钟
- [ ] 数据持久化延迟 < 5 秒
- [ ] 监控覆盖率 > 95%

---

## 13. 后续步骤

设计文档已完成，现在可以：

1. **审查设计文档** - 检查是否有遗漏或需要调整的部分
2. **开始实现计划** - 将设计转化为具体的实现步骤
3. **开始编码实现** - 按照设计文档开始实现

请审查这个设计文档，确认是否需要调整任何部分。
