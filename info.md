# Kairo — Project Overview and Skills Summary

## What This Project Is

Kairo is a real-time, AI-powered documentation and code collaboration platform (Google Docs for technical teams). It combines rich text editing, collaborative CRDT-based syncing, microservice architecture with gRPC, and optional AI copilots.

## Completed Work (from todo [x] and repo state)

- Monorepo established for microservices and Next.js frontend
- Docker and docker-compose set up for multi-service local dev
- PostgreSQL integration planned and wired into services
- Proto definitions created under `server/proto` and Go bindings generated under `server/gen`
- gRPC integrated across services with Buf-based generation
- Gateway service bridging REST (frontend) to gRPC (backend)
- WebSocket service with hub, rooms, clients (join/update/leave flow)
- Next.js client with rich editor (Tiptap) and Yjs integration for collaboration
- Document service with gRPC methods to create, load, and save deltas
- Autosave and delta storage strategy documented and implemented in `document-service`
- CI/CD scaffolding with GitHub Actions for lint/build/test of Go/gRPC
- Dockerfiles per service; local orchestration via `docker-compose`
- Ports defined for all services in root `README.md`

## Architecture at a Glance

- Frontend: Next.js app using Tiptap editor + Yjs for CRDT syncing
- Realtime: Dedicated WebSocket service managing rooms and presence
- Backend: Go microservices (`auth-service`, `document-service`, `websocket-service`, `gateway`, `ai-service` scaffold)
- Inter-service comms: gRPC + Protocol Buffers; Buf for schema and codegen
- Persistence: PostgreSQL with append-only `document_update` + periodic compaction into `current_state`
- Edge/API: Gateway translates REST from client into gRPC invocations
- Infra: Dockerized services; local dev via docker-compose; CI via GitHub Actions

## Detailed Tech Stack

- Frontend: Next.js (App Router), React, Tiptap, Yjs; TailwindCSS
- Realtime: WebSockets (custom Go service), Yjs updates over WS
- Backend: Go 1.x microservices (auth, document, ai, gateway, ws)
- RPC: gRPC, Protocol Buffers, Buf for generation and registry
- Database: PostgreSQL (Neon-compatible), schema for `Document` and `DocumentUpdate`
- Auth: JWT validation, gateway propagation (middleware in `server/pkg/interceptors`)
- Dev Tooling: TypeScript, ESLint, Prettier (client); Go tooling (linters/tests)
- CI/CD: GitHub Actions workflows (lint/build/test), container images
- Deployment Targets: Railway / Fly.io (scripts pending)
- Containerization: Dockerfiles per service, docker-compose for orchestration

## Key Design Choices (from design docs)

- Load snapshots via REST→gRPC; stream updates via WebSocket
- Store content deltas append-only; periodically merge into snapshot for fast reads
- Keep metadata updates off the WS path; use REST/gRPC endpoints for rename, etc.
- Avoid returning full document state on small metadata updates
- Index by `doc_id` and compact/merge deltas to control growth

## Core Features Targeted

- Rich text editing, code blocks, inline media, links, tables, undo/redo
- Real-time collaboration with multiple cursors and live updates
- Basic version history and eventual advanced features (comments, export, etc.)
- AI assistants for summarize/rewrite; queued processing (future)

## Demonstrated Skills and Expertise

- Microservice architecture design and implementation in Go
- gRPC API design, schema management with Buf, and multi-language codegen
- WebSocket architecture: hub/room/client patterns; low-latency broadcast
- CRDT-based collaboration using Yjs; state encoding/decoding and delta flows
- Database design for event-sourced document storage (delta + snapshot)
- Next.js App Router UX with rich text editor (Tiptap) integration
- AuthN/AuthZ patterns with JWT and gateway-level enforcement
- CI/CD setup for polyglot monorepo; containerized services; developer workflows
- Systems design trade-offs (REST vs WS, snapshot vs delta, compaction cadence)

## Service Map and Ports

- `ai-service`: 3003 (gRPC)
- `client`: 3000 (Next.js)
- `auth-service`: 3001 (gRPC)
- `document-service`: 3002 (gRPC)
- `websocket-service`: 3004 (WS)
- `gateway`: 8080 (REST → gRPC)

## Next Steps (high level)

- Presence/Awareness API in WS (cursor presence, colors)
- AI service endpoints and gateway integration
- Sharing/permissions (RBAC) and public links
- Comments, export/import, version history UX
- Deployment scripts for Railway/Fly.io and production readiness
