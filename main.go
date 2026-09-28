package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

const cleanDatasetPath = "dataset/SPARCS_2022_clean_go.csv"

func main() {
	config, err := parseConfig(os.Args[1:])

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}

		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if config.Mode == "download" {
		if err := runDownloadMode(config); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		return
	}

	if config.Mode == "clean" {
		if err := ensureRawDatasetAvailable(config); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		runCleaning()
		return
	}

	if err := ensureCleanDatasetAvailable(config); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	runPipeline(config)
}

// runDownloadMode implementa `-mode=download`: descarga el/los
// dataset(s) seleccionados por -download (clean|raw|all).
func runDownloadMode(config Config) error {
	return runDownloadTargets(downloadTargets(config.DownloadTarget), os.Stdout)
}

// runDownloadTargets descarga cada dataset nombrado en targets usando
// el cliente HTTP real y el allowlist de producción.
func runDownloadTargets(targets []string, out io.Writer) error {
	ctx := context.Background()
	client := newDatasetHTTPClient()

	for _, name := range targets {
		spec, ok := datasetSpecs[name]

		if !ok {
			return fmt.Errorf("dataset desconocido: %q", name)
		}

		if _, err := ensureDataset(ctx, client, spec, defaultDatasetAllowlist, out); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}

	return nil
}

// downloadTargets traduce el valor de -download (clean|raw|all) a la
// lista de nombres de dataset a procesar.
func downloadTargets(target string) []string {
	switch target {
	case "raw":
		return []string{"raw"}
	case "all":
		return []string{"clean", "raw"}
	default:
		return []string{"clean"}
	}
}

// ensureCleanDatasetAvailable garantiza que el dataset limpio usado por
// el pipeline de regresión exista antes de correrlo. Si -dataset
// apunta a la ruta por defecto y el archivo falta, se descarga
// automáticamente (salvo -no-download). Una ruta -dataset
// personalizada nunca se auto-descarga: solo se valida su presencia.
func ensureCleanDatasetAvailable(config Config) error {
	if config.DatasetPath != cleanDatasetPath {
		if fileExists(config.DatasetPath) {
			return nil
		}

		return fmt.Errorf(
			"el dataset %q no existe; colóquelo manualmente en esa ruta o pase una ruta válida con -dataset",
			config.DatasetPath,
		)
	}

	return ensureNamedDatasetAvailable(cleanDatasetPath, "clean", "el dataset limpio", config.NoDownload)
}

// ensureRawDatasetAvailable garantiza que el dataset original (crudo)
// usado por `-mode=clean` exista, descargándolo automáticamente si
// falta (salvo -no-download). Su ruta no es configurable vía flag.
func ensureRawDatasetAvailable(config Config) error {
	return ensureNamedDatasetAvailable(inputCSVPath, "raw", "el dataset original", config.NoDownload)
}

