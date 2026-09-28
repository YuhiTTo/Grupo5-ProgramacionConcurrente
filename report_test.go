package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildTestBenchmarkResult(name string, workers int, timesMs []int, trimFraction float64) BenchmarkResult {
	times := make([]time.Duration, len(timesMs))

	for i, ms := range timesMs {
		times[i] = time.Duration(ms) * time.Millisecond
	}

	return BenchmarkResult{
		Name:    name,
		Workers: workers,
		Times:   times,
		Stats:   computeStats(times, trimFraction),
	}
}

func TestWriteBenchmarkRunsCSV(t *testing.T) {
	outDir := t.TempDir()

	sequential := buildTestBenchmarkResult("Secuencial", 0, []int{100, 110, 105}, 0.1)
	concurrent2 := buildTestBenchmarkResult("Concurrente", 2, []int{60, 58}, 0.1)

	path, err := writeBenchmarkRunsCSV(outDir, sequential, []BenchmarkResult{concurrent2})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	expectedPath := filepath.Join(outDir, "benchmark", "benchmark_runs.csv")
	if path != expectedPath {
		t.Errorf("path = %q, se esperaba %q", path, expectedPath)
	}

	rows := readCSV(t, path)

	if rows[0][0] != "mode" || rows[0][1] != "workers" || rows[0][2] != "run" || rows[0][3] != "duration_ms" {
		t.Errorf("header inesperado: %v", rows[0])
	}

	// 3 corridas secuenciales + 2 concurrentes = 5 filas + 1 header.
	if len(rows) != 6 {
		t.Fatalf("se esperaban 6 filas (header + 5), obtuvo %d: %v", len(rows), rows)
	}

	if rows[1][0] != "Secuencial" || rows[1][1] != "0" || rows[1][2] != "1" {
		t.Errorf("primera fila inesperada: %v", rows[1])
	}
}

func TestWriteSpeedupSummaryCSV(t *testing.T) {
	outDir := t.TempDir()

	sequential := buildTestBenchmarkResult("Secuencial", 0, []int{100, 100, 100}, 0.1)

	concurrent := buildTestBenchmarkResult("Concurrente", 2, []int{50, 50, 50}, 0.1)
	concurrent.Speedup = sequential.Stats.TrimmedMean.Seconds() / concurrent.Stats.TrimmedMean.Seconds()
	concurrent.Efficiency = concurrent.Speedup / 2

	path, err := writeSpeedupSummaryCSV(outDir, sequential, []BenchmarkResult{concurrent})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	rows := readCSV(t, path)

	expectedHeader := []string{
		"mode", "workers", "runs", "mean_ms", "trimmed_mean_ms", "median_ms",
		"stddev_ms", "min_ms", "max_ms", "cv", "speedup", "efficiency",
	}

	for i, column := range expectedHeader {
		if rows[0][i] != column {
			t.Errorf("header[%d] = %q, se esperaba %q", i, rows[0][i], column)
		}
	}

	if len(rows) != 3 {
		t.Fatalf("se esperaban 3 filas (header + secuencial + concurrente), obtuvo %d", len(rows))
	}

	if rows[1][0] != "Secuencial" || rows[1][10] != "1.000000" {
		t.Errorf("fila secuencial inesperada: %v", rows[1])
	}

	if rows[2][0] != "Concurrente" || rows[2][1] != "2" {
		t.Errorf("fila concurrente inesperada: %v", rows[2])
	}
}

func TestWriteSpeedupSummaryMarkdown(t *testing.T) {
	outDir := t.TempDir()

	sequential := buildTestBenchmarkResult("Secuencial", 0, []int{100, 100, 100}, 0.1)
	concurrent := buildTestBenchmarkResult("Concurrente", 4, []int{25, 25, 25}, 0.1)
	concurrent.Speedup = 4.0
	concurrent.Efficiency = 1.0

	path, err := writeSpeedupSummaryMarkdown(outDir, sequential, []BenchmarkResult{concurrent})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	content := readFile(t, path)

	if !strings.Contains(content, "Secuencial") {
		t.Error("el markdown debería mencionar 'Secuencial'")
	}

	if !strings.Contains(content, "Workers") {
		t.Error("el markdown debería tener una columna 'Workers'")
	}

	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		t.Error("el markdown debería iniciar con un encabezado")
	}
}

