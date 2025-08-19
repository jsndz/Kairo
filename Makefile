# Root Makefile

TERM_CMD = gnome-terminal -- bash -c
# If you're on macOS, comment above and use:
# TERM_CMD = osascript -e 'tell application "Terminal" to do script'

.PHONY: all dev client gateway auth-service document-service ai-service websocket-service

all: dev

### Client ###
client:
	@echo "Starting client..."
	$(TERM_CMD) "cd client && npm run dev; exec bash"

### Go Services ###
gateway:
	@echo "Starting Gateway service..."
	$(TERM_CMD) "cd server/apps/gateway && go run cmd/server/main.go; exec bash"

auth-service:
	@echo "Starting Auth service..."
	$(TERM_CMD) "cd server/apps/auth-service && go run cmd/server/main.go; exec bash"

document-service:
	@echo "Starting Document service..."
	$(TERM_CMD) "cd server/apps/document-service && go run cmd/server/main.go; exec bash"

ai-service:
	@echo "Starting AI service..."
	$(TERM_CMD) "cd server/apps/ai-service && go run cmd/server/main.go; exec bash"

websocket-service:
	@echo "Starting WebSocket service..."
	$(TERM_CMD) "cd server/apps/websocket-service && go run cmd/server/main.go; exec bash"

### Run all services in separate terminals ###
dev: client gateway auth-service document-service ai-service websocket-service
	@echo "All services started in separate terminals "
