package main

import (
	"fmt"
	"runtime"
	"time"
)

const cleanDatasetPath = "dataset/SPARCS_2022_clean_go.csv"

func main() {
	fmt.Println(" PC2 - REGRESIÓN LINEAL")

	datasetInfo, err := inspectCleanDataset(cleanDatasetPath)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("\nRegistros encontrados: %d\n", datasetInfo.RowCount)
	fmt.Printf("Columnas encontradas:  %d\n", len(datasetInfo.Columns))

	fmt.Println("\nAnalizando categorías del dataset completo...")

	preprocessingSummary, err :=
		analyzePreprocessingSchema(cleanDatasetPath)

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
			cleanDatasetPath,
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

	const epochs = 100
	const learningRate = 0.01

	sequentialModel :=
		newLinearRegression(
			featureSchema.FeatureCount,
		)

	trainingStart :=
		time.Now()

	trainSequential(
		sequentialModel,
		trainData,
		epochs,
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
		epochs,
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

	const benchmarkRuns = 7

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" BENCHMARK FORMAL")
	fmt.Println("======================================")

	fmt.Printf(
		"Ejecuciones por configuración: %d\n",
		benchmarkRuns,
	)

	fmt.Printf(
		"Épocas por ejecución: %d\n",
		epochs,
	)

	fmt.Printf(
		"CPU lógicas disponibles: %d\n",
		runtime.NumCPU(),
	)

	sequentialBenchmark :=
		benchmarkSequential(
			trainData,
			featureSchema.FeatureCount,
			epochs,
			learningRate,
			benchmarkRuns,
		)

	workerConfigurations :=
		[]int{
			1,
			2,
			4,
			8,
			12,
			16,
			24,
			32,
		}

	benchmarkResults :=
		make(
			[]BenchmarkResult,
			0,
			len(workerConfigurations),
		)

	for _, workers := range workerConfigurations {

		result :=
			benchmarkConcurrent(
				trainData,
				featureSchema.FeatureCount,
				epochs,
				learningRate,
				workers,
				benchmarkRuns,
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
