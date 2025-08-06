FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /ws_server ./cmd/server

FROM alpine:latest

COPY --from=builder /ws_server /ws_server

EXPOSE 3004
CMD ["/ws_server"]