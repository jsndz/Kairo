# Kairo k6 Load & Stress Testing Scripts

These scripts provide external black-box HTTP and WebSocket load testing using [Grafana k6](https://k6.io/).

## Available Scripts

1. `rest_api_stress.js`: Ramps up virtual users (VUs) from 20 to 100 hitting health checks, room creation (`POST /api/rooms`), and project inspection (`GET /api/projects/{id}`). Enforces p95 latency thresholds $< 15\text{ ms}$.
2. `websocket_stress.js`: Connects simulated editors via WebSockets, verifies handshake protocol and presence broadcast latency.

## How to Run

### Option 1: Using Local k6
```bash
# REST API stress test
k6 run benchmarks/k6/rest_api_stress.js

# WebSocket stress test
k6 run benchmarks/k6/websocket_stress.js

# Target custom address
k6 run -e TARGET_URL=http://127.0.0.1:8080 benchmarks/k6/rest_api_stress.js
```

### Option 2: Using Docker (Zero Installation)
```bash
docker run --rm -i \
  --net=host \
  -v "$PWD/benchmarks/k6:/scripts" \
  grafana/k6 run /scripts/rest_api_stress.js
```
