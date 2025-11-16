$ErrorActionPreference = "Stop"

Write-Host "Running all Go tests..."
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Checking determinism..."
go run ./cmd/tools/checkdet -config config/scenario1.json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "All tests passed."
