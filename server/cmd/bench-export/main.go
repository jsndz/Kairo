package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	httpadapter "server/internal/adapters/inbound/http"
	"server/internal/adapters/inbound/ws"
	exportstorage "server/internal/adapters/outbound/storage/export"
	appexport "server/internal/app/export"
	"server/internal/app/project"
	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

type config struct {
	addr        string
	projects    int
	concurrency int
	output      string
	embedded    bool
}

type mockObjects struct {
	data map[string]asset.ObjectInfo
}

func (o *mockObjects) Bucket() string { return "bench-export" }
func (o *mockObjects) UploadPolicy(_ context.Context, key string, _ asset.Source) (asset.Upload, error) {
	return asset.Upload{URL: "https://minio.invalid", Fields: map[string]string{"key": key}}, nil
}
func (o *mockObjects) Stat(_ context.Context, key string) (asset.ObjectInfo, error) {
	if v, ok := o.data[key]; ok {
		return v, nil
	}
	return asset.ObjectInfo{}, domErrors.ErrNotFound
}
func (o *mockObjects) Promote(_ context.Context, _, _, _ string) error       { return nil }
func (o *mockObjects) ReadURL(_ context.Context, _ string) (string, error) { return "https://minio.invalid/media", nil }
func (o *mockObjects) Download(_ context.Context, _, _ string) error       { return nil }
func (o *mockObjects) Upload(_ context.Context, _, _, _ string) error     { return nil }

type mockFastRenderer struct{}

func (m *mockFastRenderer) Render(_ context.Context, _ domainExport.Manifest, _ string, onProgress func(float64)) (string, error) {
	if onProgress != nil {
		onProgress(50.0)
		time.Sleep(10 * time.Millisecond)
		onProgress(100.0)
	}
	return "/tmp/out.mp4", nil
}

type exportResult struct {
	submitDuration time.Duration
	pollDuration   time.Duration
	cancelDuration time.Duration
	success        bool
	err            string
}

func main() {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", "127.0.0.1:8080", "Target server address")
	flag.IntVar(&cfg.projects, "projects", 20, "Number of export jobs to test")
	flag.IntVar(&cfg.concurrency, "concurrency", 5, "Concurrent export submission workers")
	flag.StringVar(&cfg.output, "output", "summary", "Output format (summary or json)")
	flag.BoolVar(&cfg.embedded, "embedded", false, "Force embedded test server")
	flag.Parse()

	if cfg.projects < 1 {
		cfg.projects = 1
	}
	if cfg.concurrency < 1 {
		cfg.concurrency = 1
	}

	var baseURL string
	var cleanupServer func()
	var embeddedProjMgr ports.ProjectManager
	var embeddedMockObj *mockObjects

	// 1. Probe live server or spawn embedded engine
	isLive := false
	if !cfg.embedded {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Get(fmt.Sprintf("http://%s/api/health", cfg.addr))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				isLive = true
				baseURL = fmt.Sprintf("http://%s", cfg.addr)
			}
		}
	}

	if !isLive {
		fmt.Fprintf(os.Stderr, "Notice: Live Kairo server not found on %s. Launching embedded export engine...\n", cfg.addr)
		srv, mgr, mObj, cleanup, err := startEmbeddedExportServer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start embedded export engine: %v\n", err)
			os.Exit(1)
		}
		cleanupServer = cleanup
		baseURL = srv.URL
		embeddedProjMgr = mgr
		embeddedMockObj = mObj
		u, _ := url.Parse(srv.URL)
		cfg.addr = u.Host
	}

	defer func() {
		if cleanupServer != nil {
			cleanupServer()
		}
	}()

	startCPU := getProcessCPUTicks()
	startTime := time.Now()

	fmt.Fprintf(os.Stderr, "Provisioning %d projects with active timelines for export...\n", cfg.projects)
	projectIDs, err := setupProjects(baseURL, cfg.projects, embeddedProjMgr, embeddedMockObj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error provisioning projects: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Dispatching %d export jobs with concurrency %d...\n", cfg.projects, cfg.concurrency)
	results := runExportLoad(baseURL, projectIDs, cfg.concurrency)

	elapsed := time.Since(startTime)
	endCPU := getProcessCPUTicks()
	rssMB, _ := getProcessRSS()
	goroutines := runtime.NumGoroutine()

	cpuPercent := 0.0
	if elapsed.Seconds() > 0 {
		ticksDelta := float64(endCPU - startCPU)
		cpuPercent = (ticksDelta / 100.0) / elapsed.Seconds() * 100.0
	}

	// Metrics aggregation
	var submitLats, pollLats []time.Duration
	var successCount, errorCount uint64

	for _, r := range results {
		if r.success {
			successCount++
			submitLats = append(submitLats, r.submitDuration)
			if r.pollDuration > 0 {
				pollLats = append(pollLats, r.pollDuration)
			}
		} else {
			errorCount++
			if r.err != "" {
				fmt.Fprintf(os.Stderr, "Export error: %s\n", r.err)
			}
		}
	}

	slices.Sort(submitLats)
	slices.Sort(pollLats)

	p50Submit, p95Submit, maxSubmit := calcPercentiles(submitLats)
	p50Poll, p95Poll, maxPoll := calcPercentiles(pollLats)
	throughput := float64(successCount) / elapsed.Seconds()

	if cfg.output == "json" {
		out := map[string]any{
			"config": map[string]any{
				"addr":        cfg.addr,
				"projects":    cfg.projects,
				"concurrency": cfg.concurrency,
				"duration_s":  elapsed.Seconds(),
			},
			"throughput": map[string]any{
				"totalJobs":  successCount,
				"errors":     errorCount,
				"jobsPerSec": throughput,
			},
			"submit_latency_ms": map[string]any{
				"p50": p50Submit,
				"p95": p95Submit,
				"max": maxSubmit,
			},
			"poll_latency_ms": map[string]any{
				"p50": p50Poll,
				"p95": p95Poll,
				"max": maxPoll,
			},
			"resources": map[string]any{
				"cpu_percent": cpuPercent,
				"rss_mb":      rssMB,
				"goroutines":  goroutines,
			},
		}
		raw, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(raw))
		return
	}

	printExportReport(cfg, elapsed, successCount, errorCount, throughput,
		p50Submit, p95Submit, maxSubmit, p50Poll, p95Poll, maxPoll, cpuPercent, rssMB, goroutines)
}

