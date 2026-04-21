# 游戏服务器测试指南

## 快速测试

### 1. 启动所有服务

```bash
# 终端 1: 启动基础设施
cd game
docker-compose up -d

# 终端 2: 启动 Connector Server
cd game/cmd/connector
./connector.exe --port 3250

# 终端 3: 启动 Room Server
cd game/cmd/room
./room.exe --port 3260
```

### 2. 验证服务运行

```bash
# 检查 Connector 健康状态
curl http://localhost:8080/health

# 检查基础设施状态
docker-compose ps
```

### 3. 测试房间功能

#### 创建房间
```bash
# 使用 WebSocket 客户端连接到 ws://localhost:3250
# 发送消息:
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
```bash
{
  "route": "room.join",
  "data": {
    "room_id": "<从创建房间响应中获取>"
  }
}
```

#### 获取房间信息
```bash
{
  "route": "room.get_info",
  "data": {
    "room_id": "<房间ID>"
  }
}
```

#### 列出所有房间
```bash
{
  "route": "room.list",
  "data": {
    "limit": 20,
    "offset": 0
  }
}
```

#### 玩家移动
```bash
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

## WebSocket 测试客户端

### 使用 wscat

```bash
# 安装 wscat
npm install -g wscat

# 连接到服务器
wscat -c "ws://localhost:3250"

# 发送消息
{"route": "room.create", "data": {"room_type": "normal", "max_players": 4, "difficulty": 1}}
```

### 使用 Python 测试脚本

```python
import asyncio
import websockets
import json

async def test_game_server():
    uri = "ws://localhost:3250"
    
    async with websockets.connect(uri) as websocket:
        # 创建房间
        create_room = {
            "route": "room.create",
            "data": {
                "room_type": "normal",
                "max_players": 4,
                "difficulty": 1
            }
        }
        await websocket.send(json.dumps(create_room))
        response = await websocket.recv()
        print(f"创建房间响应: {response}")
        
        # 解析房间ID
        result = json.loads(response)
        room_id = result.get("data", {}).get("room_id")
        
        if room_id:
            # 加入房间
            join_room = {
                "route": "room.join",
                "data": {
                    "room_id": room_id
                }
            }
            await websocket.send(json.dumps(join_room))
            response = await websocket.recv()
            print(f"加入房间响应: {response}")
            
            # 获取房间信息
            get_info = {
                "route": "room.get_info",
                "data": {
                    "room_id": room_id
                }
            }
            await websocket.send(json.dumps(get_info))
            response = await websocket.recv()
            print(f"房间信息: {response}")

if __name__ == "__main__":
    asyncio.run(test_game_server())
```

## 监控和调试

### 查看日志

```bash
# Connector 日志
# 在 Connector 终端查看输出

# Room 日志
# 在 Room 终端查看输出

# 基础设施日志
docker-compose logs -f nats
docker-compose logs -f redis
```

### 检查房间状态

```bash
# 列出所有房间
{"route": "room.list", "data": {"limit": 10, "offset": 0}}

# 检查特定房间
{"route": "room.get_info", "data": {"room_id": "<room_id>"}}
```

## 性能测试

### 创建多个房间

```python
import asyncio
import websockets
import json

async def create_multiple_rooms(count):
    uri = "ws://localhost:3250"
    
    tasks = []
    for i in range(count):
        async def create_room():
            async with websockets.connect(uri) as websocket:
                create_room = {
                    "route": "room.create",
                    "data": {
                        "room_type": "normal",
                        "max_players": 4,
                        "difficulty": 1
                    }
                }
                await websocket.send(json.dumps(create_room))
                response = await websocket.recv()
                print(f"房间 {i}: {response}")
        
        tasks.append(create_room())
    
    await asyncio.gather(*tasks)

if __name__ == "__main__":
    asyncio.run(create_multiple_rooms(10))
```

## 故障排查

### 连接失败

```bash
# 检查端口是否被占用
netstat -ano | findstr :3250
netstat -ano | findstr :3260

# 检查 NATS 是否运行
curl http://localhost:8222/varz
```

### 房间创建失败

```bash
# 检查 Room Server 日志
# 查看是否有错误信息

# 检查 NATS 连接
docker-compose logs nats
```

### 玩家加入失败

```bash
# 检查房间状态
{"route": "room.get_info", "data": {"room_id": "<room_id>"}}

# 检查房间是否已满
# 查看响应中的 current_players 和 max_players
```

## 清理和重置

```bash
# 停止所有服务
docker-compose down

# 清理所有数据
docker-compose down -v

# 重新启动
docker-compose up -d
```

## 下一步

完成测试后，可以继续：
1. 实现第三阶段：游戏逻辑
2. 添加更多测试用例
3. 性能优化
4. 添加监控指标
