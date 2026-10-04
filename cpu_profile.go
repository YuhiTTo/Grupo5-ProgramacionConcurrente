package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"
)

// runCPUProfile entrena un modelo concurrente bajo el profiler de
// CPU de Go (runtime/pprof) y persiste el resultado en
// <outDir>/cpu/cpu.prof para su análisis posterior con
// `go tool pprof`.
func runCPUProfile(
	trainData []Sample,
	featureCount int,
	learningRate float64,
	outDir string,
) error {
	const profileEpochs = 3000

	workers := runtime.NumCPU()

	fmt.Println()
	fmt.Println("======================================")
	fmt.Println(" PERFIL DE CPU - SOLO EVIDENCIA")
	fmt.Println("======================================")
	fmt.Printf("Workers: %d\n", workers)
	fmt.Printf("Épocas:  %d\n", profileEpochs)
	fmt.Println("Esta ejecución NO forma parte del benchmark oficial.")
	fmt.Println()

	cpuDir := filepath.Join(outDir, "cpu")

	if err := os.MkdirAll(cpuDir, 0o755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio de perfiles: %w", err)
	}

	profilePath := filepath.Join(cpuDir, "cpu.prof")

	profileFile, err := os.Create(profilePath)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo de perfil: %w", err)
	}
	// Safety net for early returns; the success path closes explicitly
	// below and checks the error (a second Close here is harmless).
	defer profileFile.Close()

	if err := pprof.StartCPUProfile(profileFile); err != nil {
		return fmt.Errorf("no se pudo iniciar el perfil de CPU: %w", err)
	}

	defer pprof.StopCPUProfile()

	model := newLinearRegression(featureCount)

	start := time.Now()

	trainConcurrent(
		model,
		trainData,
		profileEpochs,
		learningRate,
		workers,
		false,
	)

	duration := time.Since(start)

	pprof.StopCPUProfile()

	if err := profileFile.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el archivo de perfil: %w", err)
	}

	fmt.Println()
	fmt.Println("Perfil de CPU finalizado.")
	fmt.Printf("Duración: %v\n", duration)
	fmt.Printf("Perfil guardado en: %s\n", profilePath)
	fmt.Println("Analízalo con: go tool pprof " + profilePath)

	return nil
}
