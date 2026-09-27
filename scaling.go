package main

import (
	"fmt"
	"math"
)

type ScalingParameters struct {
	LengthOfStayMean float64
	LengthOfStayStd  float64

	SeverityMean float64
	SeverityStd  float64

	TargetMean float64
	TargetStd  float64
}

func calculateScalingParameters(
	trainData []Sample,
) (ScalingParameters, error) {

	if len(trainData) == 0 {
		return ScalingParameters{},
			fmt.Errorf("el conjunto de entrenamiento está vacío")
	}

	var lengthSum float64
	var severitySum float64
	var targetSum float64

	for _, sample := range trainData {
		lengthSum += sample.LengthOfStay
		severitySum += sample.Severity
		targetSum += sample.Target
	}

	count := float64(len(trainData))

	lengthMean := lengthSum / count
	severityMean := severitySum / count
	targetMean := targetSum / count

	var lengthSquaredDifference float64
	var severitySquaredDifference float64
	var targetSquaredDifference float64

	for _, sample := range trainData {

		lengthDifference :=
			sample.LengthOfStay - lengthMean

		severityDifference :=
			sample.Severity - severityMean

		targetDifference :=
			sample.Target - targetMean

		lengthSquaredDifference +=
			lengthDifference * lengthDifference

		severitySquaredDifference +=
			severityDifference * severityDifference

		targetSquaredDifference +=
			targetDifference * targetDifference
	}

	lengthStd := math.Sqrt(
		lengthSquaredDifference / count,
	)

	severityStd := math.Sqrt(
		severitySquaredDifference / count,
	)

	targetStd := math.Sqrt(
		targetSquaredDifference / count,
	)

	if lengthStd == 0 {
		return ScalingParameters{},
			fmt.Errorf(
				"Length of Stay tiene desviación estándar igual a cero",
			)
	}

	if severityStd == 0 {
		return ScalingParameters{},
			fmt.Errorf(
				"Severity tiene desviación estándar igual a cero",
			)
	}

	if targetStd == 0 {
		return ScalingParameters{},
			fmt.Errorf(
				"Total Costs tiene desviación estándar igual a cero",
			)
	}

	return ScalingParameters{
		LengthOfStayMean: lengthMean,
		LengthOfStayStd:  lengthStd,

		SeverityMean: severityMean,
		SeverityStd:  severityStd,

		TargetMean: targetMean,
		TargetStd:  targetStd,
	}, nil
}

func applyScaling(
	data []Sample,
	parameters ScalingParameters,
) {

	for index := range data {

		data[index].LengthOfStay =
			(data[index].LengthOfStay -
				parameters.LengthOfStayMean) /
				parameters.LengthOfStayStd

		data[index].Severity =
			(data[index].Severity -
				parameters.SeverityMean) /
				parameters.SeverityStd

		data[index].Target =
			(data[index].Target -
				parameters.TargetMean) /
				parameters.TargetStd
	}
}

func restoreTargetScale(
	standardizedValue float64,
	parameters ScalingParameters,
) float64 {

	return standardizedValue*
		parameters.TargetStd +
		parameters.TargetMean
}
