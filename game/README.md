# War Inc Rising Game Server

## 项目状态

### ✅ 第一阶段：基础框架（已完成）
- [x] Protobuf 协议定义和代码生成
- [x] Connector Server 基础实现
- [x] 心跳机制实现（15秒间隔，45秒超时）
- [x] 基础的连接管理
- [x] 断线重连支持

### ✅ 第二阶段：房间系统（已完成）
- [x] Room Server 完整架构
- [x] 房间创建/销毁功能
- [x] 玩家加入/离开功能
- [x] Actor 模式房间管理
- [x] 基础游戏循环（Tick 驱动，100ms间隔）

### 🚧 第三阶段：游戏逻辑（待实现）
- [ ] 玩家移动系统
- [ ] 战斗系统实现
- [ ] 技能系统实现
- [ ] 敌人AI系统
- [ ] 波次管理

### 🚧 第四阶段：数据持久化（待实现）
- [ ] Redis 缓存集成
- [ ] MySQL 数据库集成
- [ ] 异步持久化机制
- [ ] 缓存保护机制

### 🚧 第五阶段：高级功能（待实现）
- [ ] 排行榜系统
- [ ] 匹配系统
- [ ] 断线重连优化
- [ ] 状态同步优化

### 🚧 第六阶段：监控和运维（待实现）
- [ ] 监控指标收集
- [ ] 分布式追踪集成
- [ ] 健康检查实现
- [ ] 日志系统完善

## 快速开始

### 前置要求

1. **Go 1.25+**
2. **Docker Desktop**
3. **protoc**（可选，用于重新生成 proto 文件）

### 安装依赖

```bash
# 安装 Go protoc 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

### 启动服务

```bash
# 1. 启动基础设施
cd game
docker-compose up -d

# 2. 启动 Connector Server
cd game/cmd/connector
./connector.exe --port 3250

# 3. 启动 Room Server
cd game/cmd/room
./room.exe --port 3260
```

### 测试服务

```bash
# 检查健康状态
curl http://localhost:8080/health

# 使用 WebSocket 客户端测试
wscat -c "ws://localhost:3250"
```

## 项目结构

```
game/
├── cmd/
│   ├── connector/
│   │   ├── main.go              # Connector 服务器入口
│   │   └── connector.exe       # 编译后的可执行文件
│   └── room/
│       ├── main.go              # Room 服务器入口
│       ├── room_handler.go      # RPC 处理器
│       └── room.exe             # 编译后的可执行文件
├── internal/
│   ├── connector/
│   │   └── connector.go         # 连接处理和心跳
│   ├── session/
│   │   └── session_manager.go  # 会话状态管理
│   ├── room/
│   │   ├── room_actor.go       # Actor 模式房间实现
│   │   ├── room_events.go      # 房间事件定义
│   │   └── room_manager.go     # 房间管理器
│   └── proto/
│       ├── common.pb.go        # 通用消息
│       ├── connection.pb.go    # 连接协议
│       ├── room.pb.go          # 房间协议
│       ├── game.pb.go          # 游戏协议
│       └── ranking.pb.go       # 排行榜协议
├── proto/
│   ├── common.proto
│   ├── connection.proto
│   ├── room.proto
│   ├── game.proto
│   └── ranking.proto
├── docker-compose.yml
├── README.md
├── TESTING.md
└── PHASE2_COMPLETE.md
```

## 核心特性

### 1. Actor 模式房间管理

每个房间都是一个独立的 Actor，通过事件通道串行处理所有操作，无需锁：

```go
type RoomActor struct {
    roomID    string
    eventChan chan RoomEvent  // 事件通道
    state     *RoomState      // 房间状态
    gameLoop  *GameLoop       // 游戏循环
    shutdown  chan struct{}    // 关闭信号
}

// 发送事件到房间
func (ra *RoomActor) SendEvent(event RoomEvent) error {
    select {
    case ra.eventChan <- event:
        return nil
    case <-ra.shutdown:
        return fmt.Errorf("room is shutting down")
    }
}
```

### 2. Tick 驱动游戏循环

游戏循环使用 `time.Ticker` 每 100ms 更新一次游戏状态：

```go
type GameLoop struct {
    ticker    *time.Ticker      // 100ms 间隔
    shutdown  chan struct{}     // 关闭信号
    state     *RoomState        // 房间状态
    currentTick uint64          // 当前 Tick
}

