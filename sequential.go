package main

import "fmt"

func trainSequential(
	model *LinearRegression,
	trainData []Sample,
	epochs int,
	learningRate float64,
	verbose bool,
) {

	if len(trainData) == 0 {
		return
	}

	sampleCount :=
		float64(len(trainData))

	if verbose {
		fmt.Println()
		fmt.Println("======================================")
		fmt.Println(" ENTRENAMIENTO SECUENCIAL")
		fmt.Println("======================================")

		fmt.Printf("Épocas:        %d\n", epochs)
		fmt.Printf("Learning rate: %.4f\n", learningRate)
		fmt.Printf("Registros:     %d\n", len(trainData))
		fmt.Printf("Features:      %d\n", len(model.Weights))
	}

	for epoch := 1; epoch <= epochs; epoch++ {

		weightGradients :=
			make(
				[]float64,
				len(model.Weights),
			)

		var biasGradient float64
		var squaredErrorSum float64

		for index := range trainData {

			sample :=
				&trainData[index]

			errorValue,
				squaredError :=
				accumulateSampleGradient(
					model,
					sample,
					weightGradients,
				)

			biasGradient += errorValue
			squaredErrorSum += squaredError
		}

		// Derivada del MSE:
		//
		// 2/n * SUM(error * feature)
		gradientScale :=
			2.0 / sampleCount

		for weightIndex := range model.Weights {

			model.Weights[weightIndex] -=
				learningRate *
					gradientScale *
					weightGradients[weightIndex]
		}

		model.Bias -=
			learningRate *
				gradientScale *
				biasGradient

		trainingMSE :=
			squaredErrorSum /
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
