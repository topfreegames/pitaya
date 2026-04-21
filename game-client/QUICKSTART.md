# War Inc Rising 链路测试快速开始

## 一键运行测试

### 方法 1: 使用 PowerShell 脚本（推荐）

```powershell
# 进入测试目录
cd D:\learningplace\first-game-server\game-client

# 运行测试（会自动启动服务器）
.\run-test.ps1
```

### 方法 2: 手动运行

#### 步骤 1: 启动服务器

```powershell
# 启动所有服务
.\start-servers.ps1
```

#### 步骤 2: 运行测试

```powershell
# 进入测试目录
cd D:\learningplace\first-game-server\game-client\cmd\test

# 运行基础测试
go run main.go

# 运行带报告的测试
go run main_with_report.go
```

#### 步骤 3: 停止服务器

```powershell
# 停止所有服务
.\stop-servers.ps1
```

## 测试说明

### 测试场景

1. **基础连接测试**
   - 测试客户端连接到服务器
   - 测试玩家绑定到会话
   - 测试心跳机制

2. **创建房间测试**
   - 测试创建房间功能
   - 测试获取房间信息

3. **加入房间测试**
   - 测试多玩家加入房间
   - 测试房间状态同步

4. **游戏流程测试**
   - 测试游戏开始
   - 测试玩家移动
   - 测试攻击功能
   - 测试技能使用
   - 测试游戏状态获取

5. **完整游戏流程测试**
   - 测试完整的游戏流程
   - 测试多玩家协作
   - 测试排行榜功能

### 预期结果

所有测试应该通过，输出类似：

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

## 故障排查

### 问题 1: 连接失败

**症状**: `连接服务器失败`

**解决方案**:
1. 检查 Connector Server 是否运行
2. 检查端口 3250 是否被占用
3. 检查防火墙设置

### 问题 2: RPC 调用失败

**症状**: `create room failed` 或 `join room failed`

**解决方案**:
1. 检查 Room Server 是否运行
2. 检查 NATS 连接是否正常
3. 检查 Protobuf 定义是否一致

### 问题 3: 数据库错误

**症状**: `database connection failed`

**解决方案**:
1. 检查 MySQL 是否运行
2. 检查数据库是否初始化
3. 检查连接配置

### 问题 4: 测试超时

**症状**: 测试长时间无响应

**解决方案**:
1. 检查服务器日志
2. 增加超时时间
3. 重启服务器

## 查看测试报告

运行带报告的测试后，会生成 `test-report.json` 文件：

```powershell
# 查看报告
Get-Content test-report.json | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

## 下一步

完成链路测试后，可以进行：

1. **压力测试** - 使用 k6 进行性能测试
2. **集成测试** - 测试更多复杂场景
3. **监控测试** - 验证监控和告警功能

## 联系支持

如果遇到问题，请查看：
- 项目 README.md
- 服务器日志
- 测试报告
