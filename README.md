# Kairo 🚀

Real-time, AI-Powered Documentation & Code Collaboration Platform (Google Docs for technical teams).

Kairo combines rich-text editing, live collaborative syncing using CRDTs (Conflict-free Replicated Data Types), a Go-based gRPC microservice architecture, and streaming AI assistant copilots. It is built to support technical teams with embedded, syntax-highlighted code blocks, automated diagrams, and ultra-low-latency real-time editing.

---

## 🏛️ System Architecture

Kairo is designed as a polyglot monorepo featuring a Next.js frontend, an API Gateway, and several specialized Go microservices communicating via gRPC. 

```mermaid
graph TD
    Client[Next.js Client<br/>Port 3000] -->|HTTP REST /api/v1| Gateway[API Gateway<br/>Port 8080]
    Client -->|WebSockets /ws| WS[WebSocket Service<br/>Port 3004]

    Gateway -->|gRPC| Auth[Auth Service<br/>Port 3001]
    Gateway -->|gRPC| Doc[Document Service<br/>Port 3002]
    Gateway -->|gRPC| AI[AI Service<br/>Port 3003]

    WS -->|gRPC| Doc
    WS -->|gRPC| AI
    AI -->|gRPC| Doc

    Auth -->|GORM / localhost:5433| AuthDB[(Auth DB<br/>PostgreSQL)]
    Doc -->|GORM / localhost:5434| DocDB[(Doc DB<br/>PostgreSQL)]

    AI -->|HTTP / localhost:11434| Ollama[Ollama / Llama]
    AI -->|HTTPS / OpenAI API| OpenAI[OpenAI / GPT-4o-mini]
```

### Services & Port Configuration

