## Tests Overview

Chương trình có 3 nhóm kiểm thử như sau:
- **Unit/Integration**: crypto (domain ký), encoding/state hash, state machine (nonce/ownership), network simulator (delay/drop/dup/rate-limit), mempool, và consensus (vote, replay, lock).
- **End-to-End**: các kịch bản đồng thuận (finalization, invalid header/vote, replay, delay/drop safety, determinism, và một case 8 node).
- **Determinism**: chạy simulator hai lần với cùng config, so sánh byte-log và hash state cuối.

Toàn bộ test chạy trong module `src/`.

### Run the full Go test suite

```powershell
cd ..\src
go test ./...
```

### Determinism check

```powershell
cd ..\src
go run ./cmd/tools/checkdet -config ../config/scenario1.json
```

Lệnh này chạy simulator hai lần, kiểm tra log và state giống hệt nhau.

### Convenience script

Từ thư mục gốc:

```powershell
powershell -ExecutionPolicy Bypass -File src\scripts\run_all.ps1
```

Script tự động vào `src/`, chạy `go test ./...`, rồi chạy determinism check.
