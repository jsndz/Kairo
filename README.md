# Kairo

Kairo is a collaborative video editor. The browser previews local media files immediately and uploads originals directly to MinIO. The Go backend enforces timeline validation, manages append-only operation logs, and provides real-time collaboration with clip ownership locks and two-party structural consensus using **Hexagonal Architecture (Ports & Adapters)**.

## Run locally

Install Docker Compose, Go (see `server/go.mod`), and Node.js compatible with the Vite version in `client`.

```sh
docker compose up -d --build minio
(cd server && go run ./cmd/kairo-server)
```

In another terminal:

```sh
cd client
npm ci
npm run dev
```

Open the address Vite prints. The development server and preview proxy `/api` to Go. Compose builds the pinned [upstream MinIO Community release](https://github.com/minio/minio/releases/tag/RELEASE.2025-10-15T17-29-55Z) and keeps originals in its named volume.

| Setting | Default |
|---|---|
| `KAIRO_ADDR` | `127.0.0.1:8080` |
| `KAIRO_DATA_DIR` | `data`, relative to the server working directory |
| `KAIRO_MAX_UPLOAD_BYTES` | `262144000` (250 MiB) |
| `KAIRO_API_TARGET` | `http://127.0.0.1:8080` for the Vite proxy |
| `MINIO_ENDPOINT` | `127.0.0.1:9000`, host and optional port |
| `MINIO_PUBLIC_ENDPOINT` | Same as `MINIO_ENDPOINT`, as reached by the browser |
| `MINIO_TLS` | `false` |
| `MINIO_BUCKET` | `kairo-media` |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | `kairo-dev` / `kairo-dev-secret` |
| `KAIRO_WEB_ORIGINS` | Vite development and preview origins on localhost and 127.0.0.1 |

Use matching credentials in Go and Compose. If you change the frontend origin, update Compose CORS. Go binds to loopback by default and has no authentication. Use one server process per data directory.

## How it stores data

- `metadata.json` stores the project identity and asset catalog. Go reads it into memory and replaces the file atomically when an asset changes.
- `operations.jsonl` stores accepted edits, including undo and redo, in sequence. Go flushes each record before returning success and replays the log at startup. Recovery removes an unterminated final record. A malformed complete line or invalid sequence stops startup and preserves the log.
- MinIO stores originals by stable object key. Uploads use short-lived, size-bound POST policies for staging keys. Go checks the object size and MIME type before a conditional server-side copy to the permanent key. Browser-reported media duration and dimensions are used for timeline metadata. Staging objects have a one-day expiration rule.
- IndexedDB caches user-selected files for fast local playback and upload retry. Completed assets can play from MinIO when the local file is absent.

Keep both the data directory and the MinIO volume to recover a project with its media.

## Architecture

### Backend: Hexagonal Architecture (Ports & Adapters)

The backend is structured using **Hexagonal Architecture (Ports & Adapters)** to decouple networking transports from domain and collaboration logic:

```
server/
├── cmd/
│   └── kairo-server/
│       └── main.go                         # Single Composition Root
│
├── internal/
│   ├── domain/                             # Pure business entities & invariants
│   │   ├── timeline/                       # Timeline, Clip, Track, Operations
│   │   ├── asset/                          # Asset entity, metadata
│   │   ├── record/                         # Log record definition
│   │   ├── history/                        # Undo/redo stack
│   │   └── errors/                         # Domain sentinel errors
│   │
│   ├── ports/                              # Clean Go interface contracts
│   │   ├── project.go                      # ProjectService, OperationService, State
│   │   ├── asset.go                        # AssetService, AssetManager
│   │   ├── collaboration.go                # CollaborationSession, ConsensusCoordinator
│   │   └── storage.go                      # CatalogRepo, OpLogRepo, ObjectStore
│   │
│   ├── app/                                # Application use cases
│   │   ├── project/                        # Project service (timeline commits, asset lifecycle)
│   │   └── collaboration/                  # RoomSession, Consensus, Ownership locks
│   │
│   └── adapters/                           # Transport & infrastructure adapters
│       ├── inbound/
│       │   ├── http/                       # REST HTTP handlers & JSON encoders
│       │   └── ws/                         # WebSocket transport (gorilla/websocket adapter)
│       └── outbound/
│           └── storage/                    # Persistence adapters (catalog, jsonl, minio, fs)
```

### Frontend: Clean Domain-Hook-View Architecture

The frontend (`client/src/features/editor/`) strictly separates pure TypeScript math and domain logic from React UI markup:

```
client/src/features/editor/
├── domain/                                 # Pure TypeScript domain logic (Zero React / DOM dependencies)
│   ├── timeline/                           # Scale math, tick step calculations, drag transforms, trim deltas
│   ├── media/                              # Asset resolution & search/kind filters
│   └── export/                             # Project serialization & file download helpers
│
├── hooks/                                  # Headless React controller hooks (No JSX)
│   ├── useTimelineScale.ts                 # Scale, extent, and ruler ticks
│   ├── useTimelineDrag.ts                  # @dnd-kit sensors and drag-commit logic
│   ├── useTrimHandle.ts                    # Pointer-capture edge trimming with rAF throttling
│   ├── useRulerScrub.ts                    # Playhead seeking and scrubbing
│   ├── useTimelinePresence.ts              # Playhead & selection sync over WebSockets
│   ├── useKeyboardShortcuts.ts             # Global hotkeys (Undo, Redo, Play/Pause, Split, Delete)
│   ├── useAvailableMedia.ts                # Local vs remote media resolution with failover
│   └── useNoticeDismiss.ts                 # Notification auto-dismiss timer
│
├── components/                             # Thin presentational React components (TSX)
│   ├── Timeline/                           # Ruler, RulerTicks, Playhead, TimelineClip, ClipContent, TrimHandle
│   ├── Preview/                            # Remotion player view, Composition canvas, TimeDisplay
│   ├── Inspector/                          # Selected clip properties & numeric inputs
│   └── MediaBin/                           # Media asset grid, search bar, upload button
│
└── state/                                  # Zustand store, optimistic dispatch, wire protocol
```

## API & Protocols

### REST API

| Method and path | Purpose |
|---|---|
| `GET /api/health` | Check the Go server health |
| `GET /api/project` | Read confirmed timeline, assets, revision, and undo/redo availability |
| `POST /api/project/operations` | Validate and commit a timeline edit |
| `POST /api/assets` | Register media metadata and assign an asset ID |
| `POST /api/assets/{id}/upload` | Start an upload attempt and return signed POST policy credentials |
| `POST /api/assets/{id}/complete` | Verify and promote the staged upload in MinIO |
| `POST /api/assets/{id}/fail` | Mark an interrupted upload attempt as failed |
| `GET /api/assets/{id}/content` | Redirect to a fresh presigned MinIO read URL |

Example edit body:

```json
{"id":"a-unique-operation-id","expectedRevision":0,"type":"rename","payload":{"name":"My cut"}}
```

Completion and failure requests use `{"attempt":"<attempt ID>"}`. The browser sends upload fields as multipart form data directly to MinIO, with the file as the last field. Clip edits reference registered asset IDs and use integer frames at 30 FPS.

### WebSocket Real-Time Sync

| Path | Purpose |
|---|---|
| `GET /api/project/ws` | Upgrade connection for real-time collaboration (`?clientId=<id>&projectId=<id>`) |

The WebSocket interface coordinates up to 2 simultaneous users per project room:
- **Presence**: Exchanges live cursor, playhead, and selection states.
- **Clip Ownership Locks**: Fine-grained claim, release, and interactive transfer requests on timeline clips.
- **Two-Party Consensus**: Requires explicit proposal and mutual approval for structural edits (`add-track`, `delete-track`, `reset`) with 30-second timers.
- **Grace Period Reclamation**: 15-second disconnect grace window before reclaiming clips held by disconnected users.

## Testing

Run tests across all packages with race detection:

```sh
(cd server && go test -race -v ./...)
(cd client && npm test && npm run build)
```
