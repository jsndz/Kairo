# Standardized Kairo Benchmark Report

**Timestamp**: `2026-11-25T17:34:03Z`  
**Standard Profile**: `2 vCPUs`, `1024 MB memory limit`, `GOMAXPROCS=2`, `linux/amd64`  

## 1. Storage WAL & Timeline Microbenchmarks

| Component Benchmark | Latency | Memory Allocation | Heap Allocs | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkStorage_AppendSequential` | 4.47 ms | 704 B/op | 7 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_CommitAuthor` | 4.44 ms | 4324 B/op | 50 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/100_ops` | 436.58 µs | 863874 B/op | 2209 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/500_ops` | 2.25 ms | 5757300 B/op | 11024 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/1000_ops` | 4.61 ms | 11834031 B/op | 22040 allocs/op | Deterministic heap allocation |
| `BenchmarkStorage_ReplayHydration/5000_ops` | 30.26 ms | 60743151 B/op | 110171 allocs/op | Deterministic heap allocation |
| `BenchmarkOwnership_Contention` | 989.2 ns | 952 B/op | 6 allocs/op | Deterministic heap allocation |

## 2. Real-Time Collaboration (WebSockets)

- **Actual Throughput**: `198.2 ops/sec`
- **Success Rate**: `100.0%`
- **Transit Latency**: p50 = `4.34 ms`, p95 = `7.92 ms`, p99 = `40.31 ms`
- **Resource Usage**: Server CPU = `10.2%`, Memory RSS = `20.8 MB`

## 3. Backend Export Subsystem

- **Job Throughput**: `16.9 jobs/sec`
- **Submit Latency**: p50 = `46.05 ms`, p95 = `55.63 ms`
- **Status Polling Latency**: p50 = `4.53 ms`

## 4. Frontend Engine Math

| Frontend Operation | Throughput & Latency |
| :--- | :--- |
| `Clip Composition Ordering (1,000 clips)` | 4,226.3 ops/sec (236,613.8 ns/op) |
| `Zoom & Tick Step Calculation` | 47,748,924.7 ops/sec (    20.9 ns/op) |
| `Ruler Tick Array Generation (1.8k frames)` | 851,574.3 ops/sec ( 1,174.3 ns/op) |
| `Drag Target & Transform Math` | 33,219,059.8 ops/sec (    30.1 ns/op) |
| `Export Store Preflight & Settings Dispatch` | 221,056.5 ops/sec ( 4,523.7 ns/op) |