// ensureNamedDatasetAvailable es el helper común: si path ya existe no
// hace nada; si falta y noDownload está activo devuelve un error
// accionable; si falta y se permite descargar, descarga el dataset
// registrado bajo specName en datasetSpecs.
func ensureNamedDatasetAvailable(path string, specName string, humanName string, noDownload bool) error {
	if fileExists(path) {
		return nil
	}

	if noDownload {
		return fmt.Errorf(
			"%s no existe en %q; ejecute 'go run . -mode=download -download=%s' para descargarlo (o quite -no-download)",
			humanName, path, specName,
		)
	}

	fmt.Printf("%s no encontrado en %q, descargando automáticamente...\n", humanName, path)

	spec, ok := datasetSpecs[specName]

	if !ok {
		return fmt.Errorf("dataset desconocido: %q", specName)
	}

	_, err := ensureDataset(
		context.Background(),
		newDatasetHTTPClient(),
		spec,
		defaultDatasetAllowlist,
		os.Stdout,
	)

	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runPipeline(config Config) {
	fmt.Println(" PC2 - REGRESIÓN LINEAL")

	datasetInfo, err := inspectCleanDataset(config.DatasetPath)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("\nRegistros encontrados: %d\n", datasetInfo.RowCount)
	fmt.Printf("Columnas encontradas:  %d\n", len(datasetInfo.Columns))

	fmt.Println("\nAnalizando categorías del dataset completo...")

	preprocessingSummary, err :=
		analyzePreprocessingSchema(config.DatasetPath)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	printCategorySummary(
		"Age Group",
		preprocessingSummary.AgeGroups,
	)

	printCategorySummary(
		"Type of Admission",
		preprocessingSummary.AdmissionTypes,
	)

	printCategorySummary(
		"APR MDC Description",
		preprocessingSummary.MajorDiagnosticGroups,
	)

	printCategorySummary(
		"APR Medical Surgical Description",
		preprocessingSummary.MedicalSurgicalGroups,
	)

	printCategorySummary(
		"Payment Typology 1",
		preprocessingSummary.PaymentTypes,
	)

	printCategorySummary(
		"Emergency Department Indicator",
		preprocessingSummary.EmergencyIndicators,
	)

	fmt.Println()
	fmt.Println(" RESUMEN DE PREPROCESAMIENTO")

	fmt.Printf(
		"Features numéricas finales esperadas: %d\n",
		preprocessingSummary.FinalFeatureCount,
	)

	featureSchema, err :=
		buildFeatureSchema(preprocessingSummary)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf(
		"\nFeatures configuradas: %d\n",
		featureSchema.FeatureCount,
	)

	fmt.Println()
	fmt.Println("Preparando Train/Test...")

	trainData, testData, err :=
		loadAndSplitDataset(
			config.DatasetPath,
			featureSchema,
			datasetInfo.RowCount,
			0.80,
			42,
		)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println()
	fmt.Println(" DATASET PARA REGRESIÓN LINEAL")

	fmt.Printf(
		"Total: %d\n",
		len(trainData)+len(testData),
	)

	fmt.Printf(
		"Train: %d (%.2f%%)\n",
		len(trainData),
		float64(len(trainData))/
			float64(len(trainData)+len(testData))*100,
	)

	fmt.Printf(
		"Test:  %d (%.2f%%)\n",
		len(testData),
		float64(len(testData))/
			float64(len(trainData)+len(testData))*100,
	)

	fmt.Printf(
		"Features: %d\n",
		featureSchema.FeatureCount,
	)

	fmt.Println()
	fmt.Println("Calculando parámetros de escalamiento...")

	scalingParameters, err :=
		calculateScalingParameters(trainData)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println()
	fmt.Println(" PARÁMETROS DE ESCALAMIENTO")

	fmt.Printf(
		"Length of Stay -> media: %.4f | std: %.4f\n",
		scalingParameters.LengthOfStayMean,
		scalingParameters.LengthOfStayStd,
	)

	fmt.Printf(
		"Severity       -> media: %.4f | std: %.4f\n",
		scalingParameters.SeverityMean,
		scalingParameters.SeverityStd,
	)

	fmt.Printf(
		"Total Costs    -> media: %.2f | std: %.2f\n",
		scalingParameters.TargetMean,
		scalingParameters.TargetStd,
	)

	applyScaling(
		trainData,
		scalingParameters,
	)

	applyScaling(
		testData,
		scalingParameters,
	)

	fmt.Println()
	fmt.Println("Escalamiento aplicado correctamente.")
	fmt.Println("Parámetros calculados únicamente con Train.")

	const learningRate = 0.01

	if config.Mode == "cpu-profile" {
		if err := runCPUProfile(
			trainData,
			featureSchema.FeatureCount,
			learningRate,
			config.OutDir,
		); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		return
	}

	environmentPath, err := writeEnvironmentReport(
		config.OutDir,
		EnvironmentMeta{
			TrainRows:    len(trainData),
			FeatureCount: featureSchema.FeatureCount,
			Epochs:       config.Epochs,
			LearningRate: learningRate,
			Runs:         config.Runs,
			Warmup:       config.Warmup,
			TrimFraction: config.TrimFraction,
		},
	)

	if err != nil {
		fmt.Println("Advertencia: no se pudo escribir environment.md:", err)
	} else {
		fmt.Printf("\nMetadatos de entorno guardados en: %s\n", environmentPath)
	}

	if config.Mode == "quick" || config.Mode == "all" {
		if err := runQuickComparison(
			trainData,
			testData,
			featureSchema,
			scalingParameters,
			config,
			learningRate,
		); err != nil {
			fmt.Println()
			fmt.Println("Error: verificación de equivalencia falló:", err)
			os.Exit(1)
		}
	}

	if config.Mode == "benchmark" || config.Mode == "all" {
		runBenchmarkMode(
			trainData,
			featureSchema,
			config,
			learningRate,
		)
	}

	if config.Mode == "resources" || config.Mode == "all" {
		runResourcesMode(
			trainData,
			featureSchema,
			config,
			learningRate,
		)
	}
}

func runQuickComparison(
	trainData []Sample,
	testData []Sample,
	featureSchema FeatureSchema,
	scalingParameters ScalingParameters,
	config Config,
	learningRate float64,
) error {

	sequentialModel :=
		newLinearRegression(
			featureSchema.FeatureCount,
		)

	trainingStart :=
		time.Now()

	trainSequential(
		sequentialModel,
		trainData,
		config.Epochs,
		learningRate,
		true,
	)

	sequentialDuration :=
		time.Since(trainingStart)

	fmt.Println()
	fmt.Println("Entrenamiento secuencial completado.")

	fmt.Printf(
		"Tiempo de entrenamiento: %v\n",
		sequentialDuration,
	)

	fmt.Println()
	fmt.Println("Evaluando modelo sobre Test...")

	sequentialMetrics :=
		evaluateModel(
			sequentialModel,
			testData,
			scalingParameters,
		)

	fmt.Println()
	fmt.Println(" RESULTADOS SECUENCIALES - TEST")

	fmt.Printf(
		"MSE:  %.2f\n",
		sequentialMetrics.MSE,
	)

	fmt.Printf(
		"RMSE: $%.2f\n",
		sequentialMetrics.RMSE,
	)

	fmt.Printf(
		"MAE:  $%.2f\n",
		sequentialMetrics.MAE,
	)

	fmt.Printf(
		"R²:   %.6f\n",
		sequentialMetrics.R2,
	)

	workerCount :=
		runtime.NumCPU()

	concurrentModel :=
		newLinearRegression(
			featureSchema.FeatureCount,
		)

	concurrentStart :=
		time.Now()

	trainConcurrent(
		concurrentModel,
		trainData,
		config.Epochs,
		learningRate,
		workerCount,
		true,
	)

	concurrentDuration :=
		time.Since(concurrentStart)

	fmt.Println()
	fmt.Println("Entrenamiento concurrente completado.")

	fmt.Printf(
		"Tiempo de entrenamiento: %v\n",
		concurrentDuration,
	)

	concurrentMetrics :=
		evaluateModel(
			concurrentModel,
			testData,
			scalingParameters,
		)

	fmt.Println()
	fmt.Println(" RESULTADOS CONCURRENTES - TEST")

	fmt.Printf(
		"MSE:  %.2f\n",
		concurrentMetrics.MSE,
	)

	fmt.Printf(
		"RMSE: $%.2f\n",
		concurrentMetrics.RMSE,
	)

	fmt.Printf(
		"MAE:  $%.2f\n",
		concurrentMetrics.MAE,
	)

	fmt.Printf(
		"R²:   %.6f\n",
		concurrentMetrics.R2,
	)

	speedup :=
		sequentialDuration.Seconds() /
			concurrentDuration.Seconds()

	modelDifference :=
		maxModelDifference(
			sequentialModel,
			concurrentModel,
		)

	fmt.Println()
	fmt.Println(" COMPARACIÓN INICIAL")

	fmt.Printf(
		"CPU lógicas disponibles: %d\n",
		runtime.NumCPU(),
	)

	fmt.Printf(
		"Tiempo secuencial:  %v\n",
		sequentialDuration,
	)

	fmt.Printf(
		"Tiempo concurrente: %v\n",
		concurrentDuration,
	)

	fmt.Printf(
		"Speedup preliminar: %.4fx\n",
		speedup,
	)

	fmt.Printf(
		"Diferencia máxima entre modelos: %.12f\n",
		modelDifference,
	)

	fmt.Printf(
		"R² secuencial:  %.6f\n",
		sequentialMetrics.R2,
	)

	fmt.Printf(
		"R² concurrente: %.6f\n",
		concurrentMetrics.R2,
	)

	const equivalenceTolerance = 1e-9

	equivalenceErr := verifyEquivalence(
		sequentialModel,
		concurrentModel,
		equivalenceTolerance,
	)

	if equivalenceErr != nil {
		fmt.Println()
		fmt.Println(" VERIFICACIÓN DE EQUIVALENCIA: FALLÓ")
		fmt.Printf("Tolerancia: %.12f\n", equivalenceTolerance)

		return equivalenceErr
	}

	fmt.Println()
	fmt.Printf(
		"Verificación de equivalencia: OK (diferencia máxima <= %.12f)\n",
		equivalenceTolerance,
	)

	return nil
}

func runBenchmarkMode(
	trainData []Sample,
	featureSchema FeatureSchema,
	config Config,
	learningRate float64,
) {

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" BENCHMARK FORMAL")
	fmt.Println("======================================")

	fmt.Printf(
		"Ejecuciones por configuración: %d\n",
		config.Runs,
	)

	fmt.Printf(
		"Épocas por ejecución: %d\n",
		config.Epochs,
	)

	fmt.Printf(
		"CPU lógicas disponibles: %d\n",
		runtime.NumCPU(),
	)

	sequentialBenchmark :=
		benchmarkSequential(
			trainData,
			featureSchema.FeatureCount,
			config.Epochs,
			learningRate,
			config.Runs,
			config.Warmup,
			config.TrimFraction,
		)

	benchmarkResults :=
		make(
			[]BenchmarkResult,
			0,
			len(config.Workers),
		)

	for _, workers := range config.Workers {

		result :=
			benchmarkConcurrent(
				trainData,
				featureSchema.FeatureCount,
				config.Epochs,
				learningRate,
				workers,
				config.Runs,
				config.Warmup,
				config.TrimFraction,
			)

		result.Speedup =
			sequentialBenchmark.Stats.TrimmedMean.Seconds() /
				result.Stats.TrimmedMean.Seconds()

		result.Efficiency =
			result.Speedup /
				float64(workers)

		benchmarkResults =
			append(
				benchmarkResults,
				result,
			)
	}

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" RESULTADOS DEL BENCHMARK")
	fmt.Println("======================================")

	fmt.Printf(
		"Secuencial | Media recortada: %v | Speedup: 1.0000x\n",
		sequentialBenchmark.Stats.TrimmedMean,
	)

	for _, result := range benchmarkResults {

		fmt.Printf(
			"%2d workers | Media: %v | Speedup: %.4fx | Eficiencia: %.4f\n",
			result.Workers,
			result.Stats.TrimmedMean,
			result.Speedup,
			result.Efficiency,
		)
	}

	persistBenchmarkResults(config.OutDir, sequentialBenchmark, benchmarkResults)
}

