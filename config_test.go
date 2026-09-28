package main

import (
	"reflect"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	config, err := parseConfig(nil)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "all" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "all")
	}

	if config.Runs != 10 {
		t.Errorf("Runs = %d, se esperaba %d", config.Runs, 10)
	}

	if config.Warmup != 1 {
		t.Errorf("Warmup = %d, se esperaba %d", config.Warmup, 1)
	}

	if config.Epochs != 100 {
		t.Errorf("Epochs = %d, se esperaba %d", config.Epochs, 100)
	}

	expectedWorkers := []int{1, 2, 4, 8, 12, 16, 24, 32}

	if !reflect.DeepEqual(config.Workers, expectedWorkers) {
		t.Errorf("Workers = %v, se esperaba %v", config.Workers, expectedWorkers)
	}

	if config.TrimFraction != 0.1 {
		t.Errorf("TrimFraction = %v, se esperaba %v", config.TrimFraction, 0.1)
	}

	if config.OutDir != "results" {
		t.Errorf("OutDir = %q, se esperaba %q", config.OutDir, "results")
	}

	if config.DatasetPath != cleanDatasetPath {
		t.Errorf("DatasetPath = %q, se esperaba %q", config.DatasetPath, cleanDatasetPath)
	}
}

func TestParseConfigCustomFlags(t *testing.T) {
	config, err := parseConfig([]string{
		"-mode=benchmark",
		"-runs=5",
		"-warmup=2",
		"-epochs=50",
		"-workers=1,3,6",
		"-trim=0.2",
		"-out=myresults",
		"-dataset=custom.csv",
	})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "benchmark" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "benchmark")
	}

	if config.Runs != 5 {
		t.Errorf("Runs = %d, se esperaba %d", config.Runs, 5)
	}

	if config.Warmup != 2 {
		t.Errorf("Warmup = %d, se esperaba %d", config.Warmup, 2)
	}

	if config.Epochs != 50 {
		t.Errorf("Epochs = %d, se esperaba %d", config.Epochs, 50)
	}

	expectedWorkers := []int{1, 3, 6}

	if !reflect.DeepEqual(config.Workers, expectedWorkers) {
		t.Errorf("Workers = %v, se esperaba %v", config.Workers, expectedWorkers)
	}

	if config.TrimFraction != 0.2 {
		t.Errorf("TrimFraction = %v, se esperaba %v", config.TrimFraction, 0.2)
	}

	if config.OutDir != "myresults" {
		t.Errorf("OutDir = %q, se esperaba %q", config.OutDir, "myresults")
	}

	if config.DatasetPath != "custom.csv" {
		t.Errorf("DatasetPath = %q, se esperaba %q", config.DatasetPath, "custom.csv")
	}
}

func TestParseConfigInvalidMode(t *testing.T) {
	_, err := parseConfig([]string{"-mode=bogus"})

	if err == nil {
		t.Fatal("se esperaba un error para un modo inválido")
	}
}

func TestParseConfigInvalidWorkers(t *testing.T) {
	_, err := parseConfig([]string{"-workers=1,x,3"})

	if err == nil {
		t.Fatal("se esperaba un error para una lista de workers inválida")
	}
}

func TestParseConfigEmptyWorkers(t *testing.T) {
	_, err := parseConfig([]string{"-workers="})

	if err == nil {
		t.Fatal("se esperaba un error para una lista de workers vacía")
	}
}

func TestParseConfigWorkerWithZero(t *testing.T) {
	_, err := parseConfig([]string{"-workers=0,1"})

	if err == nil {
		t.Fatal("se esperaba un error para un worker menor a 1")
	}
}

func TestParseConfigTrimOutOfRange(t *testing.T) {
	_, err := parseConfig([]string{"-trim=0.5"})

	if err == nil {
		t.Fatal("se esperaba un error para -trim fuera de rango")
	}

	_, err = parseConfig([]string{"-trim=-0.1"})

	if err == nil {
		t.Fatal("se esperaba un error para -trim negativo")
	}
}

func TestParseConfigInvalidRunsAndEpochs(t *testing.T) {
	if _, err := parseConfig([]string{"-runs=0"}); err == nil {
		t.Error("se esperaba un error para -runs=0")
	}

	if _, err := parseConfig([]string{"-epochs=0"}); err == nil {
		t.Error("se esperaba un error para -epochs=0")
	}

	if _, err := parseConfig([]string{"-warmup=-1"}); err == nil {
		t.Error("se esperaba un error para -warmup negativo")
	}
}

func TestParseConfigLegacyCPUProfilePositional(t *testing.T) {
	config, err := parseConfig([]string{"cpu-profile"})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "cpu-profile" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "cpu-profile")
	}
}

func TestParseConfigLegacyCPUProfileWithTrailingFlags(t *testing.T) {
	config, err := parseConfig([]string{"cpu-profile", "-epochs=42"})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "cpu-profile" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "cpu-profile")
	}

	if config.Epochs != 42 {
		t.Errorf("Epochs = %d, se esperaba %d", config.Epochs, 42)
	}
}

func TestParseConfigModeFlagOverridesLegacy(t *testing.T) {
	// Un -mode explícito luego del positional legacy debe poder
	// reemplazar el modo (el positional solo fija el default).
	config, err := parseConfig([]string{"cpu-profile", "-mode=quick"})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "quick" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "quick")
	}
}

func TestParseConfigDownloadDefaults(t *testing.T) {
	config, err := parseConfig(nil)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.DownloadTarget != "clean" {
		t.Errorf("DownloadTarget = %q, se esperaba %q", config.DownloadTarget, "clean")
	}

	if config.NoDownload {
		t.Error("NoDownload = true, se esperaba false por defecto")
	}
}

func TestParseConfigModeDownload(t *testing.T) {
	config, err := parseConfig([]string{"-mode=download", "-download=raw"})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.Mode != "download" {
		t.Errorf("Mode = %q, se esperaba %q", config.Mode, "download")
	}

	if config.DownloadTarget != "raw" {
		t.Errorf("DownloadTarget = %q, se esperaba %q", config.DownloadTarget, "raw")
	}
}

func TestParseConfigDownloadAllAndNoDownload(t *testing.T) {
	config, err := parseConfig([]string{"-download=all", "-no-download"})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if config.DownloadTarget != "all" {
		t.Errorf("DownloadTarget = %q, se esperaba %q", config.DownloadTarget, "all")
	}

	if !config.NoDownload {
		t.Error("NoDownload = false, se esperaba true")
	}
}

func TestParseConfigInvalidDownloadTarget(t *testing.T) {
	_, err := parseConfig([]string{"-download=bogus"})

	if err == nil {
		t.Fatal("se esperaba un error para -download inválido")
	}
}
