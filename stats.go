package main

import (
	"math"
	"sort"
	"time"
)

// DurationStats resume un conjunto de mediciones de tiempo.
type DurationStats struct {
	Mean        time.Duration
	TrimmedMean time.Duration
	Median      time.Duration
	StdDev      time.Duration
	Min         time.Duration
	Max         time.Duration
	CV          float64 // Coeficiente de variación = StdDev / Mean.
}

// trimmedMean calcula la media recortando floor(n*fraction) valores
// de cada extremo (menor y mayor) antes de promediar.
//
// fraction es la fracción recortada POR LADO: fraction=0.1 recorta
// 10% inferior + 10% superior (20% del total de mediciones). Si el
// recorte dejaría 0 o menos elementos, se usa la media simple sobre
// todos los valores.
func trimmedMean(times []time.Duration, fraction float64) time.Duration {
	if len(times) == 0 {
		return 0
	}

	sorted := sortedDurations(times)

	trimCount := int(float64(len(sorted)) * fraction)

	if trimCount*2 >= len(sorted) {
		return meanDuration(sorted)
	}

	trimmed := sorted[trimCount : len(sorted)-trimCount]

	return meanDuration(trimmed)
}

// computeStats calcula el resumen estadístico completo de un
// conjunto de mediciones (usado por el benchmark formal).
func computeStats(times []time.Duration, trimFraction float64) DurationStats {
	if len(times) == 0 {
		return DurationStats{}
	}

	sorted := sortedDurations(times)

	mean := meanDuration(sorted)
	median := medianDuration(sorted)
	stdDevNs := stdDevNanos(sorted, mean)
	stdDev := time.Duration(stdDevNs)

	// CV se calcula a partir de la desviación estándar y la media en
	// punto flotante (nanosegundos), ANTES de truncar stdDev a
	// time.Duration (int64), para no perder precisión por esa
	// cuantización a nanosegundos enteros.
	var cv float64

	if mean != 0 {
		cv = stdDevNs / float64(mean)
	}

	return DurationStats{
		Mean:        mean,
		TrimmedMean: trimmedMean(times, trimFraction),
		Median:      median,
		StdDev:      stdDev,
		Min:         sorted[0],
		Max:         sorted[len(sorted)-1],
		CV:          cv,
	}
}

func sortedDurations(times []time.Duration) []time.Duration {
	sorted := append([]time.Duration(nil), times...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return sorted
}

func meanDuration(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration

	for _, value := range times {
		total += value
	}

	return total / time.Duration(len(times))
}

func medianDuration(sorted []time.Duration) time.Duration {
	n := len(sorted)

	if n == 0 {
		return 0
	}

	if n%2 == 1 {
		return sorted[n/2]
	}

	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// stdDevNanos calcula la desviación estándar MUESTRAL (n-1 en el
// denominador) en nanosegundos, como float64 sin truncar. Con menos
// de 2 mediciones no hay variabilidad que estimar y se retorna 0.
func stdDevNanos(times []time.Duration, mean time.Duration) float64 {
	n := len(times)

	if n < 2 {
		return 0
	}

	meanNs := float64(mean)

	var sumSquaredDiff float64

	for _, value := range times {
		diff := float64(value) - meanNs
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(n-1)

	return math.Sqrt(variance)
}
