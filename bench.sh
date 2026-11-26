#!/usr/bin/env bash
set -euo pipefail

# Kairo Universal Benchmark, Load & Stress Testing Runner CLI
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

COMMAND="${1:-help}"
shift || true

case "$COMMAND" in
    all|bench-all|report)
        RUN_NAME="${1:-${NAME:-}}"
        if [ -n "$RUN_NAME" ]; then
            shift || true
        fi
        if [ -z "$RUN_NAME" ]; then
            if [ -t 0 ]; then
                read -r -p "Enter benchmark run/feature name (e.g. initial_jsonl, sqlite_migration, batch_export): " RUN_NAME
            fi
        fi
        RUN_NAME="${RUN_NAME:-run_$(date +%Y%m%d_%H%M%S)}"
        echo "==> Running Complete Standardized Benchmark Suite for: $RUN_NAME..."
        export GOMAXPROCS=2
        if command -v taskset >/dev/null 2>&1; then
            echo "==> Enforcing standardized 2-vCPU hardware core isolation (taskset -c 0,1, GOMAXPROCS=2)..."
            cd server && taskset -c 0,1 go run ./cmd/bench-suite --name="$RUN_NAME" "$@"
        else
            echo "==> Enforcing standardized 2-vCPU scheduler limits (GOMAXPROCS=2)..."
            cd server && go run ./cmd/bench-suite --name="$RUN_NAME" "$@"
        fi
        ;;
    bench|component)
        echo "==> Running Tier 1 Component Microbenchmarks (Go testing.B)..."
        cd server && go test -bench=. -benchmem -v ./benchmarks/... "$@"
        ;;
    baseline)
        echo "==> Capturing Tier 1 JSONL Baseline (5 runs)..."
        mkdir -p benchmarks/results server/benchmarks/results
        cd server && go test -bench=. -benchmem -count=5 ./benchmarks/... | tee ../benchmarks/results/baseline_jsonl.txt
        cp "$ROOT_DIR/benchmarks/results/baseline_jsonl.txt" "$ROOT_DIR/server/benchmarks/results/baseline_jsonl.txt"
        echo "==> Baseline successfully captured in benchmarks/results/baseline_jsonl.txt"
        ;;
    compare)
        echo "==> Comparing Benchmark Results..."
        cd server && go run ./cmd/bench-compare "$@"
        ;;
    isolated)
        echo "==> Running benchmarks under standardized 2-vCPU pinned core profile..."
        GOMAXPROCS=2 taskset -c 0,1 go test -bench=. -benchmem -count=5 ./server/benchmarks/... "$@"
        ;;
    docker)
        echo "==> Running benchmarks inside Docker cgroups container..."
        make bench-docker
        ;;
    k6|bench-k6)
        echo "==> Running k6 external load/stress tests..."
        make bench-k6
        ;;
    load|bench-load)
        echo "==> Running Tier 2 WebSocket Live Collaboration Load Test..."
        cd server && go run ./cmd/bench-load "$@"
        ;;
    load-quick)
        echo "==> Running Quick Tier 2 WebSocket Load Test..."
        cd server && go run ./cmd/bench-load --rooms=10 --peers=2 --rate=100 --duration=5s "$@"
        ;;
    export|bench-export)
        echo "==> Running Backend Export Subsystem Load Test..."
        cd server && go run ./cmd/bench-export "$@"
        ;;
    frontend|bench-frontend)
        echo "==> Running Frontend Engine Performance Benchmarks..."
        cd client && npm run bench
        ;;
    stress)
        echo "==> Running Concurrency & Crash Recovery Stress Tests..."
        cd server && go test -v -race ./benchmarks -run=TestStress "$@"
        ;;
    race)
        echo "==> Running Go Data Race Detection..."
        cd server && go test -race ./... "$@"
        ;;
    test)
        echo "==> Running All Unit Tests (Backend + Frontend)..."
        cd server && go test ./...
        cd "$ROOT_DIR/client" && npm test
        ;;
    help|--help|-h)
        echo "Usage: ./bench.sh [command] [extra flags]"
        echo ""
        echo "Available Commands:"
        echo "  bench | component     Run Tier 1 microbenchmarks (storage, replay, locks, FFmpeg builder)"
        echo "  baseline              Capture 5-run statistical baseline (benchstat format)"
        echo "  compare <base> <cand> Compare two benchmark files statistically"
        echo "  isolated              Run with standardized 2-vCPU isolation & core pinning (taskset)"
        echo "  load | bench-load     Run Tier 2 WebSocket live collaboration load tester"
        echo "  load-quick            Run quick 5-second smoke load test"
        echo "  export | bench-export Run backend Export subsystem load test"
        echo "  frontend              Run frontend timeline & export math benchmarks"
        echo "  all                   Run the complete benchmark & load suite across all parts"
        echo "  stress                Run crash recovery, race contention, and queue stress tests"
        echo "  race                  Run Go race detector across the server"
        echo "  test                  Run all unit tests across backend & frontend"
        echo ""
        echo "Examples:"
        echo "  ./bench.sh component"
        echo "  ./bench.sh compare benchmarks/results/baseline_jsonl.txt benchmarks/results/current.txt"
        echo "  ./bench.sh isolated"
        echo "  ./bench.sh load --rooms=50 --rate=200 --duration=10s"
        echo "  ./bench.sh export --projects=30 --concurrency=10"
        echo "  ./bench.sh all"
        ;;
    *)
        echo "Unknown command: $COMMAND. Run './bench.sh help' for usage."
        exit 1
        ;;
esac
