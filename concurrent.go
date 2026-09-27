package main

import (
	"fmt"
	"sync"
)

type GradientJob struct {
	ID    int
	Start int
	End   int
}

type GradientResult struct {
	JobID           int
	WeightGradients []float64
	BiasGradient    float64
	SquaredErrorSum float64
}

func gradientWorker(
	model *LinearRegression,
	trainData []Sample,
	jobs <-chan GradientJob,
	results chan<- GradientResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for job := range jobs {

		localGradients :=
			make([]float64, len(model.Weights))

		var localBiasGradient float64
		var localSquaredError float64

		for index := job.Start; index < job.End; index++ {

			errorValue, squaredError :=
				accumulateSampleGradient(
					model,
					&trainData[index],
					localGradients,
				)

			localBiasGradient += errorValue
			localSquaredError += squaredError
		}

		results <- GradientResult{
			JobID:           job.ID,
			WeightGradients: localGradients,
			BiasGradient:    localBiasGradient,
			SquaredErrorSum: localSquaredError,
		}
	}
}

func trainConcurrent(
	model *LinearRegression,
	trainData []Sample,
	epochs int,
	learningRate float64,
	workerCount int,
	verbose bool,
) {

	if len(trainData) == 0 {
		return
	}

	if workerCount < 1 {
		workerCount = 1
	}

	sampleCount :=
		float64(len(trainData))

	// Utilizamos varios jobs por worker para que
	// el trabajo pueda distribuirse de forma equilibrada.
	jobCount := workerCount * 4

	if jobCount > len(trainData) {
		jobCount = len(trainData)
	}

	if verbose {
		fmt.Println()
		fmt.Println("======================================")
		fmt.Println(" ENTRENAMIENTO CONCURRENTE")
		fmt.Println("======================================")

		fmt.Printf("Épocas:        %d\n", epochs)
		fmt.Printf("Learning rate: %.4f\n", learningRate)
		fmt.Printf("Registros:     %d\n", len(trainData))
		fmt.Printf("Features:      %d\n", len(model.Weights))
		fmt.Printf("Workers:       %d\n", workerCount)
		fmt.Printf("Jobs/época:    %d\n", jobCount)
	}

	for epoch := 1; epoch <= epochs; epoch++ {

		jobs :=
			make(chan GradientJob, jobCount)

		results :=
			make(chan GradientResult, jobCount)

		var wg sync.WaitGroup

		wg.Add(workerCount)

		for workerID := 0; workerID < workerCount; workerID++ {

			go gradientWorker(
				model,
				trainData,
				jobs,
				results,
				&wg,
			)
		}

		chunkSize :=
			(len(trainData) + jobCount - 1) /
				jobCount

		actualJobCount := 0

		for start := 0; start < len(trainData); start += chunkSize {

			end := start + chunkSize

			if end > len(trainData) {
				end = len(trainData)
			}

			jobs <- GradientJob{
				ID:    actualJobCount,
				Start: start,
				End:   end,
			}

			actualJobCount++
		}

		close(jobs)

		// Guardamos los resultados según su ID.
		// De esta manera la reducción se hace siempre
		// en el mismo orden, independientemente del
		// orden en que terminen los workers.
		partialResults :=
			make([]GradientResult, actualJobCount)

		for resultIndex := 0; resultIndex < actualJobCount; resultIndex++ {

			result := <-results

			partialResults[result.JobID] =
				result
		}

		wg.Wait()

		totalGradients :=
			make([]float64, len(model.Weights))

		var totalBiasGradient float64
		var totalSquaredError float64

		for jobID := 0; jobID < actualJobCount; jobID++ {

			partial :=
				partialResults[jobID]

			for weightIndex := range totalGradients {

				totalGradients[weightIndex] +=
					partial.WeightGradients[weightIndex]
			}

			totalBiasGradient +=
				partial.BiasGradient

			totalSquaredError +=
				partial.SquaredErrorSum
		}

		gradientScale :=
			2.0 / sampleCount

		for weightIndex := range model.Weights {

			model.Weights[weightIndex] -=
				learningRate *
					gradientScale *
					totalGradients[weightIndex]
		}

		model.Bias -=
			learningRate *
				gradientScale *
				totalBiasGradient

		trainingMSE :=
			totalSquaredError /
				sampleCount

		if verbose &&
			(epoch == 1 ||
				epoch%10 == 0 ||
				epoch == epochs) {

			fmt.Printf(
				"Época %2d/%d | MSE estandarizado: %.6f\n",
				epoch,
				epochs,
				trainingMSE,
			)
		}
	}
}
