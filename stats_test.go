package main

import (
	"math"
	"testing"
	"time"
)

func TestTrimmedMeanBasic(t *testing.T) {
	times := []time.Duration{
		100 * time.Millisecond,
		100 * time.Millisecond,
		100 * time.Millisecond,
		100 * time.Millisecond,
		1000 * time.Millisecond,
	}

	got := trimmedMean(times, 0.2)
	want := 100 * time.Millisecond

	if got != want {
		t.Errorf("trimmedMean = %v, se esperaba %v", got, want)
	}
}

func TestTrimmedMeanFallsBackToMeanWhenNothingLeft(t *testing.T) {
	times := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
	}

	// fraction=0.5 sobre n=3 recorta 1 por lado, dejando 1 elemento;
	// eso sigue siendo válido. Usamos fraction=0.9 para forzar el
	// caso donde el recorte dejaría 0 o menos elementos.
	got := trimmedMean(times, 0.9)

	total := 1*time.Millisecond + 2*time.Millisecond + 3*time.Millisecond
	want := total / 3

	if got != want {
		t.Errorf("trimmedMean = %v, se esperaba media simple %v", got, want)
	}
}

func TestTrimmedMeanNoTrim(t *testing.T) {
	times := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}

	got := trimmedMean(times, 0)
	want := 20 * time.Millisecond

	if got != want {
		t.Errorf("trimmedMean = %v, se esperaba %v", got, want)
	}
}

func TestTrimmedMeanEmpty(t *testing.T) {
	got := trimmedMean(nil, 0.1)

	if got != 0 {
		t.Errorf("trimmedMean(nil) = %v, se esperaba 0", got)
	}
}

func TestComputeStatsKnownInput(t *testing.T) {
	times := []time.Duration{
		time.Duration(100),
		time.Duration(100),
		time.Duration(100),
		time.Duration(100),
		time.Duration(1000),
	}

	stats := computeStats(times, 0.2)

	if stats.Mean != time.Duration(280) {
		t.Errorf("Mean = %v, se esperaba %v", stats.Mean, time.Duration(280))
	}

	if stats.TrimmedMean != time.Duration(100) {
		t.Errorf("TrimmedMean = %v, se esperaba %v", stats.TrimmedMean, time.Duration(100))
	}

	if stats.Median != time.Duration(100) {
		t.Errorf("Median = %v, se esperaba %v", stats.Median, time.Duration(100))
	}

	if stats.Min != time.Duration(100) {
		t.Errorf("Min = %v, se esperaba %v", stats.Min, time.Duration(100))
	}

	if stats.Max != time.Duration(1000) {
		t.Errorf("Max = %v, se esperaba %v", stats.Max, time.Duration(1000))
	}

	expectedStdDev := math.Sqrt(162000.0)

	if math.Abs(float64(stats.StdDev)-expectedStdDev) > 1.0 {
		t.Errorf("StdDev = %v, se esperaba ~%v ns", stats.StdDev, expectedStdDev)
	}

	// stats.StdDev se trunca a un time.Duration (int64 ns), por lo
	// que el CV derivado pierde algo de precisión frente al valor
	// calculado directamente en punto flotante; toleramos esa
	// cuantización con un delta más amplio.
	expectedCV := expectedStdDev / 280.0

	if math.Abs(stats.CV-expectedCV) > 0.01 {
		t.Errorf("CV = %v, se esperaba ~%v", stats.CV, expectedCV)
	}
}

func TestComputeStatsSingleValue(t *testing.T) {
	times := []time.Duration{50 * time.Millisecond}

	stats := computeStats(times, 0.1)

	if stats.Mean != 50*time.Millisecond {
		t.Errorf("Mean = %v, se esperaba %v", stats.Mean, 50*time.Millisecond)
	}

	if stats.StdDev != 0 {
		t.Errorf("StdDev = %v, se esperaba 0 con un solo valor", stats.StdDev)
	}

	if stats.CV != 0 {
		t.Errorf("CV = %v, se esperaba 0 con un solo valor", stats.CV)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	stats := computeStats(nil, 0.1)

	if stats != (DurationStats{}) {
		t.Errorf("se esperaba DurationStats vacío, obtuvo %+v", stats)
	}
}
