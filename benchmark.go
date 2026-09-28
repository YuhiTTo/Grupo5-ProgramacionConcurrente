package main

import (
	"fmt"
	"runtime"
	"sort"
	"time"
)

type BenchmarkResult struct {
	Name        string
	Workers     int
	Times       []time.Duration
	TrimmedMean time.Duration
	Speedup     float64
	Efficiency  float64
}

func benchmarkSequential(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	runs int,
) BenchmarkResult {

	times := make([]time.Duration, 0, runs)

	fmt.Println()
	fmt.Println("Benchmark secuencial")

	for run := 1; run <= runs; run++ {

		// Reducimos interferencia de memoria entre ejecuciones.
		runtime.GC()

		model :=
			newLinearRegression(featureCount)

		start :=
			time.Now()

		trainSequential(
			model,
			trainData,
			epochs,
			learningRate,
			false,
		)

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
		Name:        "Secuencial",
		Workers:     0,
		Times:       times,
		TrimmedMean: calculateTrimmedMean(times),
	}
}

func benchmarkConcurrent(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	workerCount int,
	runs int,
) BenchmarkResult {

	times :=
		make([]time.Duration, 0, runs)

	fmt.Printf(
		"\nBenchmark concurrente - %d worker(s)\n",
		workerCount,
	)

	for run := 1; run <= runs; run++ {

		runtime.GC()

		model :=
			newLinearRegression(featureCount)

		start :=
			time.Now()

		trainConcurrent(
			model,
			trainData,
			epochs,
			learningRate,
			workerCount,
			false,
		)

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
		Name:        "Concurrente",
		Workers:     workerCount,
		Times:       times,
		TrimmedMean: calculateTrimmedMean(times),
	}
}

func calculateTrimmedMean(
	times []time.Duration,
) time.Duration {

	if len(times) < 3 {
		var total time.Duration

		for _, value := range times {
			total += value
		}

		return total /
			time.Duration(len(times))
	}

	sortedTimes :=
		append(
			[]time.Duration(nil),
			times...,
		)

	sort.Slice(
		sortedTimes,
		func(i, j int) bool {
			return sortedTimes[i] <
				sortedTimes[j]
		},
	)

	// Eliminamos mínimo y máximo.
	trimmedTimes :=
		sortedTimes[1 : len(sortedTimes)-1]

	var total time.Duration

	for _, value := range trimmedTimes {

		total += value
	}

	return total /
		time.Duration(len(trimmedTimes))
}
