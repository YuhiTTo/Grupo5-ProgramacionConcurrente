package main

import (
	"errors"
	"flag"
	"fmt"
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

	if config.Mode == "clean" {
		runCleaning()
		return
	}

	runPipeline(config)
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
		runCPUProfile(
			trainData,
			featureSchema.FeatureCount,
			learningRate,
		)

		return
	}

	if config.Mode == "quick" || config.Mode == "all" {
		runQuickComparison(
			trainData,
			testData,
			featureSchema,
			scalingParameters,
			config,
			learningRate,
		)
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
) {

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
			)

		result.Speedup =
			sequentialBenchmark.TrimmedMean.Seconds() /
				result.TrimmedMean.Seconds()

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
		sequentialBenchmark.TrimmedMean,
	)

	for _, result := range benchmarkResults {

		fmt.Printf(
			"%2d workers | Media: %v | Speedup: %.4fx | Eficiencia: %.4f\n",
			result.Workers,
			result.TrimmedMean,
			result.Speedup,
			result.Efficiency,
		)
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
}
