package main

import (
	"math/rand"
	"testing"
)

func TestVerifyEquivalenceWithinTolerance(t *testing.T) {
	seq := &LinearRegression{Weights: []float64{1.0, 2.0}, Bias: 0.5}
	conc := &LinearRegression{Weights: []float64{1.0 + 1e-12, 2.0}, Bias: 0.5}

	if err := verifyEquivalence(seq, conc, 1e-9); err != nil {
		t.Errorf("no se esperaba error dentro de tolerancia: %v", err)
	}
}

func TestVerifyEquivalenceExceedsTolerance(t *testing.T) {
	seq := &LinearRegression{Weights: []float64{1.0, 2.0}, Bias: 0.5}
	conc := &LinearRegression{Weights: []float64{1.1, 2.0}, Bias: 0.5}

	if err := verifyEquivalence(seq, conc, 1e-9); err == nil {
		t.Error("se esperaba un error fuera de tolerancia")
	}
}

// buildSyntheticSamples genera un dataset determinístico en memoria
// (sin depender del CSV de SPARCS, que no está disponible localmente)
// para poder entrenar y comparar secuencial vs. concurrente en tests.
func buildSyntheticSamples(n int, seed int64) []Sample {
	generator := rand.New(rand.NewSource(seed))

	samples := make([]Sample, n)

	for i := 0; i < n; i++ {
		samples[i] = Sample{
			LengthOfStay:        generator.Float64() * 10,
			LOS120Plus:          float64(i % 2),
			Severity:            generator.Float64() * 4,
			SeverityUnknown:     0,
			EmergencyDepartment: float64((i + 1) % 2),

			// Todas las categóricas en su categoría de referencia
			// (-1) para mantener el dataset sintético simple; el
			// camino de gradiente para features directas es el
			// mismo que se ejercitaría con dummies activas.
			AgeFeature:             -1,
			AdmissionFeature:       -1,
			MDCFeature:             -1,
			MedicalSurgicalFeature: -1,
			PaymentFeature:         -1,

			Target: generator.Float64() * 100,
		}
	}

	return samples
}

func TestSequentialConcurrentEquivalenceAcrossWorkerCounts(t *testing.T) {
	samples := buildSyntheticSamples(64, 42)

	const featureCount = 5
	const epochs = 25
	const learningRate = 0.01

	sequentialModel := newLinearRegression(featureCount)
	trainSequential(sequentialModel, samples, epochs, learningRate, false)

	for _, workers := range []int{1, 2, 4, 8} {
		concurrentModel := newLinearRegression(featureCount)
		trainConcurrent(concurrentModel, samples, epochs, learningRate, workers, false)

		if err := verifyEquivalence(sequentialModel, concurrentModel, 1e-9); err != nil {
			t.Errorf("workers=%d: %v", workers, err)
		}
	}
}

func TestConcurrentTrainingIsDeterministic(t *testing.T) {
	samples := buildSyntheticSamples(64, 7)

	const featureCount = 5
	const epochs = 15
	const learningRate = 0.02
	const workers = 4

	firstModel := newLinearRegression(featureCount)
	trainConcurrent(firstModel, samples, epochs, learningRate, workers, false)

	secondModel := newLinearRegression(featureCount)
	trainConcurrent(secondModel, samples, epochs, learningRate, workers, false)

	if firstModel.Bias != secondModel.Bias {
		t.Errorf("Bias difiere entre corridas: %v vs %v", firstModel.Bias, secondModel.Bias)
	}

	for i := range firstModel.Weights {
		if firstModel.Weights[i] != secondModel.Weights[i] {
			t.Errorf(
				"Weights[%d] difiere entre corridas: %v vs %v",
				i,
				firstModel.Weights[i],
				secondModel.Weights[i],
			)
		}
	}
}

// buildCategoricalSamples generates a deterministic dataset whose
// categorical features are ACTIVE (index >= 0) for most rows, so that the
// categorical gradient branches of accumulateSampleGradient/predict are
// exercised. Layout: 5 direct features, then Age 5..7, Admission 8..9,
// MDC 10..11, MedicalSurgical 12, Payment 13..14 (15 features in total).
func buildCategoricalSamples(n int, seed int64) []Sample {
	generator := rand.New(rand.NewSource(seed))

	// dummy maps i to -1 (reference category) or base..base+categories-1,
	// cycling with a different period per feature so combinations vary.
	dummy := func(i, period, categories, base int) int {
		r := (i / period) % (categories + 1)

		if r == 0 {
			return -1
		}

		return base + r - 1
	}

	samples := make([]Sample, n)

	for i := 0; i < n; i++ {
		samples[i] = Sample{
			LengthOfStay:        generator.Float64() * 10,
			LOS120Plus:          float64(i % 2),
			Severity:            generator.Float64() * 4,
			SeverityUnknown:     float64((i / 3) % 2),
			EmergencyDepartment: float64((i + 1) % 2),

			AgeFeature:             dummy(i, 1, 3, 5),
			AdmissionFeature:       dummy(i, 2, 2, 8),
			MDCFeature:             dummy(i, 3, 2, 10),
			MedicalSurgicalFeature: dummy(i, 1, 1, 12),
			PaymentFeature:         dummy(i, 5, 2, 13),

			Target: generator.Float64() * 100,
		}
	}

	return samples
}

func assertEquivalentToSequential(
	t *testing.T,
	samples []Sample,
	featureCount int,
	workers int,
) {
	t.Helper()

	const epochs = 20
	const learningRate = 0.01

	sequentialModel := newLinearRegression(featureCount)
	trainSequential(sequentialModel, samples, epochs, learningRate, false)

	concurrentModel := newLinearRegression(featureCount)
	trainConcurrent(concurrentModel, samples, epochs, learningRate, workers, false)

	if err := verifyEquivalence(sequentialModel, concurrentModel, 1e-9); err != nil {
		t.Errorf("n=%d workers=%d: %v", len(samples), workers, err)
	}
}

func TestSequentialConcurrentEquivalenceWithActiveCategoricals(t *testing.T) {
	samples := buildCategoricalSamples(97, 11)

	// Sanity check: every categorical weight is actually used.
	used := make(map[int]bool)

	for _, s := range samples {
		for _, index := range []int{
			s.AgeFeature, s.AdmissionFeature, s.MDCFeature,
			s.MedicalSurgicalFeature, s.PaymentFeature,
		} {
			if index >= 0 {
				used[index] = true
			}
		}
	}

	if len(used) != 10 {
		t.Fatalf("el fixture debería activar 10 dummies, activa %d", len(used))
	}

	for _, workers := range []int{1, 2, 4, 8} {
		assertEquivalentToSequential(t, samples, 15, workers)
	}
}