func TestWriteEnvironmentReport(t *testing.T) {
	outDir := t.TempDir()

	meta := EnvironmentMeta{
		TrainRows:    1000,
		FeatureCount: 12,
		Epochs:       100,
		LearningRate: 0.01,
		Runs:         10,
		Warmup:       1,
		TrimFraction: 0.1,
	}

	path, err := writeEnvironmentReport(outDir, meta)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	content := readFile(t, path)

	for _, expected := range []string{
		"Go version:",
		"NumCPU:",
		"GOMAXPROCS:",
		"Train rows: 1000",
		"Features: 12",
		"Epochs: 100",
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("environment report debería contener %q, contenido:\n%s", expected, content)
		}
	}
}

func TestWriteResourcesCSV(t *testing.T) {
	outDir := t.TempDir()

	results := []ResourceProfileResult{
		{
			Name:                  "Secuencial",
			Workers:               0,
			AverageElapsedMS:      120.5,
			PeakHeapMB:            10.2,
			AllocatedMB:           50.0,
			AverageMallocs:        1000,
			AverageGC:             2,
			AveragePeakGoroutines: 3,
		},
		{
			Name:                  "Concurrente",
			Workers:               4,
			AverageElapsedMS:      40.1,
			PeakHeapMB:            15.0,
			AllocatedMB:           60.0,
			AverageMallocs:        1200,
			AverageGC:             3,
			AveragePeakGoroutines: 8,
		},
	}

	path, err := writeResourcesCSV(outDir, results)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	rows := readCSV(t, path)

	expectedHeader := []string{
		"workers", "elapsed_ms", "peak_heap_mb", "total_alloc_mb",
		"mallocs", "num_gc", "peak_goroutines",
	}

	for i, column := range expectedHeader {
		if rows[0][i] != column {
			t.Errorf("header[%d] = %q, se esperaba %q", i, rows[0][i], column)
		}
	}

	if len(rows) != 3 {
		t.Fatalf("se esperaban 3 filas (header + 2 resultados), obtuvo %d", len(rows))
	}

	if rows[1][0] != "0" || rows[2][0] != "4" {
		t.Errorf("columna workers inesperada: %v / %v", rows[1], rows[2])
	}
}

func TestFindEquilibrium(t *testing.T) {
	results := []BenchmarkResult{
		{Workers: 1, Speedup: 1.0},
		{Workers: 2, Speedup: 1.8},
		{Workers: 4, Speedup: 3.2},
		{Workers: 8, Speedup: 3.3},
	}

	workers, maxSpeedup, found := findEquilibrium(results, 0.95)

	if !found {
		t.Fatal("se esperaba encontrar un punto de equilibrio")
	}

	if maxSpeedup != 3.3 {
		t.Errorf("maxSpeedup = %v, se esperaba 3.3", maxSpeedup)
	}

	// threshold*max = 0.95*3.3 = 3.135; el primer worker (ascendente)
	// que alcanza eso es 4 (speedup 3.2).
	if workers != 4 {
		t.Errorf("workers = %d, se esperaba 4", workers)
	}
}

func TestFindEquilibriumEmpty(t *testing.T) {
	_, _, found := findEquilibrium(nil, 0.95)

	if found {
		t.Error("no se esperaba encontrar un punto de equilibrio sin datos")
	}
}

func TestFindEfficiencyDrop(t *testing.T) {
	results := []BenchmarkResult{
		{Workers: 1, Efficiency: 1.0},
		{Workers: 2, Efficiency: 0.9},
		{Workers: 4, Efficiency: 0.6},
		{Workers: 8, Efficiency: 0.4},
	}

	workers, found := findEfficiencyDrop(results, 0.5)

	if !found {
		t.Fatal("se esperaba encontrar una caída de eficiencia")
	}

	if workers != 8 {
		t.Errorf("workers = %d, se esperaba 8", workers)
	}
}

func TestFindEfficiencyDropNoneBelow(t *testing.T) {
	results := []BenchmarkResult{
		{Workers: 1, Efficiency: 1.0},
		{Workers: 2, Efficiency: 0.9},
	}

	_, found := findEfficiencyDrop(results, 0.5)

	if found {
		t.Error("no se esperaba encontrar una caída de eficiencia")
	}
}

func TestWriteEquilibriumMarkdown(t *testing.T) {
	outDir := t.TempDir()

	path, err := writeEquilibriumMarkdown(outDir, 4, 3.3, true, 8, true)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	content := readFile(t, path)

	if !strings.Contains(content, "4 workers") {
		t.Errorf("se esperaba mencionar 4 workers, contenido:\n%s", content)
	}

	if !strings.Contains(content, "8 workers") {
		t.Errorf("se esperaba mencionar 8 workers, contenido:\n%s", content)
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("no se pudo abrir %s: %v", path, err)
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("no se pudo leer el CSV %s: %v", path, err)
	}

	return rows
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", path, err)
	}

	return string(content)
}
