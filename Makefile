# Kairo Performance Benchmarking, Load & Stress Suite
.PHONY: bench bench-export bench-load bench-load-quick bench-frontend bench-all bench-isolated bench-docker bench-k6 report baseline compare stress race test help

help:
	@echo "Kairo Complete Application Benchmarking & Testing Suite:"
	@echo "  make report            Run full standardized benchmark suite under 2-vCPU isolation & save artifacts"
	@echo "  make bench             Run Tier 1 microbenchmarks (storage, replay, locks, FFmpeg builder)"
	@echo "  make baseline          Run Tier 1 microbenchmarks (5 runs) & save baseline_jsonl.txt"
	@echo "  make compare           Statistically compare current run against baseline_jsonl.txt"
	@echo "  make bench-isolated    Run benchmarks under standardized server profile (2 vCPUs, GOMAXPROCS=2, core-pinned)"
	@echo "  make bench-docker      Run benchmarks inside Docker container with hard cgroup limits (2.0 CPU, 1GB RAM)"
	@echo "  make bench-k6          Run external Grafana k6 REST & WebSocket stress scripts (via local k6 or Docker)"
	@echo "  make bench-load        Run Tier 2 WebSocket live collaboration load test (100 rooms, 500 ops/sec, 30s)"
	@echo "  make bench-load-quick  Run quick Tier 2 load test (10 rooms, 100 ops/sec, 5s)"
	@echo "  make bench-export      Run backend Export subsystem load test (20 jobs, concurrency 5)"
	@echo "  make bench-frontend    Run frontend performance benchmarks (timeline math, drag, zoom, export store)"
	@echo "  make bench-all         Run the complete test & benchmark suite across all application subsystems"
	@echo "  make stress            Run crash recovery, race contention, and queue saturation stress tests"
	@echo "  make race              Run Go race detector across the entire server codebase"
	@echo "  make test              Run all unit & domain tests (backend and frontend)"

report:
	@./bench.sh all $(NAME)

bench:
	cd server && go test -bench=. -benchmem -v ./benchmarks/...

baseline:
	mkdir -p benchmarks/results server/benchmarks/results
	@echo "==> Capturing golden baseline under standardized 2-vCPU server profile (GOMAXPROCS=2)..."
	GOMAXPROCS=2 cd server && go test -bench=. -benchmem -count=5 ./benchmarks/... | tee ../benchmarks/results/baseline_jsonl.txt
	cp benchmarks/results/baseline_jsonl.txt server/benchmarks/results/baseline_jsonl.txt
	@echo "Baseline saved to benchmarks/results/baseline_jsonl.txt"

compare:
	@if [ ! -f benchmarks/results/current.txt ]; then \
		echo "Capturing current run under standardized 2-vCPU profile for comparison..."; \
		GOMAXPROCS=2 cd server && go test -bench=. -benchmem -count=5 ./benchmarks/... > ../benchmarks/results/current.txt; \
	fi
	cd server && go run ./cmd/bench-compare ../benchmarks/results/baseline_jsonl.txt ../benchmarks/results/current.txt

bench-isolated:
	@echo "==> Running benchmarks under standardized 2-vCPU server isolation profile (pinned cores)..."
	GOMAXPROCS=2 taskset -c 0,1 go test -bench=. -benchmem -count=5 ./server/benchmarks/...

bench-docker:
	docker compose --profile bench run --rm benchmarks

bench-k6:
	@if command -v k6 >/dev/null 2>&1; then \
		echo "==> Running k6 REST API test with local k6..."; \
		k6 run benchmarks/k6/rest_api_stress.js; \
		echo "==> Running k6 WebSocket test with local k6..."; \
		k6 run benchmarks/k6/websocket_stress.js; \
	elif command -v docker >/dev/null 2>&1; then \
		echo "==> k6 not found locally; running via official grafana/k6 Docker container..."; \
		docker run --rm -i --network=host -v "$$(pwd)/benchmarks/k6:/scripts" grafana/k6 run /scripts/rest_api_stress.js; \
		docker run --rm -i --network=host -v "$$(pwd)/benchmarks/k6:/scripts" grafana/k6 run /scripts/websocket_stress.js; \
	else \
		echo "Error: Neither k6 nor docker is installed to run k6 scripts."; exit 1; \
	fi

bench-load:
	cd server && go run ./cmd/bench-load \
		--addr="127.0.0.1:8080" \
		--rooms=100 \
		--peers=2 \
		--rate=500 \
		--duration=30s \
		--output=summary

bench-load-quick:
	cd server && go run ./cmd/bench-load \
		--addr="127.0.0.1:8080" \
		--rooms=10 \
		--peers=2 \
		--rate=100 \
		--duration=5s \
		--output=summary

bench-export:
	cd server && go run ./cmd/bench-export \
		--addr="127.0.0.1:8080" \
		--projects=20 \
		--concurrency=5 \
		--output=summary

bench-frontend:
	cd client && npm run bench

bench-all:
	@./bench.sh all $(NAME)

stress:
	cd server && go test -v -race ./benchmarks -run=TestStress

race:
	cd server && go test -race ./...

test:
	@echo "==> Running backend tests..."
	cd server && go test ./...
	@echo "==> Running frontend tests..."
	cd client && npm test
