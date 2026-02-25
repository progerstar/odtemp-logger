# Build for Windows (PowerShell)
# Run on Windows

$ErrorActionPreference = "Stop"

$APP_NAME = "ODTemp Logger"
$APP_ID = "com.opendev.odtemp-logger"

# Extract version from main.go
$match = Select-String -Path "main.go" -Pattern 'VERSION\s*=\s*"([^"]+)"'
if (-not $match) {
    Write-Host "Error: VERSION not found in main.go" -ForegroundColor Red
    exit 1
}
$VERSION = $match.Matches[0].Groups[1].Value

Write-Host "Building $APP_NAME v$VERSION for Windows..." -ForegroundColor Cyan

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    exit 1
}

# Check fyne CLI
if (-not (Get-Command fyne -ErrorAction SilentlyContinue)) {
    Write-Host "Installing fyne CLI..." -ForegroundColor Yellow
    go install fyne.io/tools/cmd/fyne@latest
}

# Build
Write-Host "Building binary..." -ForegroundColor Green
go build -buildvcs=false -ldflags="-H windowsgui" -o "odtemp-logger_${VERSION}_windows_amd64.exe" .

# Build package with icon
Write-Host "Creating package..." -ForegroundColor Green
fyne package -os windows -name "$APP_NAME" --app-id "$APP_ID" -icon Icon.png

Write-Host ""
Write-Host "Done: odtemp-logger_${VERSION}_windows_amd64.exe" -ForegroundColor Green
