# === Config ===
SERVICES := ai-service auth-service document-service gateway websocket-service
GO_SERVICES := $(addprefix server/apps/, $(SERVICES))
CLIENT := client
PROTO_DIR := server/proto
BIN_DIR := bin

# === Default ===
.PHONY: help
help:
	@echo ""
	@echo "Commands"
	@echo ""
	@echo "General:"
	@echo "  make build                 Build all Go services"
	@echo "  make run SERVICE=name      Run a specific Go service"
	@echo "  make dev SERVICE=name      Run a service with 'air' (hot reload)"
	@echo "  make proto                 Generate code from proto/ using buf"
	@echo "  make test                  Run all Go tests"
	@echo "  make fmt                   Format all Go code"
	@echo ""
	@echo "Databases (via Docker):"
	@echo "  make db-up                 Start Postgres/Redis/Kafka containers"
	@echo "  make db-down               Stop DB containers"
	@echo "  make db-logs               Show DB logs"
	@echo ""
	@echo "Frontend:"
	@echo "  make client                Run frontend dev server (npm run dev)"
	@echo ""

# === Build Go Services ===
.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	@for service in $(GO_SERVICES); do \
		echo "Building $$service..."; \
		cd $$service && go build -o ../../../$(BIN_DIR)/$$(basename $$service) ./...; \
	done
	@echo "✅ All services built."

# === Run Go Service ===
.PHONY: run
run:
	@if [ -z "$(SERVICE)" ]; then \
		echo " Missing SERVICE. Use: make run SERVICE=auth-service"; \
		exit 1; \
	fi
	cd server/apps/auth-service && go run /cmd/server/main.go

# === Hot Reload Dev (with air) ===
.PHONY: dev
dev:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Missing SERVICE. Use: make dev SERVICE=auth-service"; \
		exit 1; \
	fi
	cd server/apps/$(SERVICE) && air

# === Proto Code Generation ===
.PHONY: proto
proto:
	cd server && buf generate
	@echo " Protobuf generated."

# === Format ===
.PHONY: fmt
fmt:
	gofmt -s -w .

# === Tests ===
.PHONY: test
test:
	go test ./...

# === Frontend ===
.PHONY: client
client:
	cd $(CLIENT) && npm install && npm run dev

# === Docker Databases ===
.PHONY: db-up db-down db-logs
db-up:
	docker-compose up -d postgres redis kafka

db-down:
	docker-compose down

db-logs:
	docker-compose logs -f