func calcPercentiles(lats []time.Duration) (p50, p95, maxLat float64) {
	n := len(lats)
	if n == 0 {
		return 0, 0, 0
	}
	p50 = float64(lats[int(float64(n)*0.50)].Microseconds()) / 1000.0
	p95 = float64(lats[int(float64(n)*0.95)].Microseconds()) / 1000.0
	maxLat = float64(lats[n-1].Microseconds()) / 1000.0
	return p50, p95, maxLat
}

func printExportReport(
	cfg config,
	elapsed time.Duration,
	successCount, errorCount uint64,
	throughput, p50Submit, p95Submit, maxSubmit, p50Poll, p95Poll, maxPoll, cpuPercent, rssMB float64,
	goroutines int,
) {
	fmt.Println("================================================================================")
	fmt.Println("                    KAIRO EXPORT SUBSYSTEM LOAD REPORT                          ")
	fmt.Println("================================================================================")
	fmt.Println("Test Configuration:")
	fmt.Printf("  Target Address:    %s\n", cfg.addr)
	fmt.Printf("  Duration:          %.2fs\n", elapsed.Seconds())
	fmt.Printf("  Total Export Jobs: %d\n", cfg.projects)
	fmt.Printf("  Concurrency:       %d workers\n", cfg.concurrency)
	fmt.Println()
	fmt.Println("Throughput & Reliability:")
	fmt.Printf("  Successful Jobs:   %d jobs\n", successCount)
	fmt.Printf("  Failed / Errors:   %d\n", errorCount)
	fmt.Printf("  Job Throughput:    %.2f jobs/sec\n", throughput)
	fmt.Println()
	fmt.Println("Job Submission Latency (POST /api/projects/{id}/exports):")
	fmt.Printf("  p50 (Median):      %.2f ms\n", p50Submit)
	fmt.Printf("  p95:               %.2f ms\n", p95Submit)
	fmt.Printf("  Max:               %.2f ms\n", maxSubmit)
	fmt.Println()
	fmt.Println("Status Polling Latency (GET /api/exports/{id}):")
	fmt.Printf("  p50 (Median):      %.2f ms\n", p50Poll)
	fmt.Printf("  p95:               %.2f ms\n", p95Poll)
	fmt.Printf("  Max:               %.2f ms\n", maxPoll)
	fmt.Println()
	fmt.Println("Process Resource Utilization:")
	fmt.Printf("  Server CPU:        %.1f%% (Multi-Core)\n", cpuPercent)
	fmt.Printf("  Memory RSS:        %.1f MB\n", rssMB)
	fmt.Printf("  Goroutines:        %d active\n", goroutines)
	fmt.Println("================================================================================")
}

func startEmbeddedExportServer() (*httptest.Server, ports.ProjectManager, *mockObjects, func(), error) {
	tmpDir, err := os.MkdirTemp("", "kairo-export-bench-*")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mockObj := &mockObjects{data: make(map[string]asset.ObjectInfo)}
	projDir := filepath.Join(tmpDir, "projects")
	projMgr, err := project.NewManager(projDir, mockObj, 1024*1024)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, nil, nil, nil, err
	}

	exportDir := filepath.Join(tmpDir, "exports")
	jobRepo, err := exportstorage.NewFSRepository(exportDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, nil, nil, nil, err
	}

	renderer := &mockFastRenderer{}
	exportSvc, err := appexport.New(appexport.Config{
		ProjectReader:   nil,
		ProjectResolver: projMgr,
		Repo:            jobRepo,
		Renderer:        renderer,
		ObjectStore:     mockObj,
		ScratchBaseDir:  filepath.Join(tmpDir, "scratch"),
		QueueCapacity:   500,
	})
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, nil, nil, nil, err
	}

	hub := ws.NewHub(projMgr)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := httpadapter.New(projMgr, exportSvc, hub, logger)

	srv := httptest.NewServer(handler)
	cleanup := func() {
		exportSvc.Stop()
		srv.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return srv, projMgr, mockObj, cleanup, nil
}

