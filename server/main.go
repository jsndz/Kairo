package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"server/internal/httpapi"
	"server/internal/service"
	"server/internal/storage/catalog"
	"server/internal/storage/fs"
	"server/internal/storage/jsonl"
	"server/internal/storage/minio"
)

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func run(logger *slog.Logger) error {
	maxUpload, err := strconv.ParseInt(env("KAIRO_MAX_UPLOAD_BYTES", "262144000"), 10, 64)
	if err != nil || maxUpload <= 0 || maxUpload > 5_000_000_000 {
		return fmt.Errorf("invalid KAIRO_MAX_UPLOAD_BYTES")
	}

	tls, err := strconv.ParseBool(env("MINIO_TLS", "false"))
	if err != nil {
		return fmt.Errorf("invalid MINIO_TLS: %w", err)
	}

	endpoint := env("MINIO_ENDPOINT", "127.0.0.1:9000")
	store, err := minio.New(
		endpoint,
		env("MINIO_PUBLIC_ENDPOINT", endpoint),
		env("MINIO_ROOT_USER", "kairo-dev"),
		env("MINIO_ROOT_PASSWORD", "kairo-dev-secret"),
		env("MINIO_BUCKET", "kairo-media"),
		tls,
	)
	if err != nil {
		return err
	}

	data := env("KAIRO_DATA_DIR", "data")
	if err = os.MkdirAll(data, 0700); err != nil {
		return err
	}

	svc, err := service.Open(
		catalog.JSONCatalog{Path: filepath.Join(data, "metadata.json")},
		jsonl.JSONL{Path: filepath.Join(data, "operations.jsonl")},
		store,
		maxUpload,
	)
	if err != nil {
		return err
	}

	if err = fs.SyncDir(data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = store.Ensure(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize MinIO bucket: %w", err)
	}

	server := &http.Server{
		Addr:              env("KAIRO_ADDR", "127.0.0.1:8080"),
		Handler:           httpapi.New(svc, logger),
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-signalContext.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", "error", err)
		}
	}()

	logger.Info("Kairo ready", "address", server.Addr, "revision", svc.Get().Revision)
	if err = server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-done
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("startup failed", "error", err)
		os.Exit(1)
	}
}
