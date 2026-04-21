在本地运行 Pitaya 项目和演示示例
📋 环境准备
必需环境
- Go 1.25+ - 主要开发语言
- Docker - 用于运行依赖服务（etcd、nats、redis）
可选环境
- Docker Compose - 用于管理 Docker 服务
- Protobuf 编译器 - 如果需要修改 protobuf 文件
🚀 快速开始
1. 克隆并设置项目
# 进入项目目录
cd D:\learningplace\first-game-server
# 安装 Go 依赖
make setup
# 安装 CI 工具（可选，用于测试）
make setup-ci
2. 启动依赖服务
# 启动测试所需的 Docker 服务（etcd、nats、redis）
cd examples/testing
docker compose up -d
# 检查服务状态
docker compose ps
# 查看日志（如果需要）
docker compose logs -f
🎮 运行演示示例
示例 1: 聊天室示例（最简单）
这是最简单的示例，使用 WebSocket 实现聊天室功能。
# 运行聊天室服务器
cd examples/demo/chat
go run main.go
访问方式：
- 打开浏览器访问：http://localhost:3251/web/
- 服务器监听端口：3250（WebSocket）
- 静态文件端口：3251（HTTP）
功能特点：
- 实时聊天室
- 用户加入/离开通知
- 消息广播
- 群组管理
示例 2: 集群示例（推荐）
这个示例展示了 Pitaya 的集群功能，需要运行多个服务器。
步骤 1: 启动前端服务器（Connector）
# 在一个终端窗口中
cd examples/demo/cluster
go run main.go --port 3250 --type connector --frontend=true
步骤 2: 启动后端服务器（Room）
# 在另一个终端窗口中
cd examples/demo/cluster
go run main.go --port 3251 --type room --frontend=false
使用 Makefile 快捷方式：
# 运行前端服务器
make run-cluster-example-frontend
# 运行后端服务器
make run-cluster-example-backend
示例 3: gRPC 集群示例
使用 gRPC 作为 RPC 传输的集群示例。
# 启动 connector 服务器
make run-cluster-grpc-example-connector
# 启动 room 服务器
make run-cluster-grpc-example-room
示例 4: Worker 示例
展示后台任务处理功能。
# 启动 room 服务器（带前端）
make run-cluster-worker-example-room
# 启动 metagame 服务器
make run-cluster-worker-example-metagame
# 启动 worker 服务器
make run-cluster-worker-example-worker
示例 5: 自定义指标示例
cd examples/demo/custom_metrics
go run main.go --port 3250
示例 6: 速率限制示例
go run examples/demo/rate_limiting/main.go
🧪 测试服务器
使用 pitaya-cli 测试
Pitaya 提供了一个 REPL 客户端用于测试。
# 构建 pitaya-cli
make build
# 运行 pitaya-cli
./build/pitaya-cli.exe  # Windows
# 或
./build/pitaya-cli      # Linux/Mac
pitaya-cli 使用示例：
# 连接到服务器
>>> connect localhost:3250
connected!
# 发送请求
>>> request room.room.entry
>>> sv-> {"code":0,"result":"ok"}
# 获取会话数据
>>> request connector.getsessiondata
>>> sv-> {"data":{}}
# 设置会话数据
>>> request connector.setsessiondata {"data":{"key":"value"}}
>>> sv-> {"code":200,"msg":"success"}
# 退出
>>> exit
使用浏览器测试聊天室
对于聊天室示例，可以直接在浏览器中测试：
1. 启动聊天室服务器：
cd examples/demo/chat
go run main.go
2. 打开多个浏览器标签页访问：http://localhost:3251/web/
3. 在不同标签页中输入消息，可以看到实时聊天效果
🔧 常用命令
构建相关
make build              # 构建 pitaya-cli
make build-k6-extension # 构建 k6 扩展
测试相关
make test               # 运行所有测试
make test-coverage      # 运行单元测试并生成覆盖率
make e2e-test           # 运行端到端测试
make e2e-test-nats      # 运行 NATS E2E 测试
make e2e-test-grpc      # 运行 gRPC E2E 测试
代码生成
make protos-compile     # 编译 protobuf 文件
make mocks              # 生成所有 mocks
依赖管理
make setup              # 安装 Go 依赖
make setup-ci           # 安装 CI 工具
Docker 服务
cd examples/testing
docker compose up -d           # 启动所有服务
docker compose down            # 停止所有服务
docker compose logs -f         # 查看日志
docker compose ps              # 查看状态
🐛 故障排除
1. Docker 服务启动失败
# 检查 Docker 是否运行
docker ps
# 查看服务日志
cd examples/testing
docker compose logs etcd
docker compose logs nats
docker compose logs redis
# 重启服务
docker compose restart
2. 端口被占用
# Windows 查看端口占用
netstat -ano | findstr :3250
# 修改示例中的端口
go run main.go --port 3252  # 使用其他端口
3. Go 依赖问题
# 清理并重新下载依赖
go mod tidy
go mod download
# 更新依赖
go get -u ./...
4. 连接失败
- 确保依赖服务正在运行：docker compose ps
- 检查防火墙设置
- 确认服务器地址和端口正确
📊 监控和调试
查看日志
服务器启动后会输出详细日志，包括：
- 服务器启动信息
- 连接状态
- RPC 调用信息
- 错误信息
Prometheus 指标
某些示例会暴露 Prometheus 指标：
- 前端服务器：http://localhost:9090/metrics
- 后端服务器：http://localhost:9091/metrics
OpenTelemetry 追踪
支持分布式追踪，需要配置 Jaeger：
make run-jaeger-aio
# 访问 Jaeger UI: http://localhost:16686
🎯 推荐的学习路径
1. 从简单开始：先运行 chat 示例，理解基本概念
2. 集群模式：运行 cluster 示例，了解分布式架构
3. 高级功能：尝试 worker、custom_metrics 等示例
4. 实际开发：基于示例创建自己的游戏服务器
💡 开发提示
- 修改代码后，重新运行 go run main.go 即可
- 使用 pitaya-cli 进行快速测试
- 查看 examples/demo/ 下的其他示例学习更多功能
- 参考 pkg/ 下的源码了解实现细节
现在你可以开始运行和探索 Pitaya 游戏服务器框架了！