package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Estas pruebas cubren únicamente las ramas de ensureCleanDatasetAvailable
// / ensureRawDatasetAvailable / ensureNamedDatasetAvailable que NO
// requieren red: archivo ya presente, ruta personalizada ausente, y
// -no-download activo. La rama de descarga real la ejerce D1
// (ensureDataset) con un servidor httptest; aquí no se dispara ninguna
// petición HTTP real.

func TestEnsureNamedDatasetAvailable_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.csv")

	if err := os.WriteFile(path, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo preparar el archivo: %v", err)
	}

	if err := ensureNamedDatasetAvailable(path, "clean", "el dataset de prueba", false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestEnsureNamedDatasetAvailable_MissingWithNoDownload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.csv")

	err := ensureNamedDatasetAvailable(path, "clean", "el dataset de prueba", true)

	if err == nil {
		t.Fatal("se esperaba un error cuando el archivo falta y -no-download está activo")
	}
}

func TestEnsureCleanDatasetAvailable_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.csv")

	if err := os.WriteFile(path, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo preparar el archivo: %v", err)
	}

	config := defaultConfig()
	config.DatasetPath = path

	if err := ensureCleanDatasetAvailable(config); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestEnsureCleanDatasetAvailable_CustomPathMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-existe.csv")

	config := defaultConfig()
	config.DatasetPath = path

	err := ensureCleanDatasetAvailable(config)

	if err == nil {
		t.Fatal("se esperaba un error: ruta -dataset personalizada ausente no debe auto-descargarse")
	}
}

func TestEnsureCleanDatasetAvailable_DefaultPathMissingWithNoDownload(t *testing.T) {
	config := defaultConfig()
	config.NoDownload = true
	// DatasetPath queda en su default (cleanDatasetPath); si por
	// alguna razón ya existiera en el repo del desarrollador, esta
	// prueba se saltea para no depender del estado del filesystem real.
	if _, statErr := os.Stat(config.DatasetPath); statErr == nil {
		t.Skip("el dataset limpio real ya existe en este checkout; se omite la prueba de -no-download")
	}

	err := ensureCleanDatasetAvailable(config)

	if err == nil {
		t.Fatal("se esperaba un error accionable cuando falta el dataset y -no-download está activo")
	}
}

func TestDownloadTargets(t *testing.T) {
	tests := []struct {
		target string
		want   []string
	}{
		{"clean", []string{"clean"}},
		{"raw", []string{"raw"}},
		{"all", []string{"clean", "raw"}},
		{"", []string{"clean"}},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := downloadTargets(tt.target)

			if len(got) != len(tt.want) {
				t.Fatalf("downloadTargets(%q) = %v, se esperaba %v", tt.target, got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("downloadTargets(%q) = %v, se esperaba %v", tt.target, got, tt.want)
				}
			}
		})
	}
}

func TestRunPipeline_MissingDatasetReturnsError(t *testing.T) {
	config := defaultConfig()
	config.DatasetPath = filepath.Join(t.TempDir(), "no-existe.csv")
	config.NoDownload = true

	if err := runPipeline(config); err == nil {
		t.Fatal("se esperaba un error cuando el dataset no existe")
	}
}
