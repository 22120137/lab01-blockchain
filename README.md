# Lab01 - Minimal Layer-1 Blockchain (Golang)

## Build & Run

```powershell
go build -o bin/simulator ./cmd/simulator
.\bin\simulator -config config/scenario1.json -out logs/run.log
```

`config/scenario1.json` defines number of validators, latency window, drop/duplicate rate, queue limits, outbound rate limit, and temporary block duration. Set `DeterministicLogs` to `true` in the config to obtain byte-stable logs.

## Testing

Run all unit and integration tests (crypto, state machine, network, consensus, simulator config):

```powershell
go test ./...
```

Or call the convenience script (runs tests + determinism check):

```powershell
pwsh scripts/run_all.ps1
```

## Determinism Check

The repository includes `cmd/tools/checkdet`, which runs the simulator twice with the same config and verifies that the logs match exactly:

```powershell
go run ./cmd/tools/checkdet -config config/scenario1.json
```

This command relies on deterministic logging and is used to justify requirement 8 in `Lab01Final.txt`.

## Submission Checklist

- Source code under `pkg/`, `cmd/`, `tests/`, and `scripts/`
- Configurations in `config/`
- Logs generated via `cmd/simulator`
- Determinism evidence via `cmd/tools/checkdet`
- `report/REPORT.pdf` describes architecture, bugs, and test plan (replace placeholder before submission)
