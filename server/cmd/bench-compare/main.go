package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type benchSample struct {
	nsPerOp     float64
	bytesPerOp  float64
	allocsPerOp float64
}

type benchResult struct {
	name        string
	samples     []benchSample
	medianNs    float64
	medianBytes float64
	medianAllocs float64
}

var benchLineRegex = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+(\d+)\s+([\d\.]+)\s+ns/op(?:\s+([\d\.]+)\s+B/op)?(?:\s+([\d\.]+)\s+allocs/op)?`)

func parseBenchFile(path string) (map[string]*benchResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	results := make(map[string]*benchResult)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := benchLineRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}

		name := matches[1]
		ns, _ := strconv.ParseFloat(matches[3], 64)
		var bytes, allocs float64
		if len(matches) > 4 && matches[4] != "" {
			bytes, _ = strconv.ParseFloat(matches[4], 64)
		}
		if len(matches) > 5 && matches[5] != "" {
			allocs, _ = strconv.ParseFloat(matches[5], 64)
		}

		res, ok := results[name]
		if !ok {
			res = &benchResult{name: name}
			results[name] = res
		}
		res.samples = append(res.samples, benchSample{
			nsPerOp:     ns,
			bytesPerOp:  bytes,
			allocsPerOp: allocs,
		})
	}

	for _, res := range results {
		computeStats(res)
	}

	return results, scanner.Err()
}

func computeStats(res *benchResult) {
	if len(res.samples) == 0 {
		return
	}
	nsList := make([]float64, len(res.samples))
	bytesList := make([]float64, len(res.samples))
	allocsList := make([]float64, len(res.samples))

	for i, s := range res.samples {
		nsList[i] = s.nsPerOp
		bytesList[i] = s.bytesPerOp
		allocsList[i] = s.allocsPerOp
	}

	slices.Sort(nsList)
	slices.Sort(bytesList)
	slices.Sort(allocsList)

	n := len(nsList)
	res.medianNs = nsList[n/2]
	res.medianBytes = bytesList[n/2]
	res.medianAllocs = allocsList[n/2]
}

func formatDuration(ns float64) string {
	if ns < 1000 {
		return fmt.Sprintf("%.1f ns", ns)
	} else if ns < 1_000_000 {
		return fmt.Sprintf("%.2f µs", ns/1000)
	} else if ns < 1_000_000_000 {
		return fmt.Sprintf("%.2f ms", ns/1_000_000)
	}
	return fmt.Sprintf("%.2f s", ns/1_000_000_000)
}

func main() {
	var format string
	var failOnRegression bool
	var threshold float64

	flag.StringVar(&format, "format", "table", "Output format (table, markdown, json)")
	flag.BoolVar(&failOnRegression, "fail-on-regression", false, "Exit with non-zero code if any benchmark regresses")
	flag.Float64Var(&threshold, "threshold", 5.0, "Regression percentage threshold (default 5.0%)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: bench-compare [flags] <baseline-file> <candidate-file>\n")
		os.Exit(1)
	}

	baselinePath := args[0]
	candidatePath := args[1]

	baseline, err := parseBenchFile(baselinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing baseline file: %v\n", err)
		os.Exit(1)
	}

	candidate, err := parseBenchFile(candidatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing candidate file: %v\n", err)
		os.Exit(1)
	}

	hasRegression := false
	type compRow struct {
		name         string
		baseNs       float64
		candNs       float64
		deltaPercent float64
		baseBytes    float64
		candBytes    float64
		deltaBytes   float64
		verdict      string
	}

	var rows []compRow
	names := make([]string, 0, len(baseline))
	for name := range baseline {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		bRes := baseline[name]
		cRes, ok := candidate[name]
		if !ok {
			continue
		}

		delta := 0.0
		if bRes.medianNs > 0 {
			delta = ((cRes.medianNs - bRes.medianNs) / bRes.medianNs) * 100.0
		}

		verdict := "OK (STABLE)"
		if delta > threshold {
			verdict = fmt.Sprintf("REGRESSION (+%.1f%%)", delta)
			hasRegression = true
		} else if delta < -threshold {
			verdict = fmt.Sprintf("FASTER (%.1f%%)", delta)
		}

		rows = append(rows, compRow{
			name:         name,
			baseNs:       bRes.medianNs,
			candNs:       cRes.medianNs,
			deltaPercent: delta,
			baseBytes:    bRes.medianBytes,
			candBytes:    cRes.medianBytes,
			deltaBytes:   cRes.medianBytes - bRes.medianBytes,
			verdict:      verdict,
		})
	}

	if format == "json" {
		raw, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Println(string(raw))
	} else if format == "markdown" {
		fmt.Println("| Benchmark | Baseline (ns/op) | Candidate (ns/op) | Delta | Memory (B/op) | Verdict |")
		fmt.Println("| :--- | :--- | :--- | :--- | :--- | :--- |")
		for _, r := range rows {
			memDelta := fmt.Sprintf("%.0f B -> %.0f B", r.baseBytes, r.candBytes)
			fmt.Printf("| `%s` | %s | %s | %+.1f%% | %s | %s |\n",
				r.name, formatDuration(r.baseNs), formatDuration(r.candNs), r.deltaPercent, memDelta, r.verdict)
		}
	} else {
		fmt.Println("==================================================================================================")
		fmt.Println("                      KAIRO STATISTICAL BENCHMARK COMPARISON REPORT                       ")
		fmt.Println("==================================================================================================")
		fmt.Printf("%-44s %-14s %-14s %-12s %-14s %s\n",
			"BENCHMARK", "BASELINE", "CANDIDATE", "DELTA", "ALLOCS (B/op)", "VERDICT")
		fmt.Println("--------------------------------------------------------------------------------------------------")
		for _, r := range rows {
			memStr := fmt.Sprintf("%.0fB -> %.0fB", r.baseBytes, r.candBytes)
			fmt.Printf("%-44s %-14s %-14s %+10.1f%% %-14s %s\n",
				r.name,
				formatDuration(r.baseNs),
				formatDuration(r.candNs),
				r.deltaPercent,
				memStr,
				r.verdict,
			)
		}
		fmt.Println("==================================================================================================")
		if hasRegression {
			fmt.Printf("STATUS: ⚠️  REGRESSION DETECTED (> %.1f%% delta)\n", threshold)
		} else {
			fmt.Println("STATUS: ✅ NO REGRESSIONS (all components within statistical tolerance)")
		}
		fmt.Println("==================================================================================================")
	}

	if failOnRegression && hasRegression {
		os.Exit(2)
	}
}
