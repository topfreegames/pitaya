# War Inc Rising Quick Test Script

Write-Host "=== War Inc Rising Quick Test ===" -ForegroundColor Green

# Check if servers are running
Write-Host "`nChecking server status..." -ForegroundColor Yellow
$connectorRunning = Get-Process -Name "connector" -ErrorAction SilentlyContinue
$roomRunning = Get-Process -Name "room" -ErrorAction SilentlyContinue

if (-not $connectorRunning -or -not $roomRunning) {
    Write-Host "Servers are not running, starting..." -ForegroundColor Red
    & ".\start-servers.ps1"
    Start-Sleep -Seconds 5
}

Write-Host "Servers are running" -ForegroundColor Green

# Run link tests
Write-Host "`nRunning link tests..." -ForegroundColor Yellow
$testPath = "D:\learningplace\first-game-server\game-client\cmd\test"
Set-Location $testPath

Write-Host "Test path: $testPath" -ForegroundColor Cyan
Write-Host "Command: go run main.go" -ForegroundColor Cyan

try {
    go run main.go
    Write-Host "`n Link tests completed" -ForegroundColor Green
} catch {
    Write-Host "`n Link tests failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host "`n=== Tests completed ===" -ForegroundColor Green
Write-Host "`nSee detailed logs above" -ForegroundColor Cyan
