# Standardized Kairo Benchmark & Load Testing Report

> **Active Run / Feature**: `production_k6_verified`  
> **Timestamp**: `2026-11-26T08:56:41Z`  
> **Standard Server Profile**: `2 vCPUs`, `1024 MB memory limit`, `GOMAXPROCS=2`, `linux/amd64`  

## 📊 Benchmark History & Incremental Feature Progression

| # | Timestamp | Run / Feature Name | Append Latency | Replay (1k) | WS Latency (p50) | Export Jobs/s | k6 Req/s | Stress & Race | Snapshot Link |
| :-: | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :-: | :--- |
| 1 | `2026-11-25 17:34 UTC` | `baseline_jsonl` | 4.47 ms (7 allocs) | 4.61 ms | 4.34 ms | 16.9/s | 115.0/s | PASS | [baseline_jsonl](results/standardized_report.md) |
| 2 | `2026-11-26 08:29 UTC` | `initial_jsonl_engine` | 5.91 µs (8 allocs) | 4.85 ms | 0.29 ms | 2286.9/s | 486.3/s | PASS | [initial_jsonl_engine](history/2026-11-26_140009_initial_jsonl_engine.md) |
| 3 | `2026-11-26 08:56 UTC` | **`production_k6_verified`** *(current)* | 5.86 µs (8 allocs) | 4.57 ms | 0.31 ms | 2332.9/s | 1092.8/s | PASS | [production_k6_verified](history/2026-11-26_142715_production_k6_verified.md) |

---

## 1. Storage WAL & Timeline Microbenchmarks (`production_k6_verified`)

| Component Benchmark | Latency | Memory Allocation | Heap Allocs | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkStorage_AppendSequential` | 5.86 µs | 609 B/op | 8 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_CommitAuthor` | 10.52 µs | 4014 B/op | 51 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/100_ops` | 455.18 µs | 865660 B/op | 2211 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/500_ops` | 2.34 ms | 5769979 B/op | 11036 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/1000_ops` | 4.57 ms | 11860408 B/op | 22059 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/5000_ops` | 27.22 ms | 60930110 B/op | 110296 allocs/op | Deterministic heap allocation |
| `BenchmarkOwnership_Contention` | 1.09 µs | 968 B/op | 6 allocs/op | Deterministic heap allocation |

## 2. Concurrency Stress & Crash Recovery

| Stress Scenario | Verification Goal | Status |
| :--- | :--- | :--- |
| `TestStress_ConcurrentConflictRace` | Hostile concurrency & data race detection | **PASS** ((0.01s)) |
| `TestStress_CorruptedLogRecovery` | Hostile concurrency & data race detection | **PASS** ((0.00s)) |
| `TestStress_OwnershipLockStorm` | Hostile concurrency & data race detection | **PASS** ((0.09s)) |
| `TestStress_ExportQueueSaturation` | Hostile concurrency & data race detection | **PASS** ((0.02s)) |

## 3. Real-Time Collaboration (WebSockets)

- **Actual Throughput**: `199.8 ops/sec`
- **Success Rate**: `100.0%`
- **Transit Latency**: p50 = `0.31 ms`, p95 = `0.60 ms`, p99 = `0.94 ms`
- **Resource Usage**: Server CPU = `10.2%`, Memory RSS = `29.2 MB`

## 4. Backend Export Subsystem

- **Job Throughput**: `2332.9 jobs/sec`
- **Submit Latency**: p50 = `0.53 ms`, p95 = `1.50 ms`
- **Status Polling Latency**: p50 = `0.09 ms`

## 5. Frontend Engine Math

| Frontend Operation | Throughput & Latency |
| :--- | :--- |
| `Clip Composition Ordering (1,000 clips)` | 7,380.2 ops/sec (135,497.6 ns/op) |
| `Zoom & Tick Step Calculation` | 41,265,088.6 ops/sec (    24.2 ns/op) |
| `Ruler Tick Array Generation (1.8k frames)` | 981,037.6 ops/sec ( 1,019.3 ns/op) |
| `Drag Target & Transform Math` | 93,350,982.9 ops/sec (    10.7 ns/op) |
| `Export Store Preflight & Settings Dispatch` | 274,403.6 ops/sec ( 3,644.3 ns/op) |

## 6. External Grafana k6 REST & WebSocket Stress

| Scenario | Throughput | Latency p95 | Failure Rate | Status |
| :--- | :--- | :--- | :--- | :--- |
| `k6 REST API Stress` | 1092.8 req/s | 0.00 ms | 0.0% | **PASS** |
| `k6 WebSocket Stress` | 23.6 req/s | 0.00 ms | 0.0% | **PASS** |
| `k6 Video Export Stress` | 519.7 req/s | 0.00 ms | 0.0% | **PASS** |
