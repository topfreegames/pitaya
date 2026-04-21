# War Inc Rising 链路测试

## 概述

这个项目包含 War Inc Rising 游戏服务器的链路测试，用于验证整个游戏流程的正确性。

## 测试场景

1. **基础连接测试** - 测试客户端连接、心跳等基础功能
2. **创建房间测试** - 测试房间创建功能
3. **加入房间测试** - 测试多玩家加入房间功能
4. **游戏流程测试** - 测试游戏核心功能（移动、攻击、技能）
5. **完整游戏流程测试** - 测试完整的游戏流程

## 前置条件

### 1. 启动基础设施服务

```bash
cd D:\learningplace\first-game-server
docker compose up -d
```

这将启动：
- Redis (localhost:6380)
- MySQL (localhost:3307)
- NATS (localhost:4222)
- etcd (localhost:2379)

### 2. 启动游戏服务器

#### 启动 Connector Server

```bash
cd D:\learningplace\first-game-server\game\cmd\connector
.\connector.exe
```

#### 启动 Room Server

```bash
cd D:\learningplace\first-game-server\game\cmd\room
.\room.exe
```

## 运行链路测试

### 1. 安装依赖

```bash
cd D:\learningplace\first-game-server\game-client
go mod tidy
```

### 2. 运行测试

```bash
cd D:\learningplace\first-game-server\game-client\cmd\test
go run main.go
```

## 测试输出

测试运行时会输出详细的测试步骤和结果：

```
=== War Inc Rising 链路测试 ===

--- 测试1: 基础连接测试 ---
✓ 成功连接到服务器
✓ 成功连接到游戏
✓ 心跳发送成功
✅ 测试通过

--- 测试2: 创建房间测试 ---
✓ 成功创建房间: xxx-xxx-xxx
✓ 房间信息: 玩家数=1, 状态=waiting
✅ 测试通过

...

=== 链路测试完成 ===
```

## 测试验证

链路测试会验证以下功能：

### 连接功能
- ✅ 客户端连接到服务器
- ✅ 玩家绑定到会话
- ✅ 心跳机制正常工作
- ✅ 断开连接正常处理

### 房间功能
- ✅ 创建房间
- ✅ 加入房间
- ✅ 离开房间
- ✅ 获取房间信息
- ✅ 房间状态管理

### 游戏功能
- ✅ 开始游戏
- ✅ 玩家移动
- ✅ 攻击敌人
- ✅ 使用技能
- ✅ 获取游戏状态
- ✅ 波次管理

### 排行榜功能
- ✅ 获取排行榜
- ✅ 获取玩家排名

## 故障排查

### 连接失败

如果测试时出现连接失败，请检查：

1. 服务器是否正常启动
2. 端口是否正确（Connector: 3250）
3. 防火墙是否阻止连接

### RPC 调用失败

如果 RPC 调用失败，请检查：

1. Room Server 是否正常启动
2. NATS 连接是否正常
3. Protobuf 定义是否一致

### 数据库错误

如果出现数据库错误，请检查：

1. MySQL 是否正常运行
2. 数据库是否正确初始化
3. 连接配置是否正确

## 下一步

完成链路测试后，可以进行：

1. **压力测试** - 使用 k6 进行性能测试
2. **集成测试** - 测试更多复杂场景
3. **监控测试** - 验证监控和告警功能
