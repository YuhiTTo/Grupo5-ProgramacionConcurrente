package main

import "testing"

func TestPredictWithKnownWeights(t *testing.T) {
	model := &LinearRegression{
		Weights: []float64{1, 2, 3, 4, 5, 10, 20},
		Bias:    0.5,
	}

	sample := &Sample{
		LengthOfStay:        1,
		LOS120Plus:          0,
		Severity:            2,
		SeverityUnknown:     1,
		EmergencyDepartment: 1,

		AgeFeature:             5, // +10
		AdmissionFeature:       -1,
		MDCFeature:             -1,
		MedicalSurgicalFeature: -1,
		PaymentFeature:         6, // +20
	}

	// 0.5 + 1*1 + 2*0 + 3*2 + 4*1 + 5*1 + 10 + 20 = 46.5
	if got := model.predict(sample); !almostEqual(got, 46.5) {
		t.Errorf("predict = %v, se esperaba 46.5", got)
	}
}

func TestAccumulateSampleGradientCategoricalBranches(t *testing.T) {
	model := &LinearRegression{Weights: make([]float64, 8)}

	sample := &Sample{
		LengthOfStay:           2,
		AgeFeature:             5,
		AdmissionFeature:       -1,
		MDCFeature:             6,
		MedicalSurgicalFeature: -1,
		PaymentFeature:         7,
		Target:                 3,
	}

	gradients := make([]float64, 8)

	errorValue, squared := accumulateSampleGradient(model, sample, gradients)

	// prediction 0, target 3 -> error -3.
	if !almostEqual(errorValue, -3) || !almostEqual(squared, 9) {
		t.Errorf("error/squared = %v/%v, se esperaba -3/9", errorValue, squared)
	}

	want := []float64{-6, 0, 0, 0, 0, -3, -3, -3}
	for i := range want {
		if !almostEqual(gradients[i], want[i]) {
			t.Errorf("gradients[%d] = %v, se esperaba %v", i, gradients[i], want[i])
		}
	}
}

func TestEvaluateModelPerfectPrediction(t *testing.T) {
	model := &LinearRegression{Weights: []float64{1, 0, 0, 0, 0}, Bias: 0}

	data := []Sample{
		{LengthOfStay: 1, Target: 1, AgeFeature: -1, AdmissionFeature: -1, MDCFeature: -1, MedicalSurgicalFeature: -1, PaymentFeature: -1},
		{LengthOfStay: 3, Target: 3, AgeFeature: -1, AdmissionFeature: -1, MDCFeature: -1, MedicalSurgicalFeature: -1, PaymentFeature: -1},
	}

	metrics := evaluateModel(model, data, ScalingParameters{TargetMean: 0, TargetStd: 1})

	if !almostEqual(metrics.MSE, 0) || !almostEqual(metrics.MAE, 0) || !almostEqual(metrics.R2, 1) {
		t.Errorf("métricas inesperadas para predicción perfecta: %+v", metrics)
	}
}
