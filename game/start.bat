@echo off
REM 游戏服务器启动脚本
REM 用于快速启动所有服务

echo ========================================
echo War Inc Rising Game Server 启动脚本
echo ========================================
echo.

REM 检查 Docker 是否运行
docker ps >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Docker 未运行，请先启动 Docker Desktop
    pause
    exit /b 1
)

echo [1/4] 启动基础设施服务...
cd game
docker-compose up -d
if %errorlevel% neq 0 (
    echo [错误] 启动基础设施失败
    pause
    exit /b 1
)
echo [完成] 基础设施服务已启动
echo.

echo [2/4] 等待服务就绪...
timeout /t 5 /nobreak >nul
echo [完成] 服务已就绪
echo.

echo [3/4] 启动 Connector Server...
start "Connector Server" cmd /k "cd /d %CD%\cmd\connector && connector.exe --port 3250"
echo [完成] Connector Server 已启动
echo.

echo [4/4] 启动 Room Server...
start "Room Server" cmd /k "cd /d %CD%\cmd\room && room.exe --port 3260"
echo [完成] Room Server 已启动
echo.

echo ========================================
echo 所有服务已启动！
echo ========================================
echo.
echo 服务地址:
echo   - Connector: ws://localhost:3250
echo   - Room:      localhost:3260
echo   - 健康检查: http://localhost:8080/health
echo.
echo 按 Ctrl+C 停止所有服务
echo.

REM 等待用户中断
pause

echo.
echo [停止] 正在停止所有服务...
docker-compose down
echo [完成] 所有服务已停止
echo.

pause
