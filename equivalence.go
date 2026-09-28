package main

import "fmt"

// verifyEquivalence compara un modelo entrenado secuencialmente
// contra uno entrenado de forma concurrente y retorna un error si su
// diferencia máxima de parámetros (ver maxModelDifference) excede la
// tolerancia numérica esperada. Con tolerancias estrictas (ej.
// 1e-9) esto certifica que ambos algoritmos de entrenamiento son
// equivalentes salvo error de punto flotante.
func verifyEquivalence(seq *LinearRegression, conc *LinearRegression, tol float64) error {
	difference := maxModelDifference(seq, conc)

	if difference > tol {
		return fmt.Errorf(
			"los modelos secuencial y concurrente difieren en %.12f, por encima de la tolerancia %.12f",
			difference,
			tol,
		)
	}

	return nil
}