| Port | Service Name | Protocol | Technology / Purpose | Location |
| :--- | :--- | :--- | :--- | :--- |
| `3000` | `kairo_client` | HTTP | Next.js (App Router, Tailwind CSS, Tiptap, Yjs) | [client/](file:///home/jaison/code/projects/Kairo/client) |
| `8080` | `kairo_gateway_service` | HTTP | Gin Gateway (Translates REST into backend gRPC requests) | [server/apps/gateway/](file:///home/jaison/code/projects/Kairo/server/apps/gateway) |
| `3001` | `kairo_auth_service` | gRPC | Go Auth Service (JWT validation, user tables) | [server/apps/auth-service/](file:///home/jaison/code/projects/Kairo/server/apps/auth-service) |
| `3002` | `kairo_document_service`| gRPC | Go Document Service (CRUD, delta storage, compaction) | [server/apps/document-service/](file:///home/jaison/code/projects/Kairo/server/apps/document-service) |
| `3003` | `kairo_ai_service` | gRPC | Go AI Service (Summarization, rewrites, Ollama/OpenAI) | [server/apps/ai-service/](file:///home/jaison/code/projects/Kairo/server/apps/ai-service) |
| `3004` | `kairo_ws_service` | WebSocket | Go WebSocket Server (Presence, low-latency room hub) | [server/apps/websocket-service/](file:///home/jaison/code/projects/Kairo/server/apps/websocket-service) |
| `3005` | `yjs-merge-service` | HTTP | Express Node.js helper (scaffolding) | [server/apps/yjs-merge-service/](file:///home/jaison/code/projects/Kairo/server/apps/yjs-merge-service) |
| `5433` | `kairo_auth` | PostgreSQL| Postgres DB storing users & credentials | Managed via [docker-compose.yml](file:///home/jaison/code/projects/Kairo/docker-compose.yml) |
| `5434` | `kairo_doc` | PostgreSQL| Postgres DB storing documents and append-only updates | Managed via [docker-compose.yml](file:///home/jaison/code/projects/Kairo/docker-compose.yml) |

---

## ✨ Features

- **Rich Text Collaboration**: Live multi-cursor editing with presence tracking (colors/usernames) powered by [Yjs](https://yjs.dev/) CRDTs and a custom Go WebSocket hub.
- **Embedded Technical Tools**: Built-in syntax highlighting for code blocks, interactive Markdown formatting, and automated diagram previews (via Mermaid.js).
- **Two-Request Loading Approach**: Large document states are streamed as binary octet-streams to bypass expensive Hex-to-JSON serialization, while document metadata is retrieved via standard JSON API.
- **Autosave & Compaction Strategy**: Edits are persisted instantly as lightweight append-only deltas (<1ms writes). An asynchronous background job merges updates into a master snapshot to keep database reads extremely fast.
- **AI Assistent (Streaming)**: Summarize or rewrite documents in real-time. Features streaming gRPC support coupled with local Ollama model options or OpenAI APIs.

---

## 📂 Repository Structure

The project is structured as a polyglot monorepo:

```
Kairo/
├── client/                     # Next.js App Router frontend
│   ├── app/                    # UI views (auth, documents, dashboard)
│   ├── components/             # React components (Tiptap Editor, Sidebar)
│   ├── lib/                    # Collaboration utilities (Yjs providers)
│   └── services/               # Client-side API request wrappers
├── server/                     # Go microservices monorepo
│   ├── apps/                   # Individual microservice apps
│   │   ├── ai-service/         # gRPC server routing LLM tasks
│   │   ├── auth-service/       # gRPC server managing credentials
│   │   ├── document-service/   # gRPC server managing document updates & compaction
│   │   ├── gateway/            # Edge API REST gateway converting to internal gRPC
│   │   └── websocket-service/  # High-throughput collaboration WebSocket hub
│   ├── proto/                  # Protobuf schemas (ai.proto, auth.proto, doc.proto)
│   ├── gen/                    # Generated Go gRPC code bindings
│   └── pkg/                    # Shared packages (db connection, env loader, gRPC clients)
├── docs/                       # Technical design decisions and specs
├── Makefile                    # Local build & dev commands
└── docker-compose.yml          # Container configuration for external databases
```

---

## 🛠️ Getting Started

### 1. Prerequisites

- [Go](https://go.dev/) (v1.20+)
- [Node.js](https://nodejs.org/) (v18+) & `npm`
- [Docker & Docker Compose](https://docs.docker.com/engine/install/)
- (Optional) [Ollama](https://ollama.com/) running locally for offline AI features

### 2. Environment Configuration

Copy the example environment file at the root to create your `.env` configuration:

```bash
cp .env.example .env
```

Ensure the configuration covers the database connection URLs. Example contents of `.env`:

```env
AUTH_DB_URL=postgresql://postgres:kairo@localhost:5433/kairo_auth
DOC_DB_URL=postgresql://postgres:kairo@localhost:5434/kairo_doc
OPENAI_API_KEY=your_optional_openai_key
```

### 3. Spin up Databases

Use Docker Compose to launch PostgreSQL databases for the auth and document services:

```bash
docker compose up -d
```

*Note: GORM `AutoMigrate` runs automatically on service startup, creating the necessary database tables.*

### 4. Build and Run the Monorepo

The [Makefile](file:///home/jaison/code/projects/Kairo/Makefile) simplifies the compilation and execution of all services.

#### Build all Go binaries:
```bash
make build
```

#### Run the entire stack in local dev mode:
```bash
make dev
```
*This command compiles the Go binaries, starts the Next.js client dev server on port `3000`, and spawns each Go microservice in its own terminal instance.*

#### Stop all running Go microservices:
```bash
make stop
```

---

## 🧠 Technical Highlights

Deep-dive specifications are located in the [docs/](file:///home/jaison/code/projects/Kairo/docs) folder:

1. **Two-Request Document Retrieval** ([docs/two-request-approach.md](file:///home/jaison/code/projects/Kairo/docs/two-request-approach.md)):
   Avoids JSON hex encoding overhead of Yjs state vectors. The client requests JSON metadata separately from the raw document updates, which are transferred directly via `application/octet-stream`.
2. **Binary WebSockets Protocol** ([docs/Binary_Message_protocol.md](file:///home/jaison/code/projects/Kairo/docs/Binary_Message_protocol.md)):
   Client updates and presence sync are framed using binary messages instead of heavy JSON strings, using a single-byte identifier prefix (e.g. `1` for joining, `2` for delta updates).
3. **Autosave Compaction** ([docs/Design Choices.md](file:///home/jaison/code/projects/Kairo/docs/Design%20Choices.md)):
   Incremental document edits are appended to `document_update` in the database. An asynchronous loop periodically queries updates since the last snapshot, merges them with the base snapshot using Yjs CRDT rules, and saves the compacted result into `current_state`.
4. **Streaming gRPC AI Answers** ([docs/streaming.md](file:///home/jaison/code/projects/Kairo/docs/streaming.md) / [docs/gRPC.md](file:///home/jaison/code/projects/Kairo/docs/gRPC.md)):
   Ensures prompt processing is streamed chunk-by-chunk using `io.Reader` and `bufio.Scanner` channels inside `ai-service`, providing real-time visual output of summaries to the editor.

---

## 🛠️ Code Generation & Proto Schema

Protobuf compilation is managed via the [Buf CLI](https://buf.build/). 

If you make modifications to files in [server/proto/](file:///home/jaison/code/projects/Kairo/server/proto), regenerate the Go packages by executing the following within the `server/` directory:

```bash
cd server
buf generate
```
*(Configuration templates are stored in [buf.yaml](file:///home/jaison/code/projects/Kairo/server/buf.yaml) and [buf.gen.yaml](file:///home/jaison/code/projects/Kairo/server/buf.gen.yaml))*