// persistBenchmarkResults escribe la evidencia del benchmark formal
// (corridas individuales, resumen CSV/Markdown y punto de
// equilibrio) bajo <outDir>/benchmark/. Los fallos de persistencia se
// reportan como advertencias y no abortan el benchmark, que ya
// terminó de ejecutarse y de imprimir sus resultados en stdout.
func persistBenchmarkResults(
	outDir string,
	sequentialBenchmark BenchmarkResult,
	benchmarkResults []BenchmarkResult,
) {

	fmt.Println()

	if path, err := writeBenchmarkRunsCSV(outDir, sequentialBenchmark, benchmarkResults); err != nil {
		fmt.Println("Advertencia: no se pudo escribir benchmark_runs.csv:", err)
	} else {
		fmt.Println("Corridas individuales guardadas en:", path)
	}

	if path, err := writeSpeedupSummaryCSV(outDir, sequentialBenchmark, benchmarkResults); err != nil {
		fmt.Println("Advertencia: no se pudo escribir speedup_summary.csv:", err)
	} else {
		fmt.Println("Resumen de speedup (CSV) guardado en:", path)
	}

	if path, err := writeSpeedupSummaryMarkdown(outDir, sequentialBenchmark, benchmarkResults); err != nil {
		fmt.Println("Advertencia: no se pudo escribir speedup_summary.md:", err)
	} else {
		fmt.Println("Resumen de speedup (Markdown) guardado en:", path)
	}

	equilibriumWorkers, maxSpeedup, equilibriumFound :=
		findEquilibrium(benchmarkResults, 0.95)

	efficiencyDropWorkers, efficiencyDropFound :=
		findEfficiencyDrop(benchmarkResults, 0.5)

	if equilibriumFound {
		fmt.Printf(
			"Punto de equilibrio: %d workers (speedup máximo observado: %.4fx)\n",
			equilibriumWorkers,
			maxSpeedup,
		)
	} else {
		fmt.Println("Punto de equilibrio: no se encontró con los datos disponibles.")
	}

	if efficiencyDropFound {
		fmt.Printf(
			"La eficiencia cae por debajo de 0.50 a partir de: %d workers\n",
			efficiencyDropWorkers,
		)
	}

	if path, err := writeEquilibriumMarkdown(
		outDir,
		equilibriumWorkers,
		maxSpeedup,
		equilibriumFound,
		efficiencyDropWorkers,
		efficiencyDropFound,
	); err != nil {
		fmt.Println("Advertencia: no se pudo escribir equilibrium.md:", err)
	} else {
		fmt.Println("Punto de equilibrio (Markdown) guardado en:", path)
	}
}

