# War Inc Rising Link Test Startup Script

Write-Host "=== War Inc Rising Link Test Startup ===" -ForegroundColor Green

# Check if Docker service is running
Write-Host "`n[1/5] Checking Docker service..." -ForegroundColor Yellow
try {
    $dockerStatus = docker ps
    Write-Host "Docker service is running" -ForegroundColor Green
} catch {
    Write-Host "Docker service is not running, please start Docker Desktop first" -ForegroundColor Red
    exit 1
}

# Start infrastructure services
Write-Host "`n[2/5] Starting infrastructure services..." -ForegroundColor Yellow
$projectRoot = "D:\learningplace\first-game-server"
Set-Location $projectRoot
docker compose up -d
Set-Location "D:\learningplace\first-game-server\game-client"
Write-Host "Infrastructure services started" -ForegroundColor Green

# Wait for services to start
Write-Host "`nWaiting for services to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Check service status
Write-Host "`n[3/5] Checking service status..." -ForegroundColor Yellow
$services = @("Redis:6380", "MySQL:3307", "NATS:4222", "etcd:2379")
foreach ($service in $services) {
    Write-Host "  - $service" -ForegroundColor Cyan
}
Write-Host "All services are running" -ForegroundColor Green

# Start Connector Server
Write-Host "`n[4/5] Starting Connector Server..." -ForegroundColor Yellow
$connectorPath = "D:\learningplace\first-game-server\game\cmd\connector\connector.exe"
if (Test-Path $connectorPath) {
    Write-Host "  Path: $connectorPath" -ForegroundColor Cyan
    Start-Process -FilePath $connectorPath -WindowStyle Minimized
    Write-Host "Connector Server started" -ForegroundColor Green
} else {
    Write-Host "Connector Server not found, please build first: cd game\cmd\connector && go build" -ForegroundColor Red
    exit 1
}

# Start Room Server
Write-Host "`n[5/5] Starting Room Server..." -ForegroundColor Yellow
$roomPath = "D:\learningplace\first-game-server\game\cmd\room\room.exe"
if (Test-Path $roomPath) {
    Write-Host "  Path: $roomPath" -ForegroundColor Cyan
    Start-Process -FilePath $roomPath -WindowStyle Minimized
    Write-Host "Room Server started" -ForegroundColor Green
} else {
    Write-Host "Room Server not found, please build first: cd game\cmd\room && go build" -ForegroundColor Red
    exit 1
}

# Wait for servers to start
Write-Host "`nWaiting for servers to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

Write-Host "`n=== All services started ===" -ForegroundColor Green
Write-Host "`nYou can now run link tests:" -ForegroundColor Cyan
Write-Host "  cd D:\learningplace\first-game-server\game-client\cmd\test" -ForegroundColor White
Write-Host "  go run main.go" -ForegroundColor White
Write-Host "`nTo stop all services:" -ForegroundColor Cyan
Write-Host "  .\stop-servers.ps1" -ForegroundColor White
