# Root Makefile

TERM_CMD = gnome-terminal -- bash -c
# If you're on macOS, comment above and use:
# TERM_CMD = osascript -e 'tell application "Terminal" to do script'

BIN_DIR = server/bin

.PHONY: all dev build client gateway auth-service document-service ai-service websocket-service

all: build

### Build All Go Services ###
build: $(BIN_DIR)/gateway $(BIN_DIR)/auth-service $(BIN_DIR)/document-service $(BIN_DIR)/ai-service $(BIN_DIR)/websocket-service
	@echo "✅ All Go services built successfully."

$(BIN_DIR)/gateway:
	@echo "Building Gateway..."
	@mkdir -p $(BIN_DIR)
	cd server/apps/gateway && go build -o ../../bin/gateway ./cmd/server

$(BIN_DIR)/auth-service:
	@echo "Building Auth Service..."
	@mkdir -p $(BIN_DIR)
	cd server/apps/auth-service && go build -o ../../bin/auth-service ./cmd/server

$(BIN_DIR)/document-service:
	@echo "Building Document Service..."
	@mkdir -p $(BIN_DIR)
	cd server/apps/document-service && go build -o ../../bin/document-service ./cmd/server

$(BIN_DIR)/ai-service:
	@echo "Building AI Service..."
	@mkdir -p $(BIN_DIR)
	cd server/apps/ai-service && go build -o ../../bin/ai-service ./cmd/server

$(BIN_DIR)/websocket-service:
	@echo "Building WebSocket Service..."
	@mkdir -p $(BIN_DIR)
	cd server/apps/websocket-service && go build -o ../../bin/websocket-service ./cmd/server

### Client ###
client:
	@echo "Starting client..."
	$(TERM_CMD) "cd client && npm run dev; exec bash"

### Run Services ###
gateway:
	@echo "Starting Gateway service..."
	$(TERM_CMD) "cd $(BIN_DIR) && ./gateway; exec bash"

auth-service:
	@echo "Starting Auth service..."
	$(TERM_CMD) "cd $(BIN_DIR) && ./auth-service; exec bash"

document-service:
	@echo "Starting Document service..."
	$(TERM_CMD) "cd $(BIN_DIR) && ./document-service; exec bash"

ai-service:
	@echo "Starting AI service..."
	$(TERM_CMD) "cd $(BIN_DIR) && ./ai-service; exec bash"

websocket-service:
	@echo "Starting WebSocket service..."
	$(TERM_CMD) "cd $(BIN_DIR) && ./websocket-service; exec bash"

### Run all services ###
dev: build client gateway auth-service document-service ai-service websocket-service
	@echo "🚀 All services started in separate terminals"
