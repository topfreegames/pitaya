# War Inc Rising Stop Servers Script

Write-Host "=== Stopping War Inc Rising Servers ===" -ForegroundColor Green

# Stop Connector Server
Write-Host "`n[1/3] Stopping Connector Server..." -ForegroundColor Yellow
$connectorProcess = Get-Process -Name "connector" -ErrorAction SilentlyContinue
if ($connectorProcess) {
    Stop-Process -Name "connector" -Force
    Write-Host "Connector Server stopped" -ForegroundColor Green
} else {
    Write-Host "Connector Server is not running" -ForegroundColor Gray
}

# Stop Room Server
Write-Host "`n[2/3] Stopping Room Server..." -ForegroundColor Yellow
$roomProcess = Get-Process -Name "room" -ErrorAction SilentlyContinue
if ($roomProcess) {
    Stop-Process -Name "room" -Force
    Write-Host "Room Server stopped" -ForegroundColor Green
} else {
    Write-Host "Room Server is not running" -ForegroundColor Gray
}

# Stop infrastructure services
Write-Host "`n[3/3] Stopping infrastructure services..." -ForegroundColor Yellow
$projectRoot = "D:\learningplace\first-game-server"
Set-Location $projectRoot
docker compose down
Set-Location "D:\learningplace\first-game-server\game-client"
Write-Host "Infrastructure services stopped" -ForegroundColor Green

Write-Host "`n=== All services stopped ===" -ForegroundColor Green
