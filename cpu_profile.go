package main

import (
	"fmt"
	"runtime"
	"time"
)

func runCPUProfile(
	trainData []Sample,
	featureCount int,
	learningRate float64,
) {
	const profileEpochs = 3000

	workers := runtime.NumCPU()

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" PERFIL DE CPU - SOLO EVIDENCIA")
	fmt.Println("======================================")
	fmt.Printf("Workers: %d\n", workers)
	fmt.Printf("Épocas:  %d\n", profileEpochs)
	fmt.Println("Esta ejecución NO forma parte del benchmark oficial.")
	fmt.Println()

	model := newLinearRegression(featureCount)

	start := time.Now()

	trainConcurrent(
		model,
		trainData,
		profileEpochs,
		learningRate,
		workers,
		false,
	)

	duration := time.Since(start)

	fmt.Println()
	fmt.Println("Perfil de CPU finalizado.")
	fmt.Printf("Duración: %v\n", duration)
}
