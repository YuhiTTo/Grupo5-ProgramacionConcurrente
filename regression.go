package main

import "math"

type LinearRegression struct {
	Weights []float64
	Bias    float64
}

type RegressionMetrics struct {
	MSE  float64
	RMSE float64
	MAE  float64
	R2   float64
}

func newLinearRegression(featureCount int) *LinearRegression {
	return &LinearRegression{
		Weights: make([]float64, featureCount),
		Bias:    0,
	}
}

func (model *LinearRegression) predict(sample *Sample) float64 {
	prediction := model.Bias

	// Features numéricas directas.
	prediction += model.Weights[0] * sample.LengthOfStay
	prediction += model.Weights[1] * sample.LOS120Plus
	prediction += model.Weights[2] * sample.Severity
	prediction += model.Weights[3] * sample.SeverityUnknown
	prediction += model.Weights[4] * sample.EmergencyDepartment

	// Features categóricas One-Hot en representación compacta.
	// Un índice -1 significa que el registro pertenece
	// a la categoría de referencia.
	if sample.AgeFeature >= 0 {
		prediction += model.Weights[sample.AgeFeature]
	}

	if sample.AdmissionFeature >= 0 {
		prediction += model.Weights[sample.AdmissionFeature]
	}

	if sample.MDCFeature >= 0 {
		prediction += model.Weights[sample.MDCFeature]
	}

	if sample.MedicalSurgicalFeature >= 0 {
		prediction += model.Weights[sample.MedicalSurgicalFeature]
	}

	if sample.PaymentFeature >= 0 {
		prediction += model.Weights[sample.PaymentFeature]
	}

	return prediction
}

// Calcula el aporte de un registro al gradiente.
// Esta función luego también podrá reutilizarse
// dentro de cada worker concurrente.
func accumulateSampleGradient(
	model *LinearRegression,
	sample *Sample,
	weightGradients []float64,
) (float64, float64) {

	prediction := model.predict(sample)

	errorValue := prediction - sample.Target

	weightGradients[0] += errorValue * sample.LengthOfStay
	weightGradients[1] += errorValue * sample.LOS120Plus
	weightGradients[2] += errorValue * sample.Severity
	weightGradients[3] += errorValue * sample.SeverityUnknown
	weightGradients[4] += errorValue * sample.EmergencyDepartment

	if sample.AgeFeature >= 0 {
		weightGradients[sample.AgeFeature] += errorValue
	}

	if sample.AdmissionFeature >= 0 {
		weightGradients[sample.AdmissionFeature] += errorValue
	}

	if sample.MDCFeature >= 0 {
		weightGradients[sample.MDCFeature] += errorValue
	}

	if sample.MedicalSurgicalFeature >= 0 {
		weightGradients[sample.MedicalSurgicalFeature] += errorValue
	}

	if sample.PaymentFeature >= 0 {
		weightGradients[sample.PaymentFeature] += errorValue
	}

	return errorValue, errorValue * errorValue
}

func evaluateModel(
	model *LinearRegression,
	data []Sample,
	scaling ScalingParameters,
) RegressionMetrics {

	if len(data) == 0 {
		return RegressionMetrics{}
	}

	var squaredErrorSum float64
	var absoluteErrorSum float64

	var targetSum float64
	var targetSquaredSum float64

	for index := range data {
		sample := &data[index]

		predictionStandardized :=
			model.predict(sample)

		prediction :=
			restoreTargetScale(
				predictionStandardized,
				scaling,
			)

		actual :=
			restoreTargetScale(
				sample.Target,
				scaling,
			)

		errorValue :=
			prediction - actual

		squaredErrorSum +=
			errorValue * errorValue

		absoluteErrorSum +=
			math.Abs(errorValue)

		targetSum += actual
		targetSquaredSum += actual * actual
	}

	count := float64(len(data))

	mse := squaredErrorSum / count
	rmse := math.Sqrt(mse)
	mae := absoluteErrorSum / count

	totalVariation :=
		targetSquaredSum -
			(targetSum*targetSum)/count

	r2 := 0.0

	if totalVariation > 0 {
		r2 =
			1 -
				squaredErrorSum/
					totalVariation
	}

	return RegressionMetrics{
		MSE:  mse,
		RMSE: rmse,
		MAE:  mae,
		R2:   r2,
	}
}

func maxModelDifference(
	first *LinearRegression,
	second *LinearRegression,
) float64 {

	maxDifference :=
		math.Abs(first.Bias - second.Bias)

	for index := range first.Weights {

		difference :=
			math.Abs(
				first.Weights[index] -
					second.Weights[index],
			)

		if difference > maxDifference {
			maxDifference = difference
		}
	}

	return maxDifference
}
