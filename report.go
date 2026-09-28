package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EnvironmentMeta agrupa los metadatos de la corrida usados en el
// reporte de entorno (environment.md).
type EnvironmentMeta struct {
	TrainRows    int
	FeatureCount int
	Epochs       int
	LearningRate float64
	Runs         int
	Warmup       int
	TrimFraction float64
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func formatMillis(duration time.Duration) string {
	return strconv.FormatFloat(duration.Seconds()*1000, 'f', 4, 64)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

// writeBenchmarkRunsCSV persiste cada corrida individual (secuencial
// y cada configuración concurrente) en <out>/benchmark/benchmark_runs.csv.
func writeBenchmarkRunsCSV(
	outDir string,
	sequential BenchmarkResult,
	concurrentResults []BenchmarkResult,
) (string, error) {

	dir := filepath.Join(outDir, "benchmark")

	if err := ensureDir(dir); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "benchmark_runs.csv")

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	if err := writer.Write([]string{"mode", "workers", "run", "duration_ms"}); err != nil {
		return "", err
	}

	allResults := append([]BenchmarkResult{sequential}, concurrentResults...)

	for _, result := range allResults {
		for index, duration := range result.Times {

			row := []string{
				result.Name,
				strconv.Itoa(result.Workers),
				strconv.Itoa(index + 1),
				formatMillis(duration),
			}

			if err := writer.Write(row); err != nil {
				return "", err
			}
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", err
	}

	return path, nil
}

func summaryRow(result BenchmarkResult, speedup float64, efficiency float64) []string {
	return []string{
		result.Name,
		strconv.Itoa(result.Workers),
		strconv.Itoa(len(result.Times)),
		formatMillis(result.Stats.Mean),
		formatMillis(result.Stats.TrimmedMean),
		formatMillis(result.Stats.Median),
		formatMillis(result.Stats.StdDev),
		formatMillis(result.Stats.Min),
		formatMillis(result.Stats.Max),
		formatFloat(result.Stats.CV),
		formatFloat(speedup),
		formatFloat(efficiency),
	}
}

func buildSummaryRows(sequential BenchmarkResult, concurrentResults []BenchmarkResult) [][]string {
	rows := make([][]string, 0, 1+len(concurrentResults))

	rows = append(rows, summaryRow(sequential, 1.0, 1.0))

	for _, result := range concurrentResults {
		rows = append(rows, summaryRow(result, result.Speedup, result.Efficiency))
	}

	return rows
}

var speedupSummaryHeader = []string{
	"mode", "workers", "runs", "mean_ms", "trimmed_mean_ms", "median_ms",
	"stddev_ms", "min_ms", "max_ms", "cv", "speedup", "efficiency",
}

// writeSpeedupSummaryCSV persiste el resumen estadístico y de
// speedup/eficiencia por configuración en
// <out>/benchmark/speedup_summary.csv.
func writeSpeedupSummaryCSV(
	outDir string,
	sequential BenchmarkResult,
	concurrentResults []BenchmarkResult,
) (string, error) {

	dir := filepath.Join(outDir, "benchmark")

	if err := ensureDir(dir); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "speedup_summary.csv")

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	if err := writer.Write(speedupSummaryHeader); err != nil {
		return "", err
	}

	for _, row := range buildSummaryRows(sequential, concurrentResults) {
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", err
	}

	return path, nil
}

// writeSpeedupSummaryMarkdown persiste el mismo resumen que
// writeSpeedupSummaryCSV como una tabla Markdown legible, en
// <out>/benchmark/speedup_summary.md.
func writeSpeedupSummaryMarkdown(
	outDir string,
	sequential BenchmarkResult,
	concurrentResults []BenchmarkResult,
) (string, error) {

	dir := filepath.Join(outDir, "benchmark")

	if err := ensureDir(dir); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "speedup_summary.md")

	var builder strings.Builder

	builder.WriteString("# Resumen de Speedup\n\n")
	builder.WriteString(
		"| Modo | Workers | Runs | Media (ms) | Media recortada (ms) | Mediana (ms) | Desv. Est. (ms) | Min (ms) | Max (ms) | CV | Speedup | Eficiencia |\n",
	)
	builder.WriteString(
		"|---|---|---|---|---|---|---|---|---|---|---|---|\n",
	)

	for _, row := range buildSummaryRows(sequential, concurrentResults) {
		builder.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

// writeEnvironmentReport persiste los metadatos del entorno de
// ejecución (versión de Go, plataforma, CPUs, configuración de la
// corrida) en <out>/environment.md.
func writeEnvironmentReport(outDir string, meta EnvironmentMeta) (string, error) {
	if err := ensureDir(outDir); err != nil {
		return "", err
	}

	path := filepath.Join(outDir, "environment.md")

	content := fmt.Sprintf(
		"# Entorno de Ejecución\n\n"+
			"- Go version: %s\n"+
			"- GOOS/GOARCH: %s/%s\n"+
			"- NumCPU: %d\n"+
			"- GOMAXPROCS: %d\n"+
			"- Timestamp: %s\n"+
			"- Train rows: %d\n"+
			"- Features: %d\n"+
			"- Epochs: %d\n"+
			"- Learning rate: %v\n"+
			"- Runs: %d\n"+
			"- Warmup: %d\n"+
			"- Trim fraction (por lado): %v\n",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		runtime.GOMAXPROCS(0),
		time.Now().Format(time.RFC3339),
		meta.TrainRows,
		meta.FeatureCount,
		meta.Epochs,
		meta.LearningRate,
		meta.Runs,
		meta.Warmup,
		meta.TrimFraction,
	)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

// writeResourcesCSV persiste el perfil de recursos (heap, allocs,
// GC, goroutines) por configuración en <out>/resources/resources.csv.
func writeResourcesCSV(outDir string, results []ResourceProfileResult) (string, error) {
	dir := filepath.Join(outDir, "resources")

	if err := ensureDir(dir); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "resources.csv")

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	header := []string{
		"workers", "elapsed_ms", "peak_heap_mb", "total_alloc_mb",
		"mallocs", "num_gc", "peak_goroutines",
	}

	if err := writer.Write(header); err != nil {
		return "", err
	}

	for _, result := range results {

		row := []string{
			strconv.Itoa(result.Workers),
			strconv.FormatFloat(result.AverageElapsedMS, 'f', 4, 64),
			strconv.FormatFloat(result.PeakHeapMB, 'f', 4, 64),
			strconv.FormatFloat(result.AllocatedMB, 'f', 4, 64),
			strconv.FormatFloat(result.AverageMallocs, 'f', 2, 64),
			strconv.FormatFloat(result.AverageGC, 'f', 2, 64),
			strconv.FormatFloat(result.AveragePeakGoroutines, 'f', 2, 64),
		}

		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", err
	}

	return path, nil
}

// findEquilibrium retorna el menor número de workers (orden
// ascendente) cuyo speedup alcanza threshold * (máximo speedup
// observado en la lista). found=false si no hay resultados o
// ninguna configuración alcanza el umbral.
func findEquilibrium(results []BenchmarkResult, threshold float64) (workers int, maxSpeedup float64, found bool) {
	if len(results) == 0 {
		return 0, 0, false
	}

	for _, result := range results {
		if result.Speedup > maxSpeedup {
			maxSpeedup = result.Speedup
		}
	}

	target := threshold * maxSpeedup

	sorted := append([]BenchmarkResult(nil), results...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Workers < sorted[j].Workers
	})

	for _, result := range sorted {
		if result.Speedup >= target {
			return result.Workers, maxSpeedup, true
		}
	}

	return 0, maxSpeedup, false
}

// findEfficiencyDrop retorna el menor número de workers (orden
// ascendente) cuya eficiencia cae por debajo de threshold.
// found=false si ninguna configuración cae por debajo del umbral.
func findEfficiencyDrop(results []BenchmarkResult, threshold float64) (workers int, found bool) {
	sorted := append([]BenchmarkResult(nil), results...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Workers < sorted[j].Workers
	})

	for _, result := range sorted {
		if result.Efficiency < threshold {
			return result.Workers, true
		}
	}

	return 0, false
}

