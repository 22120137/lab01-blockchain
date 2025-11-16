# Lab01 - Minimal Layer-1 Blockchain (Golang)

## Build & Run

```powershell
go build -o bin/simulator ./cmd/simulator
.\bin\simulator -config config/scenario1.json -out logs/run.log
```

`config/scenario1.json` defines number of validators, latency window, drop/duplicate rate, and queue limits. Set `DeterministicLogs` to `true` in the config to obtain byte-stable logs.

## Testing

Run all unit and integration tests (crypto, state machine, network, consensus, simulator config):

```powershell
go test ./...
```

## Determinism Check

The repository includes `cmd/tools/checkdet`, which runs the simulator twice with the same config and verifies that the logs match exactly:

```powershell
go run ./cmd/tools/checkdet -config config/scenario1.json
```

This command relies on deterministic logging and is used to justify requirement 8 in `Lab01Final.txt`.
