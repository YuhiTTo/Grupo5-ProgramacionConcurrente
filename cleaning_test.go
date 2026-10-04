package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCleaning_MissingInputReturnsError(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "clean.csv")

	err := runCleaning(filepath.Join(dir, "no-existe.csv"), output)

	if err == nil {
		t.Fatal("se esperaba un error cuando el dataset de entrada no existe")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Errorf("no debería existir el archivo de salida, stat err = %v", statErr)
	}
}
