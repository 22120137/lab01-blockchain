$ErrorActionPreference = "Stop"

$moduleRoot = Split-Path -Parent $PSScriptRoot
Push-Location $moduleRoot

Write-Host "Running all Go tests..."
go test ./...
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    exit $LASTEXITCODE
}

Write-Host "Checking determinism..."
go run ./cmd/tools/checkdet -config ../config/scenario1.json
$code = $LASTEXITCODE
Pop-Location
if ($code -ne 0) { exit $code }

Write-Host "All tests passed."