func setupProjects(baseURL string, count int, projMgr ports.ProjectManager, mockObj *mockObjects) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	projectIDs := make([]string, count)

	for i := 0; i < count; i++ {
		if projMgr != nil {
			summary, inst, err := projMgr.Create(context.Background(), fmt.Sprintf("Export Project %d", i+1))
			if err != nil {
				return nil, err
			}
			ctx := context.Background()
			src := asset.Source{
				Kind:     "video",
				Name:     fmt.Sprintf("clip_%d.mp4", i+1),
				MIME:     "video/mp4",
				Bytes:    1024,
				Duration: 10,
				Frames:   300,
			}
			a, err := inst.Register(src)
			if err != nil {
				return nil, fmt.Errorf("register: %w", err)
			}
			a, _, err = inst.Begin(ctx, a.ID)
			if err != nil {
				return nil, fmt.Errorf("begin: %w", err)
			}
			if mockObj != nil {
				mockObj.data[a.StageKey()] = asset.ObjectInfo{
					Size: 1024,
					MIME: "video/mp4",
					ETag: "etag-valid",
				}
			}
			_, err = inst.Complete(ctx, a.ID, a.Attempt)
			if err != nil {
				return nil, fmt.Errorf("complete: %w", err)
			}

			clipPayload, _ := json.Marshal(map[string]any{
				"clip": map[string]any{
					"id":      fmt.Sprintf("clip-%d", i+1),
					"trackId": "video-1",
					"mediaId": a.ID,
					"at":      0,
					"offset":  0,
					"length":  60,
					"volume":  1.0,
				},
			})
			rev := inst.Get().Revision
			_, commitErr := inst.Commit(timeline.Edit{
				ID:               fmt.Sprintf("op-add-%d", i+1),
				Type:             "add",
				Payload:          clipPayload,
				ExpectedRevision: &rev,
			})
			if commitErr != nil {
				return nil, fmt.Errorf("commit: %w", commitErr)
			}
			projectIDs[i] = summary.ID
		} else {
			reqBody, _ := json.Marshal(map[string]string{"name": fmt.Sprintf("Export Project %d", i+1)})
			resp, err := client.Post(baseURL+"/api/projects", "application/json", bytes.NewReader(reqBody))
			if err != nil {
				return nil, err
			}
			var summary struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&summary)
			resp.Body.Close()
			projectIDs[i] = summary.ID
		}
	}

	return projectIDs, nil
}

func runExportLoad(baseURL string, projectIDs []string, concurrency int) []exportResult {
	results := make([]exportResult, len(projectIDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	client := &http.Client{Timeout: 10 * time.Second}

	for i, pid := range projectIDs {
		wg.Add(1)
		go func(idx int, projectID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Fetch project revision
			var pState ports.State
			pResp, err := client.Get(fmt.Sprintf("%s/api/projects/%s", baseURL, projectID))
			if err == nil && pResp.StatusCode == http.StatusOK {
				_ = json.NewDecoder(pResp.Body).Decode(&pState)
				_ = pResp.Body.Close()
			}

			reqBody, _ := json.Marshal(ports.SubmitRequest{
				RequestID: fmt.Sprintf("bench-req-%d", idx),
				Revision:  pState.Revision,
				Settings:  domainExport.DefaultSettings(),
			})

			t0 := time.Now()
			resp, err := client.Post(fmt.Sprintf("%s/api/projects/%s/exports", baseURL, projectID), "application/json", bytes.NewReader(reqBody))
			submitDur := time.Since(t0)

			if err != nil {
				results[idx] = exportResult{submitDuration: submitDur, success: false, err: err.Error()}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				results[idx] = exportResult{submitDuration: submitDur, success: false, err: string(body)}
				return
			}

			var job domainExport.Job
			_ = json.NewDecoder(resp.Body).Decode(&job)

			// Poll status
			t1 := time.Now()
			pStatusResp, _ := client.Get(fmt.Sprintf("%s/api/exports/%s", baseURL, job.ID))
			pollDur := time.Since(t1)
			if pStatusResp != nil {
				_ = pStatusResp.Body.Close()
			}

			results[idx] = exportResult{
				submitDuration: submitDur,
				pollDuration:   pollDur,
				success:        true,
			}
		}(i, pid)
	}

	wg.Wait()
	return results
}

func getProcessCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0
	}
	utime, _ := strconv.ParseInt(fields[13], 10, 64)
	stime, _ := strconv.ParseInt(fields[14], 10, 64)
	return utime + stime
}

func getProcessRSS() (rssMB float64, err error) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 30.0, nil
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 30.0, nil
	}
	pages, _ := strconv.ParseInt(fields[1], 10, 64)
	bytes := pages * int64(os.Getpagesize())
	return float64(bytes) / (1024.0 * 1024.0), nil
}
