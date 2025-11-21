# Lab01 - Minimal Layer-1 Blockchain (Golang)

## Tổng quan
- Blockchain tối giản với đồng thuận kiểu Tendermint (Prevote/Precommit), có lock block, round-change khi timeout để bảo đảm safety/liveness.
- Giao dịch key/value thuộc sở hữu người gửi, kèm nonce tuần tự và chữ ký Ed25519 domain `TX:<chain_id>`; header và vote ký `HEADER:<chain_id>` và `VOTE:<chain_id>`.
- State cam kết bằng hash deterministic (dữ liệu + nonce), block commit state hash; ledger và snapshot state dùng cho kiểm thử hội tụ.
- Mạng mô phỏng tick-based: delay, drop/duplicate, rate limit, và chỉ gửi body sau khi peer prevote header; log chi tiết từng sự kiện.
- Tool determinism chạy simulator 2 lần, so sánh byte log và hash state cuối.

**Cấu trúc chính**
- `src/pkg/core/`: đồng thuận, state, encoding, crypto, network simulator.
- `src/cmd/simulator/`: chương trình mô phỏng theo file config JSON.
- `src/cmd/tools/checkdet/`: kiểm tra determinism (log + state).
- `src/scripts/run_all.ps1`: script PowerShell chạy toàn bộ test và determinism.
- `config/`: cấu hình mô phỏng (mặc định `scenario1.json` với NumNodes=8).
- `logs/`: nơi lưu log chạy (có thể để trống, dùng `.gitkeep` để giữ thư mục).
- `tests/`: README hướng dẫn chạy test từ `src/`.

## Build & Run
Go module nằm trong `src/`. Thực hiện build/test trong thư mục này.

**Windows (PowerShell):**
```powershell
cd src
if (!(Test-Path ..\bin)) { New-Item ..\bin -ItemType Directory | Out-Null }
go build -o ../bin/simulator.exe ./cmd/simulator
../bin/simulator.exe -config ../config/scenario1.json -out ../logs/run.log
```

**Linux/macOS:**
```bash
cd src
mkdir -p ../bin
go build -o ../bin/simulator ./cmd/simulator
../bin/simulator -config ../config/scenario1.json -out ../logs/run.log
```

`config/scenario1.json` cấu hình: số validator (>=8), BlocksToFinalize, MaxTicks, LatencyMin/Max, DropRate/DuplicateRate, MaxQueuePerTick, MaxOutboundPerTick, BlockDurationTicks, `DeterministicLogs`, và `ChainID` cho domain ký.

Simulator chạy tới khi **tất cả** node finalize đủ `BlocksToFinalize` hoặc đạt `MaxTicks`, rồi log snapshot state (`STATE|...`) để phục vụ determinism.

## Testing
Chạy unit + integration tests (crypto, state, network, consensus, simulator config):
```powershell
cd src
go test ./...
```

## Kiểm tra tính tất định
Tool `cmd/tools/checkdet`:
```powershell
cd src
go run ./cmd/tools/checkdet -config ../config/scenario1.json
```
Tool chạy simulator hai lần, so sánh log byte-for-byte và hash SHA-256 của snapshot state dòng `STATE|...`. Khác biệt sẽ báo lỗi; giống sẽ in hash log và state.

## Script tiện lợi từ gốc repo (chạy test + determinism):
```powershell
powershell -ExecutionPolicy Bypass -File src\scripts\run_all.ps1
```

## Các điểm kỹ thuật
- **Đồng thuận**: Prevote/Precommit, lock block khi đạt quorum prevote, round-change khi timeout, proposer = `(height+round) % numNodes`. Body chỉ gửi tới peer đã prevote.
- **Mempool**: kiểm soát nonce tăng dần, nhận TX ký đúng domain và đúng key prefix.
- **Network sim**: tick-based, log SEND/DROP/DUP/DELIVER/RATE_BLOCK/UNBLOCK kèm tick/height.
- **Snapshot/Ledger**: `SnapshotState()` và `LedgerSnapshot()` để kiểm thử hội tụ; state hash deterministic.

## Submission Checklist
- `src/` chứa Go module (`go.mod`, `cmd/`, `pkg/`, tools, scripts).
- `tests/` có hướng dẫn chạy test từ `src/`.
- `config/` chứa các file JSON (mặc định NumNodes=8).
- `logs/` dành cho output chạy; dùng `.gitkeep` nếu cần giữ thư mục.
- `README.md` và `REPORT.pdf`.
- `src/scripts/run_all.ps1` chạy test + determinism từ gốc repo.
