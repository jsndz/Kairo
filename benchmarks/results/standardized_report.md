# Standardized Kairo Benchmark & Load Testing Report

> **Active Run / Feature**: `NAME=sqlite_storage_migration`  
> **Timestamp**: `2026-11-26T14:32:20Z`  
> **Standard Server Profile**: `2 vCPUs`, `1024 MB memory limit`, `GOMAXPROCS=2`, `linux/amd64`  

## 📊 Benchmark History & Incremental Feature Progression

| # | Timestamp | Run / Feature Name | Append Latency | Replay (1k) | WS Latency (p50) | Export Jobs/s | k6 Req/s | Stress & Race | Snapshot Link |
| :-: | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :-: | :--- |
| 1 | `2026-11-25 17:34 UTC` | `baseline_jsonl` | 4.47 ms (7 allocs) | 4.61 ms | 4.34 ms | 16.9/s | 115.0/s | PASS | [baseline_jsonl](results/standardized_report.md) |
| 2 | `2026-11-26 12:05 UTC` | `jsonl` | 5.44 µs (8 allocs) | 3.89 ms | 0.16 ms | 2409.8/s | 1094.2/s | PASS | [jsonl](history/2026-11-26_173601_jsonl.md) |
| 3 | `2026-11-26 12:15 UTC` | `NAME=frontend_export_benchmarked` | 21.05 µs (7 allocs) | 3.94 ms | 0.30 ms | 825.1/s | 0.0/s | PASS | [name_frontend_export_benchmarked](history/2026-11-26_174535_name_frontend_export_benchmarked.md) |
| 4 | `2026-11-26 14:32 UTC` | **`NAME=sqlite_storage_migration`** *(current)* | 17.12 µs (17 allocs) | 38.05 µs | 0.33 ms | 470.7/s | 0.0/s | PASS | [name_sqlite_storage_migration](history/2026-11-26_200249_name_sqlite_storage_migration.md) |

---

## 1. Storage WAL & Timeline Microbenchmarks (`NAME=sqlite_storage_migration`)

| Component Benchmark | Latency | Memory Allocation | Heap Allocs | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkStorage_AppendSequential` | 24.85 µs | 608 B/op | 7 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_CommitAuthor` | 34.13 µs | 4124 B/op | 51 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/100_ops` | 471.37 µs | 863843 B/op | 2209 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/500_ops` | 2.46 ms | 5757121 B/op | 11024 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/1000_ops` | 5.84 ms | 11833580 B/op | 22039 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/5000_ops` | 35.06 ms | 60745394 B/op | 110173 allocs/op | Deterministic heap allocation |
| `BenchmarkSQLite_AppendSequential` | 17.12 µs | 599 B/op | 17 allocs/op | Deterministic heap allocation |
| `BenchmarkSQLite_SnapshotHydration/100_ops` | 37.23 µs | 3993 B/op | 109 allocs/op | Deterministic heap allocation |
| `BenchmarkSQLite_SnapshotHydration/500_ops` | 36.75 µs | 4025 B/op | 112 allocs/op | Deterministic heap allocation |
| `BenchmarkSQLite_SnapshotHydration/1000_ops` | 38.05 µs | 4026 B/op | 112 allocs/op | Deterministic heap allocation |
| `BenchmarkSQLite_SnapshotHydration/5000_ops` | 37.66 µs | 4024 B/op | 112 allocs/op | Deterministic heap allocation |
| `BenchmarkOwnership_Contention` | 953.7 ns | 938 B/op | 6 allocs/op | Deterministic heap allocation |

## 2. Concurrency Stress & Crash Recovery

| Stress Scenario | Verification Goal | Status |
| :--- | :--- | :--- |
| `TestStress_ConcurrentConflictRace` | Hostile concurrency & data race detection | **PASS** ((0.00s)) |
| `TestStress_CorruptedLogRecovery` | Hostile concurrency & data race detection | **PASS** ((0.00s)) |
| `TestStress_OwnershipLockStorm` | Hostile concurrency & data race detection | **PASS** ((0.06s)) |
| `TestStress_ExportQueueSaturation` | Hostile concurrency & data race detection | **PASS** ((0.01s)) |

## 3. Real-Time Collaboration (WebSockets)

- **Actual Throughput**: `199.8 ops/sec`
- **Success Rate**: `100.0%`
- **Transit Latency**: p50 = `0.33 ms`, p95 = `0.48 ms`, p99 = `0.94 ms`
- **Resource Usage**: Server CPU = `8.2%`, Memory RSS = `39.0 MB`

## 4. Backend Export Subsystem

- **Job Throughput**: `470.7 jobs/sec`
- **Submit Latency**: p50 = `2.03 ms`, p95 = `7.35 ms`
- **Status Polling Latency**: p50 = `0.19 ms`

## 5. Frontend Engine Math

| Frontend Operation | Throughput & Latency |
| :--- | :--- |
| `Clip Composition Ordering (1,000 clips)` | 6,297 ops/sec (158,806.3 ns/op) |
| `Zoom & Tick Step Calculation` | 55,307,532 ops/sec (    18.1 ns/op) |
| `Ruler Tick Array Generation (1.8k frames)` | 904,523 ops/sec ( 1,105.6 ns/op) |
| `Drag Target & Transform Math` | 60,979,402.4 ops/sec (    16.4 ns/op) |
| `Export Store Preflight & Settings Dispatch` | 249,265.9 ops/sec ( 4,011.8 ns/op) |
| `Export Letterbox & Aspect Ratio Math` | 35,697,826.5 ops/sec (      28 ns/op) |
| `Client Export Frame Compositing Loop` | 15,945,804.1 frames/sec (    0.06 µs/frame) |

## 6. External Grafana k6 REST & WebSocket Stress

| Scenario | Throughput | Latency p95 | Failure Rate | Status |
| :--- | :--- | :--- | :--- | :--- |
| `k6 REST API Stress` | 0.0 req/s | 0.00 ms | 0.0% | **PASS** |
| `k6 WebSocket Stress` | 0.0 req/s | 0.00 ms | 0.0% | **PASS** |
| `k6 Video Export Stress` | 0.0 req/s | 0.00 ms | 0.0% | **PASS** |