func runResourcesMode(
	trainData []Sample,
	featureSchema FeatureSchema,
	config Config,
	learningRate float64,
) {

	const resourceRuns = 3

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" PERFIL DE RECURSOS")
	fmt.Println("======================================")

	fmt.Printf(
		"Ejecuciones por configuración: %d\n",
		resourceRuns,
	)

	resourceResults :=
		make([]ResourceProfileResult, 0)

	sequentialResources :=
		profileSequentialResources(
			trainData,
			featureSchema.FeatureCount,
			config.Epochs,
			learningRate,
			resourceRuns,
		)

	resourceResults =
		append(
			resourceResults,
			sequentialResources,
		)

	for _, workers := range config.Workers {

		result :=
			profileConcurrentResources(
				trainData,
				featureSchema.FeatureCount,
				config.Epochs,
				learningRate,
				workers,
				resourceRuns,
			)

		resourceResults =
			append(
				resourceResults,
				result,
			)
	}

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" RESUMEN DE RECURSOS")
	fmt.Println("======================================")

	for _, result := range resourceResults {

		if result.Workers == 0 {

			fmt.Printf(
				"Secuencial | Peak Heap: %.2f MB | Alloc: %.2f MB | Mallocs: %.0f | GC: %.2f\n",
				result.PeakHeapMB,
				result.AllocatedMB,
				result.AverageMallocs,
				result.AverageGC,
			)

			continue
		}

		fmt.Printf(
			"%2d workers | Peak Heap: %.2f MB | Alloc: %.2f MB | Mallocs: %.0f | GC: %.2f\n",
			result.Workers,
			result.PeakHeapMB,
			result.AllocatedMB,
			result.AverageMallocs,
			result.AverageGC,
		)
	}

	fmt.Println()

	if path, err := writeResourcesCSV(config.OutDir, resourceResults); err != nil {
		fmt.Println("Advertencia: no se pudo escribir resources.csv:", err)
	} else {
		fmt.Println("Perfil de recursos guardado en:", path)
	}
}
