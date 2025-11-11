TERM_CMD = gnome-terminal --title

BIN_DIR = server/bin

.PHONY: all dev build client gateway auth_service doc_service ai_service ws_service stop

all: build

build: $(BIN_DIR)/gateway $(BIN_DIR)/auth_service $(BIN_DIR)/doc_service $(BIN_DIR)/ai_service $(BIN_DIR)/ws_service
	@echo "All Go services built."

$(BIN_DIR)/gateway:
	@mkdir -p $(BIN_DIR)
	cd server/apps/gateway && go build -o ../../bin/gateway ./cmd/server

$(BIN_DIR)/auth_service:
	@mkdir -p $(BIN_DIR)
	cd server/apps/auth-service && go build -o ../../bin/auth_service ./cmd/server

$(BIN_DIR)/doc_service:
	@mkdir -p $(BIN_DIR)
	cd server/apps/document-service && go build -o ../../bin/doc_service ./cmd/server

$(BIN_DIR)/ai_service:
	@mkdir -p $(BIN_DIR)
	cd server/apps/ai-service && go build -o ../../bin/ai_service ./cmd/server

$(BIN_DIR)/ws_service:
	@mkdir -p $(BIN_DIR)
	cd server/apps/websocket-service && go build -o ../../bin/ws_service ./cmd/server

client:
	$(TERM_CMD)="client" -- bash -c "cd client && npm run dev; exec bash"

define run_service
	@if pgrep -x "$(1)" >/dev/null; then \
		echo "$(1) is already running."; \
	else \
		$(TERM_CMD)="$(1)" -- bash -c "cd $(BIN_DIR) && ./$(1)"; \
	fi
endef


gateway:
	$(call run_service,gateway)

auth_service:
	$(call run_service,auth_service)

doc_service:
	$(call run_service,doc_service)

ai_service:
	$(call run_service,ai_service)

ws_service:
	$(call run_service,ws_service)

stop:
	@pkill -x gateway || true
	@pkill -x auth_service || true
	@pkill -x doc_service || true
	@pkill -x ai_service || true
	@pkill -x ws_service || true
	@echo "All services stopped."

dev: build client gateway auth_service doc_service ai_service ws_service
	@echo "Services started."
