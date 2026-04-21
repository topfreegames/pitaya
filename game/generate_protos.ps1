$ErrorActionPreference = "Stop"

$env:PATH = "C:\Users\BW2-14\go\bin;$env:PATH"

$ProtoDir = "game\proto"
$OutDir = "game\internal\proto"

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

$ProtoFiles = @(
    "common.proto",
    "connection.proto",
    "room.proto",
    "game.proto",
    "ranking.proto"
)

$GoPath = [System.Environment]::GetEnvironmentVariable("GOPATH", "User")
if (-not $GoPath) {
    $GoPath = "$env:USERPROFILE\go"
}

$ProtocGenGo = Join-Path $GoPath "bin\protoc-gen-go.exe"

Write-Host "Checking for protoc..."
try {
    $ProtocPath = (Get-Command protoc -ErrorAction Stop).Source
    Write-Host "Found protoc at: $ProtocPath"
} catch {
    Write-Error "protoc not found. Please install protoc first."
    Write-Host "On Windows, you can:"
    Write-Host "  1. Download from https://github.com/protocolbuffers/protobuf/releases"
    Write-Host "  2. Or use chocolatey: choco install protoc"
    Write-Host "  3. Or use scoop: scoop install protobuf"
    exit 1
}

foreach ($ProtoFile in $ProtoFiles) {
    $ProtoPath = Join-Path $ProtoDir $ProtoFile
    Write-Host "Compiling $ProtoFile..."
    
    & protoc "--proto_path=$ProtoDir" "--go_out=$OutDir" "--go_opt=paths=source_relative" $ProtoPath
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to compile $ProtoFile"
        exit 1
    }
}

Write-Host "All proto files compiled successfully!"
