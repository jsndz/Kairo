# Kairo

Kairo Stage 1 is a single-user editor. The timeline starts empty. The browser previews local files immediately and uploads each original directly to MinIO. Go validates edits and records them in an append-only JSONL file.

## Run locally

Install Docker Compose, Go (see `server/go.mod`), and Node.js compatible with the Vite version in `client`.

```sh
docker compose up -d --build minio
(cd server && go run .)
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

- `metadata.json` stores the project identity and asset catalog. Go reads it into maps and replaces the file atomically when an asset changes.
- `operations.jsonl` stores accepted edits, including undo and redo, in sequence. Go flushes each record before returning success and replays the log at startup. Recovery removes an unterminated final record. A malformed complete line or invalid sequence stops startup and preserves the log.
- MinIO stores originals by stable object key. Uploads use short-lived, size-bound POST policies for staging keys. Go checks the object size and MIME type before a conditional server-side copy to the permanent key. Browser-reported media duration and dimensions are used for timeline metadata. Staging objects have a one-day expiration rule.
- IndexedDB caches user-selected files for fast local playback and upload retry. Completed assets can play from MinIO when the local file is absent. Earlier browser-only projects and media stay in the old browser storage. Stage 1 starts with a new project.

Keep both the data directory and the MinIO volume to recover a project with its media.

## REST API

| Method and path | Purpose |
|---|---|
| `GET /api/health` | Check the Go server |
| `GET /api/project` | Read the confirmed timeline, assets, revision, and undo/redo availability |
| `POST /api/project/operations` | Validate and commit an edit |
| `POST /api/assets` | Register metadata and assign an asset ID |
| `POST /api/assets/{id}/upload` | Start an attempt and return the signed URL and form fields |
| `POST /api/assets/{id}/complete` | Verify and promote the staged upload |
| `POST /api/assets/{id}/fail` | Mark an interrupted attempt as failed |
| `GET /api/assets/{id}/content` | Redirect to a fresh signed MinIO read URL |

Example edit body:

```json
{"id":"a-unique-operation-id","expectedRevision":0,"type":"rename","payload":{"name":"My cut"}}
```

Completion and failure requests use `{"attempt":"<attempt ID>"}`. The browser sends the upload fields as multipart form data directly to MinIO, with the file as the last field. Clip edits reference registered asset IDs and use integer frames at 30 FPS.

Run `go test -race ./...` from `server` and `npm test && npm run build` from `client`.
