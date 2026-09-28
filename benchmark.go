package main

import (
	"fmt"
	"runtime"
	"time"
)

type BenchmarkResult struct {
	Name       string
	Workers    int
	Times      []time.Duration
	Stats      DurationStats
	Speedup    float64
	Efficiency float64
}

// runWarmup ejecuta `warmup` corridas de entrenamiento descartables
// antes de medir, para reducir el ruido de arranque en frío (JIT del
// runtime, caches, primer GC, etc.) en las mediciones oficiales.
func runWarmup(warmup int, train func()) {
	for run := 0; run < warmup; run++ {
		runtime.GC()
		train()
	}
}

func benchmarkSequential(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	runs int,
	warmup int,
	trimFraction float64,
) BenchmarkResult {

	trainOnce := func() {
		model := newLinearRegression(featureCount)

		trainSequential(
			model,
			trainData,
			epochs,
			learningRate,
			false,
		)
	}

	fmt.Println()
	fmt.Println("Benchmark secuencial")

	if warmup > 0 {
		fmt.Printf("  Calentamiento: %d ejecución(es) descartada(s)\n", warmup)
		runWarmup(warmup, trainOnce)
	}

	times := make([]time.Duration, 0, runs)

	for run := 1; run <= runs; run++ {

		// Reducimos interferencia de memoria entre ejecuciones.
		runtime.GC()

		start :=
			time.Now()

		trainOnce()

		duration :=
			time.Since(start)

		times = append(
			times,
			duration,
		)

		fmt.Printf(
			"  Ejecución %d/%d: %v\n",
			run,
			runs,
			duration,
		)
	}

	return BenchmarkResult{
		Name:    "Secuencial",
		Workers: 0,
		Times:   times,
		Stats:   computeStats(times, trimFraction),
	}
}

func benchmarkConcurrent(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	workerCount int,
	runs int,
	warmup int,
	trimFraction float64,
) BenchmarkResult {

	trainOnce := func() {
		model := newLinearRegression(featureCount)

		trainConcurrent(
			model,
			trainData,
			epochs,
			learningRate,
			workerCount,
			false,
		)
	}

	fmt.Printf(
		"\nBenchmark concurrente - %d worker(s)\n",
		workerCount,
	)

	if warmup > 0 {
		fmt.Printf("  Calentamiento: %d ejecución(es) descartada(s)\n", warmup)
		runWarmup(warmup, trainOnce)
	}

	times :=
		make([]time.Duration, 0, runs)

	for run := 1; run <= runs; run++ {

		runtime.GC()

		start :=
			time.Now()

		trainOnce()

		duration :=
			time.Since(start)

		times = append(
			times,
			duration,
		)

		fmt.Printf(
			"  Ejecución %d/%d: %v\n",
			run,
			runs,
			duration,
		)
	}

	return BenchmarkResult{
		Name:    "Concurrente",
		Workers: workerCount,
		Times:   times,
		Stats:   computeStats(times, trimFraction),
	}
}
