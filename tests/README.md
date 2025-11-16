## Tests Overview

The Go module now resides inside `src/`. All unit, integration, and determinism checks must be executed from that directory.

### Run the full Go test suite

```powershell
cd ..\src
go test ./...
```

This covers crypto, state machine, network simulator, consensus, and simulator packages.

### Determinism check

```powershell
cd ..\src
go run ./cmd/tools/checkdet -config ../config/scenario1.json
```

The command runs the simulator twice with the same configuration and ensures logs and state are identical.

### Convenience script

From the repository root you can run:

```powershell
powershell -ExecutionPolicy Bypass -File src\scripts\run_all.ps1
```

The script automatically enters `src/`, runs `go test ./...`, and then performs the determinism check.