// writeEquilibriumMarkdown persiste el punto de equilibrio y el
// punto de caída de eficiencia en <out>/benchmark/equilibrium.md.
func writeEquilibriumMarkdown(
	outDir string,
	equilibriumWorkers int,
	maxSpeedup float64,
	equilibriumFound bool,
	efficiencyDropWorkers int,
	efficiencyDropFound bool,
) (string, error) {

	dir := filepath.Join(outDir, "benchmark")

	if err := ensureDir(dir); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "equilibrium.md")

	var builder strings.Builder

	builder.WriteString("# Punto de Equilibrio\n\n")

	if equilibriumFound {
		fmt.Fprintf(&builder, "- Speedup máximo observado: %.4fx\n", maxSpeedup)
		fmt.Fprintf(
			&builder,
			"- Punto de equilibrio (>= 95%% del speedup máximo): %d workers\n",
			equilibriumWorkers,
		)
	} else {
		builder.WriteString("- No se encontró un punto de equilibrio con los datos disponibles.\n")
	}

	if efficiencyDropFound {
		fmt.Fprintf(
			&builder,
			"- La eficiencia cae por debajo de 0.50 a partir de: %d workers\n",
			efficiencyDropWorkers,
		)
	} else {
		builder.WriteString("- La eficiencia se mantuvo >= 0.50 en todas las configuraciones evaluadas.\n")
	}

	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}

	return path, nil
}
