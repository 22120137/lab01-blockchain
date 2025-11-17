# Lab01 - Minimal Layer-1 Blockchain (Golang)

## Build & Run

The Go module resides inside `src/`. All build/test commands should be issued from that directory.

```powershell
cd src
if (!(Test-Path ..\bin)) { New-Item ..\bin -ItemType Directory | Out-Null }
go build -o ../bin/simulator.exe ./cmd/simulator   # on Windows (.exe)
../bin/simulator.exe -config ../config/scenario1.json -out ../logs/run.log
```

On Linux/macOS replace the last two lines with:

```bash
mkdir -p ../bin
go build -o ../bin/simulator ./cmd/simulator
../bin/simulator -config ../config/scenario1.json -out ../logs/run.log
```

`config/scenario1.json` defines number of validators, latency window, drop/duplicate rate, queue limits, outbound rate limit, temporary block duration, and the `ChainID` used for domain-separated signatures. Set `DeterministicLogs` to `true` in the config to obtain byte-stable logs.

## Testing

Run all unit and integration tests (crypto, state machine, network, consensus, simulator config):

```powershell
cd src
go test ./...
```

Or call the convenience script (runs tests + determinism check) from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File src\scripts\run_all.ps1
```

## Determinism Check

The repository includes `cmd/tools/checkdet`, which runs the simulator twice with the same config and verifies that the logs match exactly:

```powershell
cd src
go run ./cmd/tools/checkdet -config ../config/scenario1.json
```

The tool now compares both the simulator logs and the final state snapshot hashes, satisfying the “identical logs and final state” requirement in `Lab01Final.txt`.

## Submission Checklist

- `src/` contains the Go module (`go.mod`, `cmd/`, `pkg/`, tools, scripts)
- `tests/` documents how to execute the test suite from `src/`
- `logs/` stores simulator outputs (or is left empty for graders to fill)
- `config/` holds scenario JSON files
- Top-level `README.md` (this file) and `REPORT.pdf` (replace placeholder with the actual report)
- `scripts/run_all.ps1` runs tests and determinism check automatically (works from repo root)
