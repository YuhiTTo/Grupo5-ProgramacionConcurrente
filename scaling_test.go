package main

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}

func TestCalculateScalingParametersUsesTrainOnly(t *testing.T) {
	train := []Sample{
		{LengthOfStay: 1, Severity: 1, Target: 10},
		{LengthOfStay: 3, Severity: 3, Target: 30},
	}

	params, err := calculateScalingParameters(train)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if !almostEqual(params.LengthOfStayMean, 2) || !almostEqual(params.LengthOfStayStd, 1) {
		t.Errorf("Length of Stay mean/std = %v/%v, se esperaba 2/1", params.LengthOfStayMean, params.LengthOfStayStd)
	}

	if !almostEqual(params.SeverityMean, 2) || !almostEqual(params.SeverityStd, 1) {
		t.Errorf("Severity mean/std = %v/%v, se esperaba 2/1", params.SeverityMean, params.SeverityStd)
	}

	if !almostEqual(params.TargetMean, 20) || !almostEqual(params.TargetStd, 10) {
		t.Errorf("Target mean/std = %v/%v, se esperaba 20/10", params.TargetMean, params.TargetStd)
	}

	// The test split is scaled with the TRAIN parameters, not its own.
	test := []Sample{{LengthOfStay: 4, Severity: 0, Target: 50}}
	applyScaling(test, params)

	if !almostEqual(test[0].LengthOfStay, 2) || !almostEqual(test[0].Severity, -2) || !almostEqual(test[0].Target, 3) {
		t.Errorf("escalamiento inesperado sobre test: %+v", test[0])
	}

	if !almostEqual(restoreTargetScale(test[0].Target, params), 50) {
		t.Errorf("restoreTargetScale no revierte el escalamiento del target")
	}
}

func TestCalculateScalingParametersErrors(t *testing.T) {
	if _, err := calculateScalingParameters(nil); err == nil {
		t.Error("se esperaba un error con train vacío")
	}

	constant := []Sample{
		{LengthOfStay: 2, Severity: 1, Target: 1},
		{LengthOfStay: 2, Severity: 2, Target: 2},
	}

	if _, err := calculateScalingParameters(constant); err == nil {
		t.Error("se esperaba un error por desviación estándar cero en Length of Stay")
	}
}