func (gl *GameLoop) processTick() {
    gl.currentTick++
    gl.state.Tick = gl.currentTick
    gl.state.GameTime += gl.tickRate
}
```

### 3. 心跳机制

Connector 实现了完整的心跳检测：

- **心跳间隔**: 15秒
- **超时阈值**: 45秒
- **自动清理**: 检测到死连接自动清理

### 4. 会话保留

支持断线重连，会话状态保留 5 分钟：

```go
type SessionState struct {
    SessionID  int64
    UserID     string
    RoomID     string
    GameState  []byte
    LastActive time.Time
    CreatedAt  time.Time
}
```

## API 文档

### 房间管理

#### 创建房间
```json
{
  "route": "room.create",
  "data": {
    "room_type": "normal",
    "max_players": 4,
    "difficulty": 1
  }
}
```

#### 加入房间
```json
{
  "route": "room.join",
  "data": {
    "room_id": "<room_id>"
  }
}
```

#### 离开房间
```json
{
  "route": "room.leave",
  "data": {
    "room_id": "<room_id>"
  }
}
```

#### 获取房间信息
```json
{
  "route": "room.get_info",
  "data": {
    "room_id": "<room_id>"
  }
}
```

#### 列出房间
```json
{
  "route": "room.list",
  "data": {
    "limit": 20,
    "offset": 0
  }
}
```

### 游戏操作

#### 玩家移动
```json
{
  "route": "room.move",
  "data": {
    "target_position": {
      "x": 10.0,
      "y": 20.0,
      "z": 0.0
    }
  }
}
```

#### 攻击
```json
{
  "route": "room.attack",
  "data": {
    "target_id": "<target_id>",
    "skill_id": 1,
    "target_position": {
      "x": 10.0,
      "y": 20.0,
      "z": 0.0
    }
  }
}
```

#### 使用技能
```json
{
  "route": "room.use_skill",
  "data": {
    "skill_id": 1,
    "target_position": {
      "x": 10.0,
      "y": 20.0,
      "z": 0.0
    }
  }
}
```

## 性能指标

- **Tick 间隔**: 100ms（10 ticks/秒）
- **心跳间隔**: 15秒
- **心跳超时**: 45秒
- **会话 TTL**: 5分钟
- **房间 TTL**: 2小时
- **最大玩家数**: 10
- **事件通道容量**: 1000

## 开发指南

### 重新生成 Proto 文件

```bash
cd game
protoc --proto_path=proto --go_out=internal/proto --go_opt=paths=source_relative proto/*.proto
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/room
```

### 代码检查

```bash
# 格式化代码
go fmt ./...

# 代码检查
go vet ./...

# 运行 linter
golangci-lint run
```

## 故障排查

### 端口被占用

```bash
# Windows
netstat -ano | findstr :3250
taskkill /PID <pid> /F
```

### NATS 连接失败

```bash
# 检查 NATS 状态
curl http://localhost:8222/varz

# 重启 NATS
docker-compose restart nats
```

### 房间创建失败

```bash
# 检查 Room Server 日志
# 查看是否有错误信息

# 检查 NATS 连接
docker-compose logs nats
```

## 下一步

1. **实现第三阶段**：游戏逻辑
   - 玩家移动系统
   - 战斗系统
   - 技能系统
   - 敌人AI
   - 波次管理

2. **添加测试**
   - 单元测试
   - 集成测试
   - 性能测试

3. **优化性能**
   - 状态同步优化
   - 带宽优化
   - 内存优化

## 技术栈

- **语言**: Go 1.25+
- **框架**: Pitaya v3
- **通信**: Protobuf + WebSocket
- **消息队列**: NATS
- **服务发现**: etcd
- **缓存**: Redis
- **数据库**: MySQL

## 架构设计

### 前后端分离

- **Connector Server**: 处理连接、路由、心跳
- **Room Server**: 处理游戏逻辑、房间管理

### 通信流程

```
Client <--WebSocket--> Connector <--NATS RPC--> Room
                              |
                              +---> Redis (cache)
                              |
                              +---> MySQL (persistence)
```

### 并发模型

- **Actor 模式**: 每个房间一个 Actor
- **事件驱动**: 所有操作通过事件通道
- **无锁设计**: 串行处理，避免竞态条件

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题，请提交 Issue 或联系维护者。
